package app

import (
	"encoding/json"

	"io"
	"log"
	"net/http"

	"github.com/SUSE/telemetry/pkg/restapi"
	"github.com/SUSE/telemetry/pkg/types"
	"github.com/go-playground/validator/v10"
)

func (a *App) ReportTelemetry(ar *AppRequest) {
	log.Printf("INF: %s %s Processing", ar.R.Method, ar.R.URL)

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
	log.Printf("INF: %s %s reqBody: %s", ar.R.Method, ar.R.URL, reqBody)

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

	log.Printf("INF: %s %s trReq: %s", ar.R.Method, ar.R.URL, &trReq)

	// save the report into the staging db
	a.StageTelemetryReport(reqBody, &trReq.TelemetryReport.Header)

	// process pending reports
	a.ProcessStagedReports()

	// initialise a telemetry report response
	trResp := restapi.NewTelemetryReportResponse(0, types.Now())
	log.Printf("INF: %s %s trResp: %s", ar.R.Method, ar.R.URL, trResp)

	// respond success with the telemetry report response
	ar.JsonResponse(http.StatusOK, trResp)
}
