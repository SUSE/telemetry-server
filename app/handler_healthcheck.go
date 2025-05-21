package app

import (
	"net/http"
)

func (a *App) HealthCheck(ar *AppRequest) {
	ar.Log.Debug("Processing")
	// respond success
	ar.JsonResponse(http.StatusOK, `{"alive": true}`)
}

func (a *App) LiveCheck(ar *AppRequest) {
	ar.Log.Debug("Checking liveness probe")
	err := a.TelemetryDB.dbMgr.Ping()
	if err != nil {
		ar.Log.Error("Failed liveness probe")
		ar.JsonResponse(http.StatusInternalServerError, `{"live": false}`)
		return
	}

	err = a.OperationalDB.dbMgr.Ping()
	if err != nil {
		ar.Log.Error("Failed liveness probe")
		ar.JsonResponse(http.StatusInternalServerError, `{"live": false}`)
		return
	}

	ar.JsonResponse(http.StatusOK, `{"live": true}`)
}
