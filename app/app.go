package app

import (
	"compress/gzip"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

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

// AppRequest is a struct tracking the resources associated with handling a request
type AppRequest struct {
	W    http.ResponseWriter
	R    *http.Request
	Vars map[string]string
	Log  *slog.Logger
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
		ar.Log.Info("Response", slog.Int("code", code))
	}
}

// App is a struct tracking the resources associated with the application
type App struct {
	debugMode     bool
	Config        *Config
	TelemetryDB   DbConnection
	OperationalDB DbConnection
	StagingDB     DbConnection
	Address       ServerAddress
	Handler       http.Handler
	Xformers      TelemetryRowXformMapper
	LogManager    *LogManager
}

func NewApp(cfg *Config, handler http.Handler, debugMode bool) *App {
	a := new(App)

	a.Config = cfg
	a.Handler = handler
	a.debugMode = debugMode

	// setup logging first so remaining setup logs with config settings
	if err := a.SetupLogging(); err != nil {
		panic(err)
	}

	// setup databases
	a.TelemetryDB.Setup("Telemetry", cfg.DataBases.Telemetry)
	a.OperationalDB.Setup("Operational", cfg.DataBases.Operational)
	a.StagingDB.Setup("Staging", cfg.DataBases.Staging)

	// setup telemetry transformations
	a.Xformers = new(TelemetryRowXformMap)
	a.Xformers.SetDefault(new(DefaultTelemetryDataRow))
	a.Xformers.Register("SLE-SERVER-SCCHwInfo", new(SccHwInfoTelemetryDataRow))

	// setup address
	a.Address.Setup(cfg.API)

	return a
}

func (a *App) SetupLogging() error {
	logCfg := &a.Config.Logging

	a.LogManager = NewLogManager()

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

	if err := a.StagingDB.EnsureTablesExist(dbTablesStaging); err != nil {
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

	if err := a.TelemetryDB.EnsureTablesExist(dbTablesTelemetry); err != nil {
		slog.Error("Telemetry DB standard tables setup failed", slog.String("error", err.Error()))
		return err
	}

	if err := a.TelemetryDB.EnsureTablesExist(dbTablesXform); err != nil {
		slog.Error("Telemetry DB transform tables exist", slog.String("error", err.Error()))
		return err
	}

	// telemetry type specific transform setup
	if err := a.Xformers.SetupDB(&a.TelemetryDB); err != nil {
		slog.Error("Telemetry transformers setup failed", slog.String("error", err.Error()))
		return err
	}

	return nil
}

func (a *App) Run() {
	slog.Info("Starting Telemetry Server", slog.String("listenOn", a.ListenOn()))
	if err := http.ListenAndServe(a.ListenOn(), a.Handler); err != nil {
		slog.Error("ListenAndServe() failed", slog.Any("error", err.Error()))
		panic(err)
	}
}
