package app

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/SUSE/telemetry/pkg/restapi"
	"github.com/SUSE/telemetry/pkg/types"
	"github.com/go-playground/validator/v10"
)

func (a *App) ReportTelemetry(ar *AppRequest) {
	ar.Log.Info("Processing")

	token := ar.GetAuthToken()
	if err := a.AuthManager.VerifyToken(token); err != nil {
		// TODO: Set WWW-Authenticate header appropriately, per
		// https://www.rfc-editor.org/rfc/rfc9110.html#name-www-authenticate
		ar.ErrorResponse(http.StatusUnauthorized, "Missing or Invalid Authorization")
	}
	ar.Log.Debug("Authorized", slog.String("token", token))

	// TODO: Report has a valid token. It is from a registered client?

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
