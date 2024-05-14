package app

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/SUSE/telemetry/pkg/restapi"
	"github.com/SUSE/telemetry/pkg/types"
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
	log.Printf("INF: %s %s trReq: %s", ar.R.Method, ar.R.URL, &trReq)

	// store received telemetry report in reports datastore
	a.Extractor.AddReport(&trReq.TelemetryReport)

	// trigger telemetry processing
	a.ProcessTelemetry()

	// initialise a telemetry report response
	trResp := restapi.NewTelemetryReportResponse(0, types.Now())
	log.Printf("INF: %s %s trResp: %s", ar.R.Method, ar.R.URL, trResp)

	// respond success with the telemetry report response
	ar.JsonResponse(http.StatusOK, trResp)
}
