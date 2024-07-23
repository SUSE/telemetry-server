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

	// verify that a valid authtoken has been provided
	token := ar.GetAuthToken()
	if err := a.AuthManager.VerifyToken(token); err != nil {
		// TODO: Set WWW-Authenticate header appropriately, per
		// https://www.rfc-editor.org/rfc/rfc9110.html#name-www-authenticate
		ar.ErrorResponse(http.StatusUnauthorized, "Missing or Invalid Authorization")
	}

	ar.Log.Debug(
		"Bearer Authorization Valid",
		slog.String("token", token),
	)

	// verify that the provided client id is a valid number
	hdrClientId := ar.GetClientId()
	clientId, err := strconv.ParseInt(hdrClientId, 0, 64)
	if err != nil {
		// TODO: Set WWW-Authenticate header appropriately, per
		// https://www.rfc-editor.org/rfc/rfc9110.html#name-www-authenticate
		ar.ErrorResponse(http.StatusUnauthorized, "Invalid Client Id")
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
		// TODO: Set WWW-Authenticate header appropriately, per
		// https://www.rfc-editor.org/rfc/rfc9110.html#name-www-authenticate
		ar.ErrorResponse(http.StatusUnauthorized, "Invalid Client Id")
	}

	// verify that the provided authtoken matches last authtoken issued to the client
	if client.AuthToken != token {
		// TODO: Set WWW-Authenticate header appropriately, per
		// https://www.rfc-editor.org/rfc/rfc9110.html#name-www-authenticate
		ar.ErrorResponse(http.StatusUnauthorized, "Invalid Authorization")
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
