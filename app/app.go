package app

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strings"
	"time"

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
	Config      *Config
	TelemetryDB DbConnection
	StagingDB   DbConnection
	Address     ServerAddress
	Handler     http.Handler
}

func NewApp(cfg *Config, handler http.Handler) *App {
	a := new(App)

	a.Config = cfg

	a.TelemetryDB.Setup(cfg.DataBases.Telemetry)
	a.StagingDB.Setup(cfg.DataBases.Staging)
	a.Address.Setup(cfg.API)
	a.Handler = handler

	return a
}

func (a *App) ListenOn() string {
	return a.Address.String()
}

func (a *App) Initialize() {

	if err := a.TelemetryDB.Connect(); err != nil {
		log.Fatalf("failed to initialize DB connection: %s", err.Error())
	}

	if err := a.TelemetryDB.EnsureTablesExist(dbTables); err != nil {
		log.Fatalf("failed to ensure required tables exist: %s", err.Error())
	}

	if err := a.StagingDB.Connect(); err != nil {
		log.Fatalf("failed to initialize Staging DB connection: %s", err.Error())
	}

	if err := a.StagingDB.EnsureTablesExist(dbTablesStaging); err != nil {
		log.Fatalf("failed to ensure required tables exist: %s", err.Error())
	}
}

func (a *App) Run() {
	log.Printf("Starting Telemetry Server App on %s", a.ListenOn())
	log.Fatal(http.ListenAndServe(a.ListenOn(), a.Handler))
}

func (a *App) ProcessBundles(report *telemetrylib.TelemetryReport) (err error) {

	// process available bundles, extracting the data items and
	// storing them in the telemetry DB
	for _, bundle := range report.TelemetryBundles {
		bKey := bundle.Header.BundleId
		log.Printf("INF: processing bundle %q", bKey)

		// for each data item in the bundle, process it
		for _, item := range bundle.TelemetryDataItems {
			if err := a.StoreTelemetryData(&item, &bundle.Header); err != nil {
				log.Printf("ERR: failed to store telemetry data from bundle %q: %s", bKey, err.Error())
				return err
			}
		}
	}

	a.DeleteTelemetryReport(report)

	return
}

const tagSetSep = "|"

func uniqueSortTags(tags []string) []string {
	// only need to sort if 2 or more tags present
	if len(tags) < 2 {
		return tags
	}

	// create a map where existing key entries are set to true and new keys are appended to deDuped
	tagMap := map[string]bool{}
	uniqueTags := []string{}

	for _, tag := range tags {
		if tagMap[tag] {
			continue
		}
		tagMap[tag] = true
		uniqueTags = append(uniqueTags, tag)
	}

	// sort unique tag list
	slices.Sort(uniqueTags)

	return uniqueTags
}

func createTagSet(tags []string) string {
	// append the bundle tags to the data item tags
	uniqueTags := uniqueSortTags(tags)
	return fmt.Sprintf("%s%s%s", tagSetSep, strings.Join(uniqueTags, tagSetSep), tagSetSep)
}

func (a *App) StoreTelemetryData(dataItem *telemetrylib.TelemetryDataItem, bHeader *telemetrylib.TelemetryBundleHeader) (err error) {
	// generate a tagSet from the bundle and data item tags
	tagSet := createTagSet(append(dataItem.Header.TelemetryAnnotations, bHeader.BundleAnnotations...))

	tsRow := TagSetRow{
		TagSet: tagSet,
	}

	// add the tagSet to the tagSets table, if not already present
	if !tsRow.Exists(a.TelemetryDB.Conn) {
		if err := tsRow.Insert(a.TelemetryDB.Conn); err != nil {
			log.Printf("ERR: failed to add tagSet %q: %s", tsRow.TagSet, err.Error())
			return err
		}

		log.Printf("INF: successfully added tagSet %q as telemetryData entry %d", tsRow.TagSet, tsRow.Id)
	}

	// when adding a telemetry data item we also need to ensure that all of
	// associated annontations (tags) exist in the tagElements table, then
	// we can add entries to the tagList table to associate the tagElement
	// entries with the telemetryData entries.

	// create a TelemetryDataRow
	tdRow := TelemetryDataRow{
		ClientId:      bHeader.BundleClientId,
		CustomerId:    bHeader.BundleCustomerId,
		TelemetryId:   dataItem.Header.TelemetryId,
		TelemetryType: dataItem.Header.TelemetryType,
		Timestamp:     dataItem.Header.TelemetryTimeStamp,
		TagSetId:      tsRow.Id,
	}

	// marshal telemetry data as JSON
	jsonData, err := json.Marshal(dataItem.TelemetryData)
	if err != nil {
		log.Printf("ERR: failed to marshal telemetry data for client id %q, customer id %q, telemetry id %q as JSON: %s", tdRow.ClientId, tdRow.CustomerId, tdRow.TelemetryId, err.Error())
		return
	}
	tdRow.DataItem = jsonData

	if !tdRow.Exists(a.TelemetryDB.Conn) {
		if err := tdRow.Insert(a.TelemetryDB.Conn); err != nil {
			log.Printf("ERR: failed to add data item %q: %s", dataItem.Header.TelemetryId, err.Error())
			return err
		}

		log.Printf("INF: successfully added data item %q as telemetryData entry %d", dataItem.Header.TelemetryId, tdRow.Id)
	}

	return
}

func (a *App) StoreTelemetryReport(reqBody []byte, key string) (err error) {
	//Stores the report body into the staging database in the reports table
	receivedTimestamp := time.Now()

	// create a ReportStagingTableRow struct
	reportStagingRow := ReportStagingTableRow{
		Key:               key,
		Data:              reqBody,
		Processed:         false,
		ReceivedTimestamp: receivedTimestamp.String(),
	}

	if err := reportStagingRow.Insert(a.StagingDB.Conn); err != nil {
		log.Printf("ERR: failed to insert report with ReportId %q: %s", key, err.Error())
		return err
	}

	return
}

func (a *App) DeleteTelemetryReport(report *telemetrylib.TelemetryReport) (err error) {

	reportStagingRow := ReportStagingTableRow{
		Key: report.Header.ReportId,
	}

	if err := reportStagingRow.Delete(a.StagingDB.Conn); err != nil {
		log.Printf("ERR: failed to delete report with ReportId %q: %s", report.Header.ReportId, err.Error())
		return err
	}
	return
}
