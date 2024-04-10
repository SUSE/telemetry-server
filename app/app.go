package app

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
)

// DbConnection is a struct tracking a DB connection and associated DB settings
type DbConnection struct {
	Conn       *sql.DB
	Driver     string
	DataSource string
}

func (d DbConnection) String() string {
	return fmt.Sprintf("%p:%s:%s", d.Conn, d.Driver, d.DataSource)
}

func (d *DbConnection) Setup(driver, dataSource string) {
	d.Driver, d.DataSource = driver, dataSource
}

func (d *DbConnection) Connect() (err error) {
	d.Conn, err = sql.Open(d.Driver, d.DataSource)
	if err != nil {
		log.Printf("Failed to connect to DB '%s:%s': %s", d.Driver, d.DataSource, err.Error())
	}

	return
}

const clientsTableColumns = `(
	id               INTEGER     NOT NULL PRIMARY KEY,
	clientInstanceId VARCHAR(64) NOT NULL,
	registrationDate VARCHAR(32) NOT NULL,
	authToken        VARCHAR(32) NOT NULL
)`

type ClientsRow struct {
	Id               int64  `json:"id"`
	ClientInstanceId string `json:"clientInstanceId"`
	RegistrationDate string `json:"registrationDate"`
	AuthToken        string `json:"AuthToken"`
}

func (c *ClientsRow) String() string {
	bytes, _ := json.Marshal(c)
	return string(bytes)
}

func (c *ClientsRow) Exists(DB *sql.DB) bool {
	row := DB.QueryRow(`SELECT id FROM clients WHERE clientInstanceId = ?`, c.ClientInstanceId)
	if err := row.Scan(&c.Id); err != nil {
		if err != sql.ErrNoRows {
			log.Printf("failed when checking for existence of client with clientInstanceId = %q: %s", c.ClientInstanceId, err.Error())
		}
		return false
	}
	return true
}

const tagElementTableColumns = `(
	id INTEGER NOT NULL PRIMARY KEY,
	tag VARCHAR(256) NOT NULL
)`

const tagListTableColumns = `(
	telemetryId INTEGER NOT NULL,
	tagId INTEGER NOT NULL,
	FOREIGN KEY (telemetryId) REFERENCES telemetryData (id)
	FOREIGN KEY (tagId) REFERENCES tagElement (id)
	PRIMARY KEY (telemetryId, tagId)
)`

const telemetryDataTableColumns = `(
	id INTEGER NOT NULL PRIMARY KEY,
	blob VARCHAR(1024) NOT NULL,
	timestamp VARCHAR(32) NOT NULL
)`

var dbTables = map[string]string{
	"clients":       clientsTableColumns,
	"tagElement":    tagElementTableColumns,
	"tagList":       tagListTableColumns,
	"telemetryData": telemetryDataTableColumns,
}

func (d *DbConnection) EnsureTablesExist() (err error) {
	for name, columns := range dbTables {
		createCmd := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s %s", name, columns)
		log.Printf("createCmd: %q", createCmd)
		_, err = d.Conn.Exec(createCmd)
		if err != nil {
			log.Printf("failed to create table %q: %s", name, err.Error())
			return
		}
	}

	return
}

// ServerAddress is a struct tracking the server address
type ServerAddress struct {
	Hostname string
	Port     int
}

func (s ServerAddress) String() string {
	return fmt.Sprintf("%s:%d", s.Hostname, s.Port)
}

func (s *ServerAddress) Setup(hostname string, port int) {
	s.Hostname, s.Port = hostname, port
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
	DB      DbConnection
	Address ServerAddress
	Handler http.Handler
}

func NewApp(driver, dataSource, hostname string, port int, handler http.Handler) *App {
	a := new(App)

	a.DB.Setup(driver, dataSource)
	a.Address.Setup(hostname, port)
	a.Handler = handler

	return a
}

func (a *App) ListenOn() string {
	return a.Address.String()
}

func (a *App) Initialize() {
	if err := a.DB.Connect(); err != nil {
		log.Fatalf("failed to initialize DB connection: %s", err.Error())
	}

	if err := a.DB.EnsureTablesExist(); err != nil {
		log.Fatalf("failed to ensure required tables exist: %s", err.Error())
	}
}

func (a *App) Run() {
	log.Printf("Starting Telemetry Server App on %s", a.ListenOn())
	log.Fatal(http.ListenAndServe(a.ListenOn(), a.Handler))
}
