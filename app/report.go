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
	log.Printf("INF: %s %s reqBody: %q", ar.R.Method, ar.R.URL, reqBody)

	// unmarshal the request body to the request struct
	//var trReq telemetrylib.TelemetryReport
	var trReq restapi.TelemetryReportRequest
	err = json.Unmarshal(reqBody, &trReq)
	if err != nil {
		ar.ErrorResponse(http.StatusBadRequest, err.Error())
		return
	}
	log.Printf("INF: %s %s trReq: %q", ar.R.Method, ar.R.URL, &trReq)

	// initialise a client registration response
	trResp := restapi.NewTelemetryReportResponse(0, types.Now())
	log.Printf("INF: %s %s trResp: %q", ar.R.Method, ar.R.URL, trResp)

	// respond success with the client registration response
	ar.JsonResponse(http.StatusOK, trResp)
}
