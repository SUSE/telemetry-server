package app

import (
	"database/sql"
	"log"

	telemetrylib "github.com/SUSE/telemetry/pkg/lib"
)

func (a *App) StoreTelemetry(dItm *telemetrylib.TelemetryDataItem, bHeader *telemetrylib.TelemetryBundleHeader) (err error) {
	// generate a tagSet from the bundle and data item tags
	tagSet := createTagSet(append(dItm.Header.TelemetryAnnotations, bHeader.BundleAnnotations...))

	tsRow := TagSetRow{
		TagSet: tagSet,
	}

	// add the tagSet to the tagSets table, if not already present
	if !tsRow.Exists(a.TelemetryDB.Conn) {
		if err := tsRow.Insert(a.TelemetryDB.Conn); err != nil {
			log.Printf("ERR: failed to add tagSet %q: %s", tsRow.TagSet, err.Error())
			return err
		}

		log.Printf("INF: successfully added tagSet %q as entry %v", tsRow.TagSet, tsRow.Id)
	}

	// store the telemetry
	err = a.StoreTelemetryData(dItm, bHeader, tsRow.Id)
	if err != nil {
		log.Printf("ERR: Failed to store telemetry: %s", err.Error())
	}
	return
}

func (a *App) StoreTelemetryData(dItm *telemetrylib.TelemetryDataItem, bHdr *telemetrylib.TelemetryBundleHeader, tagSetId int64) (err error) {
	tdRow := a.Xformers.Get(dItm.Header.TelemetryType)

	err = tdRow.Init(dItm, bHdr, tagSetId)
	if err != nil {
		log.Printf(
			"ERR: Init() failed for unstructured tdRow for telemetryId %q: %s",
			dItm.Header.TelemetryId, err.Error(),
		)
		return
	}

	if !tdRow.Exists() {
		if err := tdRow.Insert(); err != nil {
			log.Printf(
				"ERR: failed to add entry to table %q for telemetryId %q: %s",
				tdRow.TableName(), dItm.Header.TelemetryId, err.Error(),
			)
			return err
		}

		log.Printf(
			"INF: successfully added table %q entry for telemetryID %q id %v",
			tdRow.TableName(), dItm.Header.TelemetryId, tdRow.RowId(),
		)
	}

	return
}

type TelemetryDataRow interface {
	// Initialise the row fields
	Init(dItm *telemetrylib.TelemetryDataItem, bHdr *telemetrylib.TelemetryBundleHeader, tagSetId int64) error

	// Setup DB access
	SetupDB(db *sql.DB) error

	// Retrieve the TableName
	TableName() string

	// Retrieve the RowId
	RowId() int64

	// Return string representation of the row
	String() string

	// Check if the row exists in the DB, and if so populate it
	Exists() bool

	// Insert row into the table
	Insert() error

	// Update row in the table
	Update() error

	// Delete row from the table
	Delete() error
}

type TelemetryDataCommon struct {
	// private db settings
	db     *sql.DB
	table  string
	exists *sql.Stmt
	insert *sql.Stmt
	update *sql.Stmt
	delete *sql.Stmt

	// public table fields
	Id            int64  `json:"id"`
	ClientId      int64  `json:"clientId"`
	CustomerId    string `json:"customerId"`
	TelemetryId   string `json:"telemetryId"`
	TelemetryType string `json:"telemetryType"`
	Timestamp     string `json:"timestamp"`
	TagSetId      int64  `json:"tagSetId"`
}

func (t *TelemetryDataCommon) SetupDB(db *sql.DB) {
	// save DB reference
	t.db = db
}

func (t *TelemetryDataCommon) Init(dItm *telemetrylib.TelemetryDataItem, bHdr *telemetrylib.TelemetryBundleHeader, tagSetId int64) {
	// init common telemetry data fields
	t.ClientId = bHdr.BundleClientId
	t.CustomerId = bHdr.BundleCustomerId
	t.TelemetryId = dItm.Header.TelemetryId
	t.TelemetryType = dItm.Header.TelemetryType
	t.Timestamp = dItm.Header.TelemetryTimeStamp
	t.TagSetId = tagSetId
}
