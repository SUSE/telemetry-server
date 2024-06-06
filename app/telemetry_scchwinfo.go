package app

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strconv"

	telemetrylib "github.com/SUSE/telemetry/pkg/lib"
)

const sccHwInfoTelemetryTableColumns = `(
	id INTEGER NOT NULL PRIMARY KEY,
	clientId INTEGER NOT NULL,
	customerId INTEGER NOT NULL,
	telemetryId VARCHAR(64) NOT NULL,
	telemetryType VARCHAR(64) NOT NULL,
	tagSetId INTEGER NULL,
	timestamp VARCHAR(32) NOT NULL,
	hostname VARCHAR NOT NULL,
	distroTarget VARCHAR NOT NULL,
	cpus INTEGER NOT NULL,
	sockets INTEGER NOT NULL,
	memTotal INTEGER NOT NULL,
	arch VARCHAR NOT NULL,
	uuid VARCHAR NOT NULL,
	hypervisor VARCHAR NULL,
	cloudProvider VARCHAR NULL
)`

type SccHwInfoTelemetryDataRow struct {
	TelemetryDataCommon
	//Id            int64  `json:"id"`
	//ClientId      int64  `json:"clientId"`
	//CustomerId    string `json:"customerId"`
	//TelemetryId   string `json:"telemetryId"`
	//TelemetryType string `json:"telemetryType"`
	//Timestamp     string `json:"timestamp"`
	//TagSetId      int 64 `json:"tagSetId"`
	Hostname      string `json:"hostname"`
	DistroTarget  string `json:"distroTarget"`
	Cpus          int64  `json:"cpus"`
	Sockets       int64  `json:"sockets"`
	MemTotal      int64  `json:"memTotal"`
	Arch          string `json:"arch"`
	UUID          string `json:"uuid"`
	Hypervisor    string `json:"hypervisor"`
	CloudProvider string `json:"cloudProvider"`
}

func checkRequiredMapFieldsExist(data map[string]any, fields ...string) (err error) {
	missing := []string{}
	for _, field := range fields {
		if _, ok := data[field]; ok {
			continue
		}
		missing = append(missing, field)
	}

	if len(missing) > 0 {
		err = fmt.Errorf("required fields %q not found", missing)
	}

	return
}

func int64Conv(value any) (outValue int64, err error) {
	switch t := value.(type) {
	case uint64:
		outValue = int64(value.(uint64))
	case float64:
		outValue = int64(math.Round(value.(float64)))
	case string:
		outValue, err = strconv.ParseInt(value.(string), 0, 64)
	default:
		err = fmt.Errorf("unsupport type %T for integer conversion", t)
	}

	return
}

func (t *SccHwInfoTelemetryDataRow) Init(dItm *telemetrylib.TelemetryDataItem, bHdr *telemetrylib.TelemetryBundleHeader, tagSetId int64) (err error) {
	t.TelemetryDataCommon.Init(dItm, bHdr, tagSetId)

	err = checkRequiredMapFieldsExist(dItm.TelemetryData, "hwinfo", "distro_target")
	if err != nil {
		log.Printf("ERR: telemetryType %q: %s", t.TelemetryType, err.Error())
		return
	}
	hwiName := "hwinfo"
	hwi, ok := dItm.TelemetryData[hwiName].(map[string]any)
	if !ok {
		err := fmt.Errorf("field %q in telemetryType %q data is not a map", hwiName, t.TelemetryType)
		log.Printf("ERR: %s", err.Error())
		return err
	}
	err = checkRequiredMapFieldsExist(hwi, "hostname", "cpus", "sockets", "mem_total", "arch", "uuid", "hypervisor", "cloud_provider")
	if err != nil {
		log.Printf("ERR: field %q in telemetryType %q: %s", hwiName, t.TelemetryType, err.Error())
		return
	}

	t.Hostname = hwi["hostname"].(string)
	t.DistroTarget = dItm.TelemetryData["distro_target"].(string)

	t.Cpus, err = int64Conv(hwi["cpus"])
	if err != nil {
		log.Printf("ERR: failed to convert %q field to %q for %t: %s", "cpus", "int64", hwi["cpus"], err.Error())
	}

	t.Sockets, err = int64Conv(hwi["sockets"])
	if err != nil {
		log.Printf("ERR: failed to convert %q field to %q for %t: %s", "sockets", "int64", hwi["sockets"], err.Error())
	}

	t.MemTotal, err = int64Conv(hwi["mem_total"])
	if err != nil {
		log.Printf("ERR: failed to convert %q field to %q for %t: %s", "mem_total", "int64", hwi["mem_total"], err.Error())
	}

	t.Arch = hwi["arch"].(string)
	t.UUID = hwi["uuid"].(string)
	t.Hypervisor = hwi["hypervisor"].(string)
	t.CloudProvider = hwi["cloud_provider"].(string)

	return
}

