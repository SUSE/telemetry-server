package main

// Telemetry Server application using using the gorilla/mux routing framework.

import (
	"encoding/json"
	"flag"
	"log/slog"
	"net/http"

	"github.com/SUSE/telemetry-server/app"
	"github.com/SUSE/telemetry-server/app/config"
	"github.com/SUSE/telemetry/pkg/logging"
	"github.com/gorilla/mux"
)

type routerWrapper struct {
	router *mux.Router
	app    *app.App
}

func newRouterWrapper(router *mux.Router, app *app.App) *routerWrapper {
	return &routerWrapper{router: router, app: app}
}

func (rw *routerWrapper) healthCheck(w http.ResponseWriter, r *http.Request) {
	rw.app.HealthCheck(app.QuietAppRequest(w, r, mux.Vars(r)))
}

func (rw *routerWrapper) liveCheck(w http.ResponseWriter, r *http.Request) {
	rw.app.LiveCheck(app.QuietAppRequest(w, r, mux.Vars(r)))
}

func (rw *routerWrapper) getVersion(w http.ResponseWriter, r *http.Request) {
	rw.app.Version(app.QuietAppRequest(w, r, mux.Vars(r)))
}

// options is a struct of the options
type options struct {
	Config string `json:"config"`
	Debug  bool   `json:"debug"`
}

func (o options) String() string {
	str, _ := json.Marshal(o)
	return string(str)
}

var opts options

func main() {

	parseCommandLineFlags()

	// setup basic logging that will later be superseded by the settings
	// specified in the config file, providing some level of consistency
	// for log messages generated before and after the config is loaded.
	logging.SetupBasicLogging(opts.Debug)

	slog.Debug("Preparing to start gorilla/mux based server", slog.Any("options", opts))

	cfg := config.NewConfig(opts.Config)
	if err := cfg.Load(); err != nil {
		slog.Error("config load failed", slog.String("config", opts.Config), slog.String("error", err.Error()))
		panic(err)
	}

	slog.Debug("Loaded config", slog.String("path", opts.Config), slog.Any("config", cfg))

	a, _ := InitializeApp(cfg, opts.Debug)

	a.Run()
}

func parseCommandLineFlags() {
	// define available flags
	flag.StringVar(&opts.Config, "config", config.DEFAULT_CONFIG, "Path to `config` file to use")
	flag.BoolVar(&opts.Debug, "debug", false, "Enables debug level messages")

	// parse supplied command line flags
	flag.Parse()
}

func SetupRouterWrapper(router *mux.Router, app *app.App) {
	wrapper := newRouterWrapper(router, app)

	router.HandleFunc("/healthz", wrapper.healthCheck).Methods("GET", "HEAD")
	router.HandleFunc("/live", wrapper.liveCheck).Methods("GET", "HEAD")
	router.HandleFunc("/version", wrapper.getVersion).Methods("GET", "HEAD")
}

func InitializeApp(cfg *config.Config, debug bool) (a *app.App, router *mux.Router) {
	router = mux.NewRouter()

	a = app.NewApp("Admin", cfg, router, debug)

	SetupRouterWrapper(router, a)

	if err := a.Initialize(); err != nil {
		panic(err)
	}

	return
}
