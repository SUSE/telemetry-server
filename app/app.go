package app

import (
	"compress/gzip"
	"compress/zlib"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/SUSE/telemetry/pkg/logging"
	_ "github.com/mattn/go-sqlite3"
)

// ServerAddress is a struct tracking the server address
type ServerAddress struct {
	Hostname string
	Port     int
}

func (s ServerAddress) String() string {
	return fmt.Sprintf("%s:%d", s.Hostname, s.Port)
}

func (s *ServerAddress) Setup(api APIConfig) {
	s.Hostname, s.Port = api.Host, api.Port
}

// AppVars is a map var name to value
type AppVars map[string]string

// AppRequest is a struct tracking the resources associated with handling a request
type AppRequest struct {
	W     http.ResponseWriter
	R     *http.Request
	Vars  AppVars
	Log   *slog.Logger
	Quiet bool
}

func (ar *AppRequest) getReader() (io.ReadCloser, error) {
	// Check the Content-Encoding header
	switch ar.R.Header.Get("Content-Encoding") {
	case "gzip":
		return gzip.NewReader(ar.R.Body)
	case "deflate":
		return zlib.NewReader(ar.R.Body)
	default:
		return ar.R.Body, nil
	}

}

func ReqLogger(r *http.Request) *slog.Logger {
	return slog.Default().With(slog.String("method", r.Method), slog.Any("URL", r.URL))
}

func NewAppRequest(w http.ResponseWriter, r *http.Request, v AppVars) *AppRequest {
	return &AppRequest{
		W:    w,
		R:    r,
		Vars: v,
		Log:  ReqLogger(r),
	}
}

func QuietAppRequest(w http.ResponseWriter, r *http.Request, v AppVars) (ar *AppRequest) {
	ar = NewAppRequest(w, r, v)
	ar.Quiet = true
	return
}

func (ar *AppRequest) GetHeader(header string) (value string) {
	value = ar.R.Header.Get(header)
	ar.Log.Debug("Request header", slog.String(header, value))
	return
}

func (ar *AppRequest) GetAuthorization() string {
	return ar.GetHeader("Authorization")
}

func (ar *AppRequest) GetAuthToken() string {
	return strings.TrimPrefix(ar.GetAuthorization(), "Bearer ")
}

func (ar *AppRequest) GetRegistrationId() string {
	return ar.GetHeader("X-Telemetry-Registration-Id")
}

func (ar *AppRequest) SetHeader(header, value string) {
	ar.Log.Debug("Response header", slog.String(header, value))
	ar.W.Header().Set(header, value)
}

func (ar *AppRequest) ContentType(contentType string) {
	ar.SetHeader("Content-Type", contentType)
}

func (ar *AppRequest) ContentTypeJSON() {
	ar.ContentType("application/json")
}

func (ar *AppRequest) SetWwwAuthenticate(challenge, realm, scope string) {
	ar.SetHeader(
		"WWW-Authenticate",
		fmt.Sprintf(`%s realm="%s" scope="%s"`, challenge, realm, scope),
	)
}

func (ar *AppRequest) SetWwwAuthScope(scope string) {
	ar.SetWwwAuthenticate("Bearer", "suse-telemetry-service", scope)
}

func (ar *AppRequest) SetWwwAuthReauth() {
	ar.SetWwwAuthScope("authenticate")
}

func (ar *AppRequest) SetWwwAuthRegister() {
	ar.SetWwwAuthScope("register")
}

func (ar *AppRequest) Status(statusCode int) {
	ar.Log.Debug("Response status", slog.Int("code", statusCode))
	ar.W.WriteHeader(statusCode)
}

func (ar *AppRequest) StatusInternalServerError() {
	ar.Status(http.StatusInternalServerError)
}

func (ar *AppRequest) Write(data []byte) (code int, err error) {
	ar.Log.Debug("Response write", slog.Int("length", len(data)))
	code, err = ar.W.Write(data)
	return
}

func (ar *AppRequest) ErrorResponse(code int, errorMessage string) {
	ar.Log.Debug("Setting error response", slog.Int("code", code), slog.String("error", errorMessage))
	ar.JsonResponse(code, map[string]string{"error": errorMessage})
}

func (ar *AppRequest) JsonResponse(code int, payload any) {
	respContent, err := json.Marshal(payload)
	if err != nil {
		ar.ErrorResponse(http.StatusInternalServerError, err.Error())
		ar.Log.Error("Payload marshal failed", slog.Int("code", code), slog.String("error", err.Error()))
		return
	}

	ar.ContentTypeJSON()
	ar.Status(code)
	writeCode, err := ar.Write(respContent)
	if err != nil {
		ar.Log.Error("Response write failed", slog.Int("code", code), slog.Int("writeCode", writeCode), slog.String("error", err.Error()))
	} else {
		if !ar.Quiet {
			ar.Log.Info("Response", slog.Int("code", code))
		} else {
			ar.Log.Debug("Response", slog.Int("code", code))
		}
	}
}

// App is a struct tracking the resources associated with the application
type App struct {
	// public
	Name          string
	Config        *Config
	TelemetryDB   DbConnection
	OperationalDB DbConnection
	StagingDB     DbConnection
	Address       ServerAddress
	Handler       http.Handler
	LogManager    *logging.LogManager
	AuthManager   *AuthManager

	// private
	server    *http.Server
	signals   chan os.Signal
	debugMode bool
}

func NewApp(name string, cfg *Config, handler http.Handler, debugMode bool) *App {
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

	// setup databases
	a.TelemetryDB.Setup("Telemetry", cfg.DataBases.Telemetry)
	a.OperationalDB.Setup("Operational", cfg.DataBases.Operational)
	a.StagingDB.Setup("Staging", cfg.DataBases.Staging)

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

func (a *App) Initialize() error {

	// staging DB setup
	if err := a.StagingDB.Connect(); err != nil {
		slog.Error("Staging DB connection setup failed", slog.String("error", err.Error()))
		return err
	}

	if err := a.StagingDB.EnsureTableSpecsExist(stagingTables); err != nil {
		slog.Error("Staging DB tables setup failed", slog.String("error", err.Error()))
		return err
	}

	// operational DB setup
	if err := a.OperationalDB.Connect(); err != nil {
		slog.Error("Operational DB connection setup failed", slog.String("error", err.Error()))
		return err
	}

	if err := a.OperationalDB.EnsureTableSpecsExist(operationalTables); err != nil {
		slog.Error("Operational DB tables setup failed", slog.String("error", err.Error()))
		return err
	}

	// telemetry DB setup
	if err := a.TelemetryDB.Connect(); err != nil {
		slog.Error("Telemetry DB connection setup failed", slog.String("error", err.Error()))
		return err
	}

	if err := a.TelemetryDB.EnsureTableSpecsExist(telemetryTables); err != nil {
		slog.Error("Telemetry DB tables setup failed", slog.String("error", err.Error()))
		return err
	}

	return nil
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
	dbConns := []DbConnection{
		a.TelemetryDB,
		a.StagingDB,
		a.OperationalDB,
	}
	for _, dbConn := range dbConns {
		slog.Debug("Attempting " + dbConn.name + " DB close...")
		if err = dbConn.Conn.Close(); err != nil {
			slog.Debug(dbConn.name+" DB close failed", slog.String("err", err.Error()))
			return
		}
		slog.Debug("Succeeded in close of " + dbConn.name + " DB")
	}

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
