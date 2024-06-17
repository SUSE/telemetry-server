package app

import (
	"encoding/json"
	"fmt"
	"log"
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
}

func (ar *AppRequest) ContentType(contentType string) {
	ar.W.Header().Set("Content-Type", contentType)
}

func (ar *AppRequest) ContentTypeJSON() {
	ar.ContentType("application/json")
}

func (ar *AppRequest) Status(statusCode int) {
	ar.W.WriteHeader(statusCode)
}

func (ar *AppRequest) StatusInternalServerError() {
	ar.Status(http.StatusInternalServerError)
}

func (ar *AppRequest) Write(data []byte) (code int, err error) {
	code, err = ar.W.Write(data)
	return
}

func (ar *AppRequest) ErrorResponse(code int, errorMessage string) {
	ar.JsonResponse(code, map[string]string{"error": errorMessage})
}

func (ar *AppRequest) JsonResponse(code int, payload any) {
	respContent, err := json.Marshal(payload)
	if err != nil {
		ar.ErrorResponse(http.StatusInternalServerError, err.Error())
		log.Printf("ERR: %s %s %d: failed to marshal payload: %q", ar.R.Method, ar.R.URL, code, err.Error())
		return
	}

	ar.ContentTypeJSON()
	ar.Status(code)
	writeCode, err := ar.Write(respContent)
	if err != nil {
		log.Printf("ERR: %s %s %d: response write failed (%d, %q)", ar.R.Method, ar.R.URL, code, writeCode, err.Error())
	} else {
		log.Printf("INF: %s %s %d", ar.R.Method, ar.R.URL, code)
	}
}

// App is a struct tracking the resources associated with the application
type App struct {
	Config      *Config
	TelemetryDB DbConnection
	StagingDB   DbConnection
	Address     ServerAddress
	Handler     http.Handler
	Xformers    TelemetryRowXformMapper
}

func NewApp(cfg *Config, handler http.Handler) *App {
	a := new(App)

	a.Config = cfg

	a.TelemetryDB.Setup(cfg.DataBases.Telemetry)
	a.StagingDB.Setup(cfg.DataBases.Staging)
	a.Address.Setup(cfg.API)
	a.Handler = handler

	a.Xformers = new(TelemetryRowXformMap)
	a.Xformers.SetDefault(new(DefaultTelemetryDataRow))
	a.Xformers.Register("SLE-SERVER-SCCHwInfo", new(SccHwInfoTelemetryDataRow))

	return a
}

func (a *App) ListenOn() string {
	return a.Address.String()
}

func (a *App) Initialize() {

	// telemetry DB setup
	if err := a.TelemetryDB.Connect(); err != nil {
		log.Fatalf("ERR: failed to initialize Telemetry DB connection: %s", err.Error())
	}

	if err := a.TelemetryDB.EnsureTablesExist(dbTablesTelemetry); err != nil {
		log.Fatalf("ERR: failed to ensure Telemetry DB required tables exist: %s", err.Error())
	}

	if err := a.TelemetryDB.EnsureTablesExist(dbTablesXform); err != nil {
		log.Fatalf("ERR: failed to ensure Telemetry DB transform tables exist: %s", err.Error())
	}

	// staging DB setup
	if err := a.StagingDB.Connect(); err != nil {
		log.Fatalf("ERR: failed to initialize Staging DB connection: %s", err.Error())
	}

	if err := a.StagingDB.EnsureTablesExist(dbTablesStaging); err != nil {
		log.Fatalf("ERR: failed to ensure Staging DB required tables exist: %s", err.Error())
	}

	// telemetry type specific transform setup
	if err := a.Xformers.SetupDB(a.TelemetryDB.Conn); err != nil {
		log.Fatalf("ERR: failed to setup storage transforms: %s", err.Error())
	}
}

func (a *App) Run() {
	log.Printf("INF: Starting Telemetry Server App on %s", a.ListenOn())
	log.Fatal(http.ListenAndServe(a.ListenOn(), a.Handler))
}
