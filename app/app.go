package app

import (
	"context"
	_ "embed"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/SUSE/telemetry-server/app/config"
	"github.com/SUSE/telemetry-server/app/database"
	"github.com/SUSE/telemetry-server/app/database/operationaldb"
	"github.com/SUSE/telemetry-server/app/database/telemetrydb"
	"github.com/SUSE/telemetry/pkg/logging"
	_ "github.com/mattn/go-sqlite3"
)

// Telemetry Service Library Version
//
//go:embed VERSION
var tslVersion string

func GetVersion() string {
	slog.Warn("GetVersion", slog.String("tslVersion", tslVersion))
	return strings.TrimSpace(tslVersion)
}

// App is a struct tracking the resources associated with the application
type App struct {
	// public
	Name          string
	Config        *config.Config
	TelemetryDB   *database.AppDb
	OperationalDB *database.AppDb
	Address       ServerAddress
	Handler       http.Handler
	LogManager    *logging.LogManager
	AuthManager   *AuthManager

	// private
	server    *http.Server
	signals   chan os.Signal
	debugMode bool
}

func NewApp(name string, cfg *config.Config, handler http.Handler, debugMode bool) *App {
	var err error

	a := new(App)

	a.Name = name
	a.Config = cfg
	a.Handler = handler
	a.debugMode = debugMode
	a.signals = make(chan os.Signal, 1)

	// setup logging first so remaining setup logs with config settings
	if err := a.SetupLogging(); err != nil {
		panic(err)
	}

	// setup operational database
	a.OperationalDB, err = operationaldb.New(cfg)
	if err != nil {
		panic(err)
	}

	// setup telemetry database
	a.TelemetryDB, err = telemetrydb.New(cfg)
	if err != nil {
		panic(err)
	}

	// setup address
	a.Address.Setup(cfg.API)

	// create the server
	a.server = &http.Server{
		Addr:    a.ListenOn(),
		Handler: handler,
	}

	// instantiate a new AuthManager based upon auth config settings
	authManager, err := NewAuthManager(&cfg.Auth)
	if err != nil {
		panic(err)
	}
	a.AuthManager = authManager

	return a
}

func (a *App) SetupLogging() error {
	logCfg := &a.Config.Logging

	a.LogManager = logging.NewLogManager()

	if err := a.LogManager.Config(&a.Config.Logging); err != nil {
		slog.Error("Failed to configure logging", slog.Any("config", logCfg), slog.String("error", err.Error()))
		return err
	}

	if a.debugMode {
		slog.Debug("Debug mode enabled - setting log level to debug")
		a.LogManager.SetLevel("DEBUG")
	}

	if err := a.LogManager.Setup(); err != nil {
		slog.Error("Failed to setup logging", slog.Any("config", logCfg), slog.String("error", err.Error()))
		return err
	}

	return nil
}

func (a *App) ListenOn() string {
	return a.Address.String()
}

func (a *App) Initialize() (err error) {

	adbs := []*database.AppDb{
		a.TelemetryDB,
		a.OperationalDB,
	}
	for _, adb := range adbs {
		slog.Debug(
			"Attempting to DB connect and setup tables",
			slog.String("database", adb.Name()),
		)
		// DB connect & setup tables
		if err = adb.Connect(); err != nil {
			slog.Error(
				"DB connection and table setup failed",
				slog.String("database", adb.Name()),
				slog.String("error", err.Error()),
			)
			return
		}
		slog.Debug(
			"Successful DB connect and setup tables",
			slog.String("database", adb.Name()),
		)
	}

	return
}

func (a *App) ListenAndServe() (err error) {
	// start the server up
	slog.Info("Starting Telemetry "+a.Name, slog.String("listenOn", a.ListenOn()))
	if err = a.server.ListenAndServe(); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		} else {
			slog.Error("ListenAndServe() failed", slog.Any("error", err.Error()))
			return
		}
	}
	slog.Info("Shutdown of Telemetry "+a.Name+" complete", slog.String("listenOn", a.ListenOn()))
	return
}

func (a *App) Shutdown() (err error) {
	// create a timeout context to kill the server if shutdown takes too long,
	// deferring a call of the returned cancel() which will cancel the timeout
	// if this routine completes normally, or with error
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	// shutdown the server
	slog.Debug("Attempting server shutdown...")
	if err = a.server.Shutdown(ctx); err != nil {
		slog.Debug("Server shutdown failed", slog.String("err", err.Error()))
		return
	}
	slog.Debug("Succeeded in shutdown of server")

	// close the DB connections
	adbs := []*database.AppDb{
		a.TelemetryDB,
		a.OperationalDB,
	}
	for _, adb := range adbs {
		slog.Debug(
			"Attempting DB close",
			slog.String("database", adb.Name()),
		)
		if err = adb.Close(); err != nil {
			slog.Error(
				"DB close failed",
				slog.String("database", adb.Name()),
				slog.String("err", err.Error()),
			)
			return
		}
		slog.Info(
			"Closed DB",
			slog.String("database", adb.Name()),
		)
	}

	slog.Info("Shutdown complete")

	return
}

const (
	shutdownTimeout = 5 * time.Second
)

var (
	caughtSignals = []os.Signal{
		os.Interrupt,    // generic Ctrl-C or equivalent signal
		syscall.SIGTERM, // linux specific SIGTERM
	}
)

func (a *App) Run() {
	// relay signals
	signal.Notify(a.signals, caughtSignals...)

	// start the server in a goroutine so it doesn't block execution
	go func() {
		err := a.ListenAndServe()
		if err != nil {
			panic(err)
		}
	}()

	// block waiting for signals
	sig := <-a.signals
	slog.Info(
		"Received signal",
		slog.String("signal", sig.String()),
	)

	// shutdown the server
	if err := a.Shutdown(); err != nil {
		slog.Error(
			"Failed to shutdown Telemetry "+a.Name,
			slog.String("err", err.Error()),
		)
		panic(err)
	}
}
