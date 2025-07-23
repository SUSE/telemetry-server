package app

import (
	"log/slog"
	"net/http"
)

func (a *App) Version(ar *AppRequest) {
	ar.Log.Debug("Processing")

	// respond with the version
	payload := struct {
		Version string `json:"version"`
	}{
		Version: GetVersion(),
	}

	ar.Log.Warn("Version payload", slog.String("version", payload.Version))

	ar.JsonResponse(http.StatusOK, payload)
}
