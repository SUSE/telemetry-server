package main

import (
	"log"
	"net/http"

	"github.com/SUSE/telemetry-server/app"
	"github.com/gorilla/mux"
)

const (
	DB_DRIVER = "sqlite3"
	DB_URI    = "../../../server.db"
	HOST      = "localhost"
	PORT      = 9999
)

type routerWrapper struct {
	router *mux.Router
	app    *app.App
}

func newRouterWrapper(router *mux.Router, app *app.App) *routerWrapper {
	return &routerWrapper{router: router, app: app}
}

func (rw *routerWrapper) registerClient(w http.ResponseWriter, r *http.Request) {
	req := &app.AppRequest{W: w, R: r, Vars: mux.Vars(r)}

	rw.app.RegisterClient(req)
}

func (rw *routerWrapper) reportTelemetry(w http.ResponseWriter, r *http.Request) {
	req := &app.AppRequest{W: w, R: r, Vars: mux.Vars(r)}

	rw.app.ReportTelemetry(req)
}

func main() {
	log.Println("Preparing to start gorilla/mux based server")

	router := mux.NewRouter()

	a := app.NewApp(DB_DRIVER, DB_URI, HOST, PORT, router)

	wrapper := newRouterWrapper(router, a)

	router.HandleFunc("/telemetry/register", wrapper.registerClient).Methods("POST")
	router.HandleFunc("/telemetry/report", wrapper.reportTelemetry).Methods("POST")

	a.Initialize()
	a.Run()
}
