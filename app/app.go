package app

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	_ "github.com/mattn/go-sqlite3"

	telemetrylib "github.com/SUSE/telemetry/pkg/lib"
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
	Config    *Config
	Extractor telemetrylib.TelemetryExtractor
	DB        DbConnection
	Address   ServerAddress
	Handler   http.Handler
}

func NewApp(cfg *Config, handler http.Handler) *App {
	a := new(App)

	a.Config = cfg

	a.DB.Setup(cfg.DataBases.Telemetry)
	a.Address.Setup(cfg.API)
	a.Handler = handler

	return a
}

func (a *App) ListenOn() string {
	return a.Address.String()
}

func (a *App) Initialize() {
	extractor, err := telemetrylib.NewTelemetryExtractor(&a.Config.DataStores)
	if err != nil {
		log.Fatalf("failed to initialize telemetry extractor: %s", err.Error())
	}
	a.Extractor = extractor
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

func (a *App) ProcessTelemetry() (err error) {
	err = a.ProcessReports()
	if err != nil {
		log.Printf("ERR: processing reports failed: %s", err.Error())
	}

	err = a.ProcessBundles()
	if err != nil {
		log.Printf("ERR: processing bundles failed: %s", err.Error())
	}

	err = a.ProcessDataItems()
	if err != nil {
		log.Printf("ERR: processing data items failed: %s", err.Error())
	}

	return
}

func (a *App) ProcessReports() (err error) {

	numReports, err := a.Extractor.ReportCount()
	if err != nil {
		log.Printf("ERR: failed to determine number of staged reports: %s", err.Error())
		return
	}

	log.Printf("INF: attempting to process %d reports into bundles", numReports)

	err = a.Extractor.ReportsToBundles()
	if err != nil {
		log.Printf("ERR: failed to process reports to bundles: %s", err.Error())
		return
	}

	log.Printf("INF: successfully processed %d reports into bundles", numReports)

	return
}

func (a *App) ProcessBundles() (err error) {

	numBundles, err := a.Extractor.BundleCount()
	if err != nil {
		log.Printf("ERR: failed to determine number of staged bundles: %s", err.Error())
		return
	}

	log.Printf("INF: attempting to process %d bundles into data items", numBundles)

	err = a.Extractor.BundlesToDataItems()
	if err != nil {
		log.Printf("ERR: failed to process bundles to data items: %s", err.Error())
		return
	}

	log.Printf("INF: successfully processed %d bundles into data items", numBundles)

	return
}

func (a *App) ProcessDataItems() (err error) {

	numDataItems, err := a.Extractor.DataItemCount()
	if err != nil {
		log.Printf("ERR: failed to determine number of staged data items: %s", err.Error())
		return
	}

	log.Printf("INF: attempting to process %d data items", numDataItems)

	dataItems, err := a.Extractor.GetDataItems()
	if err != nil {
		log.Printf("ERR: failed to process bundles to data items: %s", err.Error())
		return
	}

	for _, dataItem := range dataItems {
		err = a.StoreDataItem(&dataItem, "placeholder")
		if err != nil {
			log.Printf("ERR: failed to store data item %s: %s", dataItem.Key(), err.Error())
			return err
		}

		a.Extractor.DeleteDataItem(&dataItem)
	}

	log.Printf("INF: successfully processed %d bundles into data items", numDataItems)

	return
}

func (a *App) StoreDataItem(dataItem *telemetrylib.TelemetryDataItem, clientId string) (err error) {
	// when adding a telemetry data item we also need to ensure that all of
	// associated annontations (tags) exist in the tagElements table, then
	// we can add entries to the tagList table to associate the tagElement
	// entries with the telemetryData entries.

	// create a TelemetryDataRow
	tdRow := TelemetryDataRow{
		ClientId:      clientId,
		TelemetryId:   dataItem.Header.TelemetryId,
		TelemetryType: dataItem.Header.TelemetryType,
		Timestamp:     dataItem.Header.TelemetryTimeStamp,
	}

	// marshal telemetry data as JSON
	jsonData, err := json.Marshal(dataItem.TelemetryData)
	if err != nil {
		log.Printf("ERR: failed to marshal telemetry data for client id %q, telemetry id %q as JSON: %s", tdRow.ClientId, tdRow.TelemetryId, err.Error())
		return
	}
	tdRow.DataItem = jsonData

	if !tdRow.Exists(a.DB.Conn) {
		res, err := a.DB.Conn.Exec(
			`INSERT INTO telemetryData(clientId, telemetryId, telemetryType, timestamp, dataItem) VALUES(?, ?, ?, ?, ?)`,
			tdRow.ClientId, tdRow.TelemetryId, tdRow.TelemetryType, tdRow.Timestamp, tdRow.DataItem,
		)
		if err != nil {
			log.Printf("failed to add telemetryData entry for clientId %q telemetryId %q: %s", tdRow.ClientId, tdRow.TelemetryId, err.Error())
			return err
		}
		id, err := res.LastInsertId()
		if err != nil {
			log.Printf("ERR: failed to retrieve id for inserted telemetryData %q: %s", tdRow.TelemetryId, err.Error())
			return err
		}
		tdRow.Id = id
	}

	// create an array of TagElementRows matching dataItem's annontations,
	// adding any that are not already pressent to the tagElement table.
	teRows := make([]TagElementRow, len(dataItem.Header.TelemetryAnnotations))
	for _, tag := range dataItem.Header.TelemetryAnnotations {
		teRow := TagElementRow{Tag: tag}
		if !teRow.Exists(a.DB.Conn) {
			res, err := a.DB.Conn.Exec(`INSERT INTO tagElement(tag) VALUES(?)`, teRow.Tag)
			if err != nil {
				log.Printf("ERR: failed to add tag %q to tagElements table: %s", teRow.Tag, err.Error())
				return err
			}
			id, err := res.LastInsertId()
			if err != nil {
				log.Printf("ERR: failed to retrieve id for inserted tag %q: %s", teRow.Tag, err.Error())
				return err
			}
			teRow.Id = id
		}
		teRows = append(teRows, teRow)
	}

	// add tagList entries to relate tagElement entries to telemetryData entries
	for _, teRow := range teRows {
		tlRow := TagListRow{TelemetryId: tdRow.Id, TagId: teRow.Id}
		if !tlRow.Exists(a.DB.Conn) {
			_, err := a.DB.Conn.Exec(
				`INSERT INTO tagList(telemetryId, tagId) VALUES(?, ?)`,
				tlRow.TelemetryId, tlRow.TagId,
			)
			if err != nil {
				log.Printf("ERR: failed to add tagList (%d, %d): %s", tlRow.TelemetryId, tlRow.TagId, err.Error())
				return err
			}
		}
	}

	return
}