func (t *SccHwInfoTelemetryDataRow) SetupDB(db *sql.DB) (err error) {
	t.TelemetryDataCommon.SetupDB(db)

	t.table = `telemetrySccHwInfo`

	// prepare exists check statement
	t.exists, err = t.db.Prepare(
		`SELECT id, customerId, telemetryType, tagSetId, hostname, distroTarget, cpus, sockets, memTotal, arch, uuid, hypervisor, cloudProvider FROM telemetrySccHwInfo WHERE clientId = ? AND telemetryId = ? AND timestamp = ?`,
	)
	if err != nil {
		log.Printf("ERR: Failed to prepare exists statement for %q: %s", t.table, err.Error())
		return
	}

	// prepare insert statement
	t.insert, err = t.db.Prepare(
		`INSERT INTO telemetrySccHwInfo(clientId, customerId, telemetryId, telemetryType, timestamp, tagSetId, hostname, distroTarget, cpus, sockets, memTotal, arch, uuid, hypervisor, cloudProvider) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		log.Printf("ERR: Failed to prepare insert statement for %q: %s", t.table, err.Error())
		return
	}

	// prepare update statement
	t.update, err = t.db.Prepare(
		`UPDATE telemetrySccHwInfo SET clientId = ?, customerId = ?, telemetryId = ?, telemetryType = ?, timestamp = ?, tagSetId = ?, hostname = ?, distroTarget = ?, cpus = ?, sockets = ?, memTotal = ?, arch = ?, uuid = ?, hypervisor = ?, cloudProvider = ? WHERE id = ?`,
	)
	if err != nil {
		log.Printf("ERR: Failed to prepare update statement for %q: %s", t.table, err.Error())
		return
	}

	// prepare delete statement
	t.delete, err = t.db.Prepare(
		`DELETE FROM telemetrySccHwInfo WHERE id = ?`,
	)
	if err != nil {
		log.Printf("ERR: Failed to prepare delete statement for %q: %s", t.table, err.Error())
		return
	}

	return
}

func (t *SccHwInfoTelemetryDataRow) TableName() string {
	return t.table
}

func (t *SccHwInfoTelemetryDataRow) RowId() int64 {
	return t.Id
}

func (t *SccHwInfoTelemetryDataRow) String() string {
	bytes, _ := json.Marshal(t)
	return string(bytes)
}

func (t *SccHwInfoTelemetryDataRow) Exists() bool {
	row := t.exists.QueryRow(
		t.ClientId,
		t.TelemetryId,
		t.Timestamp,
	)
	// if the entry was found, all fields not used to find the entry will have
	// been updated to match what is in the DB
	if err := row.Scan(
		&t.Id,
		&t.CustomerId,
		&t.TelemetryType,
		&t.TagSetId,
		&t.Hostname,
		&t.DistroTarget,
		&t.Cpus,
		&t.Sockets,
		&t.MemTotal,
		&t.Arch,
		&t.UUID,
		&t.Hypervisor,
		&t.CloudProvider,
	); err != nil {
		if err != sql.ErrNoRows {
			log.Printf(
				"ERR: failed when checking table %q for entry matching clientId %q, telemetryId %q, timestamp %q: %s",
				t.table, t.ClientId, t.TelemetryId, t.Timestamp, err.Error(),
			)
		}
		return false
	}
	return true
}

func (t *SccHwInfoTelemetryDataRow) Insert() (err error) {
	res, err := t.insert.Exec(
		t.ClientId,
		t.CustomerId,
		t.TelemetryId,
		t.TelemetryType,
		t.Timestamp,
		t.TagSetId,
		t.Hostname,
		t.DistroTarget,
		t.Cpus,
		t.Sockets,
		t.MemTotal,
		t.Arch,
		t.UUID,
		t.Hypervisor,
		t.CloudProvider,
	)
	if err != nil {
		log.Printf(
			"ERR: failed to add table %q entry for clientId %q, telemetryId %q, timestamp %q: %s",
			t.table, t.ClientId, t.TelemetryId, t.Timestamp, err.Error(),
		)
		return
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Printf(
			"ERR: failed to retrieve id for newly inserted table %q entry for clientId %q, telemetryId %q, timestamp %q: %s",
			t.table, t.ClientId, t.TelemetryId, t.Timestamp, err.Error(),
		)
		return
	}
	t.Id = id

	return
}

func (t *SccHwInfoTelemetryDataRow) Update() (err error) {
	_, err = t.update.Exec(
		t.ClientId,
		t.CustomerId,
		t.TelemetryId,
		t.TelemetryType,
		t.Timestamp,
		t.TagSetId,
		t.Hostname,
		t.DistroTarget,
		t.Cpus,
		t.Sockets,
		t.MemTotal,
		t.Arch,
		t.UUID,
		t.Hypervisor,
		t.CloudProvider,
		t.Id,
	)
	if err != nil {
		log.Printf(
			"ERR: failed to update table %q entry %v: %s",
			t.table, t.Id, err.Error(),
		)
	}
	return
}

func (t *SccHwInfoTelemetryDataRow) Delete() (err error) {
	_, err = t.delete.Exec(
		t.Id,
	)
	if err != nil {
		log.Printf(
			"ERR: failed to delete table %q entry %v: %s",
			t.table, t.Id, err.Error(),
		)
	}
	return
}

// validate that SccHwInfoTelemetryDataRow implements TelemetryDataRow interface
var _ TelemetryDataRow = (*SccHwInfoTelemetryDataRow)(nil)
