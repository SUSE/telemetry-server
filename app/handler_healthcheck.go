package app

import (
	"net/http"
)

func (a *App) HealthCheck(ar *AppRequest) {
	ar.Log.Info("Processing")
	ar.SetHeader("Content-Type", "application/json")
	// respond success
	ar.JsonResponse(http.StatusOK, `{"alive": true}`)

}
