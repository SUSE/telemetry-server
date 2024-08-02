package app

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/SUSE/telemetry/pkg/restapi"
	"github.com/SUSE/telemetry/pkg/types"
	"github.com/go-playground/validator/v10"
)

func (a *App) ReportTelemetry(ar *AppRequest) {
	ar.Log.Info("Processing")

	// retrieve required headers
	hdrClientId := ar.GetClientId()
	token := ar.GetAuthToken()

	// missing clientId or token suggests client needs to register
	if (hdrClientId == "") || (token == "") {
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

	// verify that the provided client id is a valid number
	clientId, err := strconv.ParseInt(hdrClientId, 0, 64)
	if err != nil {
		// client needs to register
		ar.SetWwwAuthRegister()
		ar.ErrorResponse(http.StatusUnauthorized, "Invalid Client Id")
		return
	}

	// verify that the request is from a registered client
	client := new(ClientsRow)
	client.InitClientId(clientId)
	if err = client.SetupDB(&a.OperationalDB); err != nil {
		ar.Log.Error("clientsRow.SetupDB() failed", slog.String("error", err.Error()))
		ar.ErrorResponse(http.StatusInternalServerError, "failed to access DB")
		return
	}

	if !client.Exists() {
		// client needs to register
		ar.SetWwwAuthRegister()
		ar.ErrorResponse(http.StatusUnauthorized, "Invalid Client Id")
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
		"Client Authorizated",
		slog.Int64("clientId", clientId),
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

	validate := validator.New(validator.WithRequiredStructEnabled())
	err = validate.Struct(&trReq)
	if err != nil {
		ar.ErrorResponse(http.StatusBadRequest, err.Error())
		return
	}

	ar.Log.Debug("Unmarshaled", slog.Any("trReq", &trReq))

	// save the report into the staging db
	a.StageTelemetryReport(reqBody, &trReq.TelemetryReport.Header)

	// process pending reports
	a.ProcessStagedReports()

	// initialise a telemetry report response
	trResp := restapi.NewTelemetryReportResponse(0, types.Now())
	ar.Log.Debug("Response", slog.Any("trResp", trResp))

	// respond success with the telemetry report response
	ar.JsonResponse(http.StatusOK, trResp)
}
