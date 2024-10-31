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
	if string(crReq.ClientInstanceId) == "" {
		ar.ErrorResponse(http.StatusBadRequest, "no ClientInstanceId value provided")
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
	if client.InstIdExists() {
		ar.ErrorResponse(http.StatusConflict, "specified clientInstanceId already exists")
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
		ClientId:  client.Id,
		AuthToken: client.AuthToken,
		RegistrationDate: client.RegistrationDate,
	}
	ar.Log.Debug("Response", slog.Any("crResp", crResp))

	// respond success with the client registration response
	ar.JsonResponse(http.StatusOK, crResp)
}
