package app

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/SUSE/telemetry/pkg/restapi"
)

// RegisterClient is responsible for handling client registrations
func (a *App) AuthenticateClient(ar *AppRequest) {
	ar.Log.Info("Processing", ar.R.Method, ar.R.URL)

	// retrieve the request body
	reqBody, err := io.ReadAll(ar.R.Body)
	if err != nil {
		ar.ErrorResponse(http.StatusBadRequest, err.Error())
		return
	}
	ar.Log.Debug("Extracted", slog.Any("body", reqBody))

	// unmarshal the request body to the request struct
	var caReq restapi.ClientAuthenticationRequest
	err = json.Unmarshal(reqBody, &caReq)
	if err != nil {
		ar.ErrorResponse(http.StatusBadRequest, err.Error())
		return
	}
	if caReq.RegistrationId <= 0 {
		ar.SetWwwAuthRegister()
		ar.ErrorResponse(http.StatusUnauthorized, "Invalid registrationId value provided")
		return
	}
	ar.Log.Debug("Unmarshaled", slog.Any("caReq", &caReq))

	// register the client
	client := new(ClientsRow)
	if err = client.SetupDB(&a.OperationalDB); err != nil {
		ar.Log.Error("clientsRow.SetupDB() failed", slog.String("error", err.Error()))
		ar.ErrorResponse(http.StatusInternalServerError, "failed to access DB")
		return
	}

	// confirm that the client has been registered
	client.InitAuthentication(&caReq)
	if !client.Exists() {
		// client needs to register
		ar.SetWwwAuthRegister()
		ar.ErrorResponse(http.StatusUnauthorized, "Client not registered")
		return
	}

	// confirm that the provided registration hash matches the registered one
	regHash := client.GetClientRegistration().Hash(caReq.RegHash.Method)
	if !regHash.Match(&caReq.RegHash) {
		ar.Log.Error(
			"Registration hash mismatch",
			slog.String("Req Hash", caReq.RegHash.String()),
			slog.String("DB Hash", regHash.String()),
		)
		// client needs to re-register
		ar.SetWwwAuthRegister()
		ar.ErrorResponse(http.StatusUnauthorized, "Registration mismatch")
		return
	}

	// TODO: return existing token if remaining duration is >= half of
	// a new tokens duration

	// create a new token for the client
	client.AuthToken, err = a.AuthManager.CreateToken()
	if err != nil {
		ar.ErrorResponse(http.StatusInternalServerError, "failed to create new authtoken for client")
	}

	// update token stored in the DB
	err = client.Update()
	if err != nil {
		ar.ErrorResponse(http.StatusInternalServerError, "failed to client authtoken")
		return
	}

	// initialise a client registration response
	caResp := restapi.ClientAuthenticationResponse{
		RegistrationId:   client.Id,
		AuthToken:        client.AuthToken,
		RegistrationDate: client.RegistrationDate,
	}
	ar.Log.Debug("Response", slog.Any("caResp", caResp))

	// respond success with the client registration response
	ar.JsonResponse(http.StatusOK, caResp)
}
