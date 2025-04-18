package app

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/SUSE/telemetry/pkg/restapi"
	"github.com/SUSE/telemetry/pkg/types"
)

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

	// verify that the request is from a registered client
	client := new(ClientsRow)
	if err = client.SetupDB(&a.OperationalDB); err != nil {
		ar.Log.Error("clientsRow.SetupDB() failed", slog.String("error", err.Error()))
		ar.ErrorResponse(http.StatusInternalServerError, "failed to access DB")
		return
	}

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

	// save the report into the operational db
	err = a.StageTelemetryReport(reqBody, &trReq.TelemetryReport.Header)
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

	// initialise a telemetry report response
	trResp := restapi.NewTelemetryReportResponse(0, types.Now())
	ar.Log.Debug("Response", slog.Any("trResp", trResp))

	// respond success with the telemetry report response
	ar.JsonResponse(http.StatusOK, trResp)
}
