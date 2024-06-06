package app

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	telemetrylib "github.com/SUSE/telemetry/pkg/lib"
	"github.com/SUSE/telemetry/pkg/types"
)

func (a *App) StageTelemetryReport(reqBody []byte, rHeader *telemetrylib.TelemetryReportHeader) (err error) {
	//Stores the report body into the staging database in the reports table

	// create a ReportStagingTableRow struct
	reportStagingRow := ReportStagingTableRow{
		ClientId:   fmt.Sprintf("%d", rHeader.ReportClientId),
		ReportId:   rHeader.ReportId,
		Data:       reqBody,
		ReceivedAt: types.Now().String(),
	}

	if err := reportStagingRow.Insert(a.StagingDB.Conn); err != nil {
		log.Printf("ERR: failed to insert %s: %s", reportStagingRow.ReportIdentifer(), err.Error())
		return err
	}

	return
}

func (a *App) ProcessStagedReports() {
	var reportRow = ReportStagingTableRow{}

	for reportRow.FirstUnallocated(a.StagingDB.Conn) {
		err := a.ProcessStagedReport(&reportRow)
		if err != nil {
			log.Printf("ERR: Failed to process report: %s", err.Error())
			continue
		}
		err = reportRow.Delete(a.StagingDB.Conn)
		if err != nil {
			log.Printf("ERR: Failed to deleted processed report: %s", err.Error())
		}
	}
}

func (a *App) ProcessStagedReport(reportRow *ReportStagingTableRow) (err error) {
	log.Printf("INF: Processing %s", reportRow.ReportIdentifer())

	var report telemetrylib.TelemetryReport

	err = json.Unmarshal(reportRow.Data.([]byte), &report)
	if err != nil {
		log.Printf("ERR: failed to unmarshal data for %s: %s", reportRow.ReportIdentifer(), err.Error())
		return
	}

	// process available bundles, extracting the data items and
	// storing them in the telemetry DB
	for _, bundle := range report.TelemetryBundles {
		bKey := bundle.Header.BundleId
		log.Printf("INF: processing bundle %q", bKey)

		// for each data item in the bundle, process it
		for _, item := range bundle.TelemetryDataItems {
			if err := a.StoreTelemetry(&item, &bundle.Header); err != nil {
				log.Printf("ERR: failed to store telemetry data from bundle %q: %s", bKey, err.Error())
				return err
			}
		}
	}

	return
}

const reportsTableColumns = `(
	id INTEGER NOT NULL PRIMARY KEY,
	clientId INTEGER NOT NULL,
	reportId VARCHAR(64) NOT NULL,
	data BLOB NOT NULL,
	receivedAt VARCHAR(32) NOT NULL,
	allocated BOOLEAN DEFAULT false NOT NULL,
	allocatedAt VARCHAR(32) NULL
)`

type ReportStagingTableRow struct {
	Id          int64  `json:"id"`
	ClientId    string `json:"clientId"`
	ReportId    string `json:"reportId"`
	Data        any    `json:"data"`
	ReceivedAt  string `json:"receivedAt"`
	Allocated   bool   `json:"allocated"`
	AllocatedAt string `json:"allocatedAt"`
}

func (r *ReportStagingTableRow) ReportIdentifer() string {
	return fmt.Sprintf("report %q, client %q, received at %q", r.ReportId, r.ClientId, r.ReceivedAt)
}

func (r *ReportStagingTableRow) Exists(DB *sql.DB) bool {
	row := DB.QueryRow(
		`SELECT id FROM reports WHERE clientId = ? AND reportId = ?`,
		r.ClientId,
		r.ReportId,
	)
	if err := row.Scan(&r.Id); err != nil {
		if err != sql.ErrNoRows {
			log.Printf("ERR: failed when checking for existence of report id %q: %s", r.ReportId, err.Error())
		}
		return false
	}
	return true
}

func (r *ReportStagingTableRow) FirstUnallocated(DB *sql.DB) bool {
	// start a transaction
	TX, err := DB.Begin()
	if err != nil {
		log.Printf("ERR: failed to create a new transaction: %s", err.Error())
		return false
	}

	// retrieve the first unallocated report from the table, returning false if none was found
	row := TX.QueryRow(
		`SELECT id, clientId, reportId, data, receivedAt FROM reports WHERE allocated = false LIMIT 1`,
	)
	if err := row.Scan(&r.Id, &r.ClientId, &r.ReportId, &r.Data, &r.ReceivedAt); err != nil {
		if err == sql.ErrNoRows {
			log.Print("INF: No unallocated staged report rows found")
		} else {
			log.Printf("ERR: failed to retrieve an unallocated staged report row: %s", err.Error())
		}

		if err := TX.Rollback(); err != nil {
			log.Printf("ERR: failed to rollback empty transaction: %s", err.Error())
		}

		return false
	}

	// set AllocatedAt to Now, allows for detection of report processing that got lost
	r.Allocated = true
	r.AllocatedAt = types.Now().String()

	_, err = TX.Exec(`UPDATE reports SET allocated = ?, allocatedAt = ? WHERE id = ?`, r.Allocated, r.AllocatedAt, r.Id)
	if err != nil {
		log.Printf("ERR: failed to update allocation details for staged report entry %v: %s", r.Id, err.Error())

		if err := TX.Rollback(); err != nil {
			log.Printf("ERR: failed to rollback update transaction: %s", err.Error())
		}

		return false
	}

	if err := TX.Commit(); err != nil {
		log.Printf("ERR: failed to commit transaction for staged report entry %v: %s", r.Id, err.Error())

		if err := TX.Rollback(); err != nil {
			log.Printf("ERR: failed to rollback update transaction: %s", err.Error())
		}

		return false
	}

	return true
}

func (r *ReportStagingTableRow) Insert(DB *sql.DB) (err error) {
	_, err = DB.Exec(
		`INSERT INTO reports(clientId, reportId, data, receivedAt) VALUES(?, ?, ?, ?)`,
		r.ClientId, r.ReportId, r.Data, r.ReceivedAt,
	)
	if err != nil {
		log.Printf("ERR: failed to insert Report entry with ReportId %q: %s", r.ReportId, err.Error())
		return err
	}

	return
}

func (r *ReportStagingTableRow) Delete(DB *sql.DB) (err error) {
	_, err = DB.Exec("DELETE FROM reports WHERE reportId = ?", r.ReportId)
	if err != nil {
		log.Printf("ERR: failed to delete Report entry with ReportId %q: %s", r.ReportId, err.Error())
		return err
	}

	return
}
