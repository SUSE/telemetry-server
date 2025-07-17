package app

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/SUSE/telemetry-server/app/database"
	telemetrylib "github.com/SUSE/telemetry/pkg/lib"
	"github.com/SUSE/telemetry/pkg/restapi"
	"github.com/SUSE/telemetry/pkg/types"
)

// Telemetry reports can be processed immediately or
// staged for later processing. This variable is used
// to control the default mode of operation, which is
// currently disabled by default.
var stageTelemetryReports bool = false

// Enable telemetry report staging by default
func EnableTelemetryReportStaging() {
	stageTelemetryReports = true
}

// Disable telemetry report staging by default
func DisableTelemetryReportStaging() {
	stageTelemetryReports = false
}

func (a *App) ReportTelemetry(ar *AppRequest) {
	ar.Log.Info("Processing")

	// retrieve required headers
	hdrRegistrationId := ar.GetRegistrationId()
	token := ar.GetAuthToken()

	// missing registrationId or token suggests client needs to register
	if (hdrRegistrationId == "") || (token == "") {
		// client needs to register
		ar.SetWwwAuthRegister()
		ar.ErrorResponse(http.StatusUnauthorized, "Client registration required")
		return
	}

	// verify that a valid authtoken has been provided
	if err := a.AuthManager.VerifyToken(token); err != nil {
		// client needs to re-authenticate
		ar.SetWwwAuthReauth()
		ar.ErrorResponse(http.StatusUnauthorized, "Invalid Authorization")
		return
	}

	ar.Log.Debug(
		"Bearer Authorization Valid",
		slog.String("token", token),
	)

	// verify that the provided registration id is a valid number
	registrationId, err := strconv.ParseInt(hdrRegistrationId, 0, 64)
	if err != nil {
		// client needs to register
		ar.SetWwwAuthRegister()
		ar.ErrorResponse(http.StatusUnauthorized, "Invalid Registration Id")
		return
	}

	//
	// create an operationalDb transaction
	//
	odbTx, err := a.OperationalDB.StartTx()
	if err != nil {
		ar.ErrorResponse(http.StatusInternalServerError, err.Error())
		return
	}

	// defer a rollback of the operationalDb transaction
	defer func() {
		a.OperationalDB.RollbackTx(odbTx, "AuthenticateClient")
	}()

	// verify that the request is from a registered client
	client := new(database.ClientsRow)
	client.SetupDB(a.OperationalDB, odbTx)

	// check that the registration exists
	client.InitRegistrationId(registrationId)
	if !client.Exists() {
		// client needs to register
		ar.SetWwwAuthRegister()
		ar.ErrorResponse(http.StatusUnauthorized, "Invalid Registration Id")
		return
	}

	// verify that the provided authtoken matches last authtoken issued to the client
	if client.AuthToken != token {
		// TODO detect cloned clients, where InstID matches ClientId, but authtoken will
		// will be stale
		// client needs to re-authenticate
		ar.SetWwwAuthReauth()
		ar.ErrorResponse(http.StatusUnauthorized, "Invalid Authorization")
		return
	}

	// commit the transaction
	if err = a.OperationalDB.CommitTx(odbTx); err != nil {
		ar.ErrorResponse(http.StatusInternalServerError, "failed to commit client report")
		return
	}

	ar.Log.Debug(
		"Client Authorized",
		slog.Int64("registrationId", registrationId),
	)

	// handle payload compression
	reader, err := ar.getReader()
	if err != nil {
		ar.ErrorResponse(http.StatusInternalServerError, "Failed to decompress request body")
		return
	}
	defer reader.Close()

	// retrieve the request body
	reqBody, err := io.ReadAll(reader)
	if err != nil {
		ar.ErrorResponse(http.StatusBadRequest, err.Error())
		return
	}
	ar.Log.Debug("Extracted", slog.Any("body", reqBody))

	// unmarshal the request body to the request struct
	var trReq restapi.TelemetryReportRequest
	err = json.Unmarshal(reqBody, &trReq)
	if err != nil {
		ar.ErrorResponse(http.StatusBadRequest, err.Error())
		return
	}

	ar.Log.Debug("Unmarshaled", slog.Any("trReq", &trReq))

	// validate structure
	err = trReq.TelemetryReport.Validate()
	if err != nil {
		ar.ErrorResponse(http.StatusBadRequest, err.Error())
		return
	}
	ar.Log.Debug("Structure validated")

	// verify checksums
	err = trReq.TelemetryReport.VerifyChecksum()
	if err != nil {
		ar.ErrorResponse(http.StatusBadRequest, err.Error())
		return
	}
	ar.Log.Debug("Checksums verified")

	// telemetry reports can be either handled inline or staged
	// for later processing
	var stagingId int64 = 0
	if !stageTelemetryReports {
		err = a.ProcessTelemetryReport(&trReq.TelemetryReport)
		if err != nil {
			ar.ErrorResponse(
				http.StatusBadRequest,
				fmt.Errorf("report processing failed: %w", err).Error(),
			)
			return
		}
	} else {
		// save the report into the operational db, obtaining the staging
		// db's entry id if successful
		stagingId, err = a.StageTelemetryReport(
			reqBody,
			&trReq.TelemetryReport.Header,
		)
		if err != nil {
			ar.ErrorResponse(http.StatusBadRequest, err.Error())
			return
		}

		// process pending reports
		err = a.ProcessStagedReports()
		if err != nil {
			// err is a joined slice of multiple errors for which err.Error()
			// will return a multiline string, one error per line, so prepend
			// a summary error and fail request with combined error
			ar.ErrorResponse(
				http.StatusBadRequest,
				fmt.Errorf("staged report processing failed:\n%w", err).Error(),
			)
			return
		}
	}

	// initialise a telemetry report response, stagingId will be 0 if we
	// processed the report inline, otherwise it will be the id of the
	// entry in the staging table, which will be processed at a later time.
	trResp := restapi.NewTelemetryReportResponse(stagingId, types.Now())
	ar.Log.Debug("Response", slog.Any("trResp", trResp))

	// respond success with the telemetry report response
	ar.JsonResponse(http.StatusOK, trResp)
}

