package app

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/SUSE/telemetry/pkg/restapi"
	"github.com/SUSE/telemetry/pkg/types"
)

// RegisterClient is responsible for handling client registrations
func (a *App) RegisterClient(ar *AppRequest) {
	log.Printf("INF: %s %s Processing", ar.R.Method, ar.R.URL)
	// retrieve the request body
	reqBody, err := io.ReadAll(ar.R.Body)
	if err != nil {
		ar.ErrorResponse(http.StatusBadRequest, err.Error())
		return
	}
	log.Printf("INF: %s %s reqBody: %s", ar.R.Method, ar.R.URL, reqBody)

	// unmarshal the request body to the request struct
	var crReq restapi.ClientRegistrationRequest
	err = json.Unmarshal(reqBody, &crReq)
	if err != nil {
		ar.ErrorResponse(http.StatusBadRequest, err.Error())
		return
	}
	if crReq.ClientInstanceId == "" {
		ar.ErrorResponse(http.StatusBadRequest, "no ClientInstanceId value provided")
		return
	}
	log.Printf("INF: %s %s crReq: %s", ar.R.Method, ar.R.URL, &crReq)

	// register the client
	client := ClientsRow{ClientInstanceId: crReq.ClientInstanceId}
	if client.Exists(a.TelemetryDB.Conn) {
		ar.ErrorResponse(http.StatusConflict, "specified clientInstanceId already exists")
		return
	}

	client.RegistrationDate = types.Now().String()
	client.AuthToken = "sometoken"
	err = client.Insert(a.TelemetryDB.Conn)
	if err != nil {
		ar.ErrorResponse(http.StatusInternalServerError, "failed to register new client")
		return
	}

	// initialise a client registration response
	crResp := restapi.ClientRegistrationResponse{
		ClientId:  client.Id,
		AuthToken: client.AuthToken,
		IssueDate: client.RegistrationDate,
	}
	log.Printf("INF: %s %s crResp: %s", ar.R.Method, ar.R.URL, &crResp)

	// respond success with the client registration response
	ar.JsonResponse(http.StatusOK, crResp)
}
