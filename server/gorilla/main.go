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

func (rw *routerWrapper) registerClient(w http.ResponseWriter, r *http.Request) {
	req := app.AppRequest{W: w, R: r, Vars: mux.Vars(r)}

	rw.app.RegisterClient(&req)
}

func main() {
	log.Println("Starting gorilla/mux based server")
	app.Hello()

	router := mux.NewRouter()

	a := app.App{}
	a.Setup(DB_DRIVER, DB_URI, HOST, PORT, router)

	wrapper := routerWrapper{router: router, app: &a}

	router.HandleFunc("/register", wrapper.registerClient).Methods("POST")

	a.Initialize()
	a.Run()
}
