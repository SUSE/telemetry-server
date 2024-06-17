package main

// Telemetry Server application using using the gorilla/mux routing framework.

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/SUSE/telemetry-server/app"
	"github.com/gorilla/mux"
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

	parseCommandLineFlags()

	log.Printf("INF: Preparing to start gorilla/mux based server with options: %s", opts)

	cfg := app.NewConfig(opts.config)
	if err := cfg.Load(); err != nil {
		log.Fatal(err)
	}

	log.Printf("INF: Config: %v", cfg)

	a, _ := InitializeApp(cfg)

	a.Run()
}

func parseCommandLineFlags() {
	flag.StringVar(&opts.config, "config", app.DEFAULT_CONFIG, "Path to config file to use")
	flag.Parse()
}

func SetupRouterWrapper(router *mux.Router, app *app.App) {
	wrapper := newRouterWrapper(router, app)

	router.HandleFunc("/telemetry/register", wrapper.registerClient).Methods("POST")
	router.HandleFunc("/telemetry/report", wrapper.reportTelemetry).Methods("POST")

}

func InitializeApp(cfg *app.Config) (a *app.App, router *mux.Router) {
	router = mux.NewRouter()

	a = app.NewApp(cfg, router)

	SetupRouterWrapper(router, a)

	a.Initialize()

	return
}
