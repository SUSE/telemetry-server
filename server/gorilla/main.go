package main

import (
	"flag"
	"fmt"
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

// options is a struct of the options
type options struct {
	config string
}

func (o options) String() string {
	return fmt.Sprintf("Options: config=%q", o.config)
}

var opts options

func main() {
	log.Printf("Preparing to start gorilla/mux based server with options: %s", opts)

	router := mux.NewRouter()

	cfg := app.NewConfig(opts.config)
	if err := cfg.Load(); err != nil {
		log.Fatal(err)
	}

	log.Printf("Config: %v", cfg)

	a := app.NewApp(cfg, router)

	wrapper := newRouterWrapper(router, a)

	router.HandleFunc("/telemetry/register", wrapper.registerClient).Methods("POST")
	router.HandleFunc("/telemetry/report", wrapper.reportTelemetry).Methods("POST")

	a.Initialize()
	a.Run()
}

func init() {
	flag.StringVar(&opts.config, "config", app.DEFAULT_CONFIG, "Path to config file to use")
	flag.Parse()
}
