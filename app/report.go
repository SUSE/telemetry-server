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
	// retrieve the request body
	reqBody, err := io.ReadAll(ar.R.Body)
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

	//Save the report into the staging db
	a.StoreTelemetryReport(reqBody, trReq.TelemetryReport.Header.ReportId)

	a.ProcessBundles(&trReq.TelemetryReport)

	// initialise a telemetry report response
	trResp := restapi.NewTelemetryReportResponse(0, types.Now())
	log.Printf("INF: %s %s trResp: %s", ar.R.Method, ar.R.URL, trResp)

	// respond success with the telemetry report response
	ar.JsonResponse(http.StatusOK, trResp)
}