func (a *App) ProcessTelemetryReport(report *telemetrylib.TelemetryReport) error {
	numBundles := len(report.TelemetryBundles)
	var totalItems int

	slog.Info(
		"Processing telemetry report",
		slog.String("reportId", report.Header.ReportId),
		slog.String("reportClientId", report.Header.ReportClientId),
		slog.Int("numBundles", numBundles),
	)

	// process available bundles, extracting the data items and
	// storing them in the telemetry DB
	for _, bundle := range report.TelemetryBundles {
		numItems := len(bundle.TelemetryDataItems)

		slog.Debug(
			"Processing telemetry bundle",
			slog.String("bundleId", bundle.Header.BundleId),
			slog.String("bundleClientId", bundle.Header.BundleClientId),
			slog.Int("numItems", numItems),
		)

		// for each data item in the bundle, process it
		for _, item := range bundle.TelemetryDataItems {
			slog.Debug(
				"Processing telemetry data item",
				slog.String("telemetryId", item.Header.TelemetryId),
				slog.String("telemetryType", item.Header.TelemetryType),
			)

			if err := a.StoreTelemetry(&item, &bundle.Header); err != nil {
				slog.Error(
					"Failed to store telemetry data item",
					slog.String("telemetryId", item.Header.TelemetryId),
					slog.String("telemetryType", item.Header.TelemetryType),
					slog.String("bundleId", bundle.Header.BundleId),
					slog.String("bundleClientId", bundle.Header.BundleClientId),
					slog.String("error", err.Error()),
				)
				return fmt.Errorf(
					"failed to store telemetry item %q from bundle %q in report %q: %w",
					item.Header.TelemetryId,
					bundle.Header.BundleId,
					report.Header.ReportId,
					err,
				)
			}
		}

		// increment the number of items processed
		totalItems += numItems
	}

	slog.Info(
		"Successfully processed telemetry report",
		slog.String("reportId", report.Header.ReportId),
		slog.String("reportClientId", report.Header.ReportClientId),
		slog.Int("numBundles", numBundles),
		slog.Int("totalItems", totalItems),
	)

	return nil
}
