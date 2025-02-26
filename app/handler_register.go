package app

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/SUSE/telemetry/pkg/restapi"
	"github.com/SUSE/telemetry/pkg/types"
)

// RegisterClient is responsible for handling client registrations
func (a *App) RegisterClient(ar *AppRequest) {
	ar.Log.Info("Processing", ar.R.Method, ar.R.URL)

	// retrieve the request body
	reqBody, err := io.ReadAll(ar.R.Body)
	if err != nil {
		ar.ErrorResponse(http.StatusBadRequest, err.Error())
		return
	}
	ar.Log.Debug("Extracted", slog.Any("body", reqBody))

	// unmarshal the request body to the request struct
	var crReq restapi.ClientRegistrationRequest
	err = json.Unmarshal(reqBody, &crReq)
	if err != nil {
		ar.ErrorResponse(http.StatusBadRequest, err.Error())
		return
	}
	// verify that clientId and timestamp are specified in registration
	if string(crReq.ClientRegistration.ClientId) == "" {
		ar.ErrorResponse(http.StatusBadRequest, "missing registration clientId")
		return
	}
	if string(crReq.ClientRegistration.Timestamp) == "" {
		ar.ErrorResponse(http.StatusBadRequest, "missing registration timestamp")
		return
	}
	ar.Log.Debug("Unmarshaled", slog.Any("crReq", &crReq))

	// register the client
	client := new(ClientsRow)
	if err = client.SetupDB(&a.OperationalDB); err != nil {
		ar.Log.Error("clientsRow.SetupDB() failed", slog.String("error", err.Error()))
		ar.ErrorResponse(http.StatusInternalServerError, "failed to access DB")
		return
	}

	client.InitRegistration(&crReq)
	// check if the supplied registration already exists, e.g. cloned system
	if client.RegistrationExists() {
		ar.ErrorResponse(http.StatusConflict, "specified registration already exists")
		return
	}

	// check if the supplied registration's clientID already exists, e.g. a new
	// client generated the same UUID value that an existing client is using
	if client.ClientIdExists() {
		ar.ErrorResponse(http.StatusConflict, "specified registration clientId already exists")
		return
	}

	client.AuthToken, err = a.AuthManager.CreateToken()
	if err != nil {
		ar.ErrorResponse(http.StatusInternalServerError, "failed to create authtoken for client")
	}

	client.RegistrationDate = types.Now().String()
	err = client.Insert()
	if err != nil {
		ar.ErrorResponse(http.StatusInternalServerError, "failed to register new client")
		return
	}

	// initialise a client registration response
	crResp := restapi.ClientRegistrationResponse{
		RegistrationId:   client.Id,
		AuthToken:        client.AuthToken,
		RegistrationDate: client.RegistrationDate,
	}
	ar.Log.Debug("Response", slog.Any("crResp", crResp))

	// respond success with the client registration response
	ar.JsonResponse(http.StatusOK, crResp)
}
