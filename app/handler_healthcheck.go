package app

import (
	"net/http"
)

func (a *App) HealthCheck(ar *AppRequest) {
	ar.Log.Debug("Processing")
	// respond success
	ar.JsonResponse(http.StatusOK, `{"alive": true}`)
}
