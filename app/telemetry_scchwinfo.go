package app

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"strconv"

	telemetrylib "github.com/SUSE/telemetry/pkg/lib"
)

var sccHwInfoTelemetryTableSpec = TableSpec{
	Name: "telemetrySccHwInfo",
	Columns: []TableSpecColumn{
		// common fields
		{Name: "id", Type: "INTEGER", PrimaryKey: true, Identity: true},
		{Name: "clientId", Type: "INTEGER"},
		{Name: "customerId", Type: "INTEGER"},
		{Name: "telemetryId", Type: "VARCHAR"},
		{Name: "telemetryType", Type: "VARCHAR"},
		{Name: "tagSetId", Type: "INTEGER", Nullable: true},
		{Name: "timestamp", Type: "VARCHAR"},

		// telemetry type specific fields
		{Name: "hostname", Type: "VARCHAR"},
		{Name: "distroTarget", Type: "VARCHAR"},
		{Name: "cpus", Type: "INTEGER"},
		{Name: "sockets", Type: "INTEGER"},
		{Name: "memTotal", Type: "INTEGER"},
		{Name: "arch", Type: "VARCHAR"},
		{Name: "uuid", Type: "VARCHAR"},
		{Name: "hypervisor", Type: "VARCHAR", Nullable: true},
		{Name: "cloudProvider", Type: "VARCHAR", Nullable: true},
	},
}

type SccHwInfoTelemetryDataRow struct {
	TelemetryDataCommon

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

	// unmarshal the provided telemetry JSON blob
	var tData map[string]any
	err = json.Unmarshal([]byte(dItm.TelemetryData), &tData)
	if err != nil {
		slog.Error(
			"Failed to unmarshal telemetry data JSON blob",
			slog.String("telemetryType", t.TelemetryType),
			slog.String("error", err.Error()),
		)
		return
	}

	hwiName := "hwinfo"
	err = checkRequiredMapFieldsExist(tData, hwiName, "distro_target")
	if err != nil {
		slog.Error("required data fields missing", slog.String("telemetryType", t.TelemetryType), slog.String("error", err.Error()))
		return
	}
	hwi, ok := tData[hwiName].(map[string]any)
	if !ok {
		err := fmt.Errorf("field %q in telemetryType %q data is not a map", hwiName, t.TelemetryType)
		return err
	}
	err = checkRequiredMapFieldsExist(hwi, "hostname", "cpus", "sockets", "mem_total", "arch", "uuid", "hypervisor", "cloud_provider")
	if err != nil {
		slog.Error("required data subfields missing", slog.String("field", hwiName), slog.String("telemetryType", t.TelemetryType), slog.String("error", err.Error()))
		return
	}

	t.Hostname = hwi["hostname"].(string)
	t.DistroTarget = tData["distro_target"].(string)

	t.Cpus, err = int64Conv(hwi["cpus"])
	if err != nil {
		slog.Error(
			"type conversion failed",
			slog.String("field", hwiName+".cpus"),
			slog.Any("value", hwi["cpus"]),
			slog.String("type", "int64"),
			slog.String("error", err.Error()),
		)
		return
	}

	t.Sockets, err = int64Conv(hwi["sockets"])
	if err != nil {
		slog.Error(
			"type conversion failed",
			slog.String("field", hwiName+".sockets"),
			slog.Any("value", hwi["sockets"]),
			slog.String("type", "int64"),
			slog.String("error", err.Error()),
		)
		return
	}

	t.MemTotal, err = int64Conv(hwi["mem_total"])
	if err != nil {
		slog.Error(
			"type conversion failed",
			slog.String("field", hwiName+".mem_total"),
			slog.Any("value", hwi["mem_total"]),
			slog.String("type", "int64"),
			slog.String("error", err.Error()),
		)
		return
	}

	t.Arch = hwi["arch"].(string)
	t.UUID = hwi["uuid"].(string)
	t.Hypervisor = hwi["hypervisor"].(string)
	t.CloudProvider = hwi["cloud_provider"].(string)

	return
}

func (t *SccHwInfoTelemetryDataRow) SetupDB(db *DbConnection) error {
	t.tableSpec = &sccHwInfoTelemetryTableSpec
	return t.TelemetryDataCommon.SetupDB(db)
}

func (t *SccHwInfoTelemetryDataRow) TableName() string {
	return t.TableRowCommon.TableName()
}

func (t *SccHwInfoTelemetryDataRow) RowId() int64 {
	return t.Id
}

func (t *SccHwInfoTelemetryDataRow) String() string {
	bytes, _ := json.Marshal(t)
	return string(bytes)
}

func (t *SccHwInfoTelemetryDataRow) Exists() bool {
	stmt, err := t.SelectStmt(
		// select columns
		[]string{
			"id",
			"customerId",
			"telemetryType",
			"tagSetId",
			"hostname",
			"distroTarget",
			"cpus",
			"sockets",
			"memTotal",
			"arch",
			"uuid",
			"hypervisor",
			"cloudProvider",
		},
		// match columns
		[]string{
			"clientId",
			"telemetryId",
			"timestamp",
		},
		SelectOpts{}, // no special options
	)
	if err != nil {
		slog.Error(
			"exists statement generation failed",
			slog.String("table", t.TableName()),
			slog.String("error", err.Error()),
		)
		panic(err)
	}

	row := t.DB().QueryRow(
		stmt,
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
			slog.Error(
				"check for matching entry failed",
				slog.String("table", t.TableName()),
				slog.Int64("clientId", t.ClientId),
				slog.String("telemetryId", t.TelemetryId),
				slog.String("timestamp", t.Timestamp),
				slog.String("error", err.Error()),
			)
		}
		return false
	}
	return true
}

func (t *SccHwInfoTelemetryDataRow) Insert() (err error) {
	stmt, err := t.InsertStmt(
		[]string{
			"clientId",
			"customerId",
			"telemetryId",
			"telemetryType",
			"timestamp",
			"tagSetId",
			"hostname",
			"distroTarget",
			"cpus",
			"sockets",
			"memTotal",
			"arch",
			"uuid",
			"hypervisor",
			"cloudProvider",
		},
		"id",
	)
	if err != nil {
		slog.Error(
			"insert statement generation failed",
			slog.String("table", t.TableName()),
			slog.String("error", err.Error()),
		)
		return
	}

	row := t.DB().QueryRow(
		stmt,
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
	if err = row.Scan(
		&t.Id,
	); err != nil {
		slog.Error(
			"insert failed",
			slog.String("table", t.TableName()),
			slog.Int64("clientId", t.ClientId),
			slog.String("telemetryId", t.TelemetryId),
			slog.String("timestamp", t.Timestamp),
			slog.String("error", err.Error()),
		)
	}

	return
}

func (t *SccHwInfoTelemetryDataRow) Update() (err error) {
	stmt, err := t.UpdateStmt(
		[]string{
			"clientId",
			"customerId",
			"telemetryId",
			"telemetryType",
			"timestamp",
			"tagSetId",
			"hostname",
			"distroTarget",
			"cpus",
			"sockets",
			"memTotal",
			"arch",
			"uUID",
			"hypervisor",
			"cloudProvider",
		},
		[]string{
			"Id",
		},
	)
	if err != nil {
		slog.Error(
			"update statement generation failed",
			slog.String("table", t.TableName()),
			slog.String("error", err.Error()),
		)
		return
	}

	_, err = t.DB().Exec(
		stmt,
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
		slog.Error(
			"update failed",
			slog.String("table", t.TableName()),
			slog.Int64("id", t.Id),
			slog.String("error", err.Error()),
		)
	}

	return
}

func (t *SccHwInfoTelemetryDataRow) Delete() (err error) {
	stmt, err := t.DeleteStmt(
		[]string{
			"id",
		},
	)
	if err != nil {
		slog.Error(
			"delete statement generation failed",
			slog.String("table", t.TableName()),
			slog.String("error", err.Error()),
		)
		return
	}

	_, err = t.DB().Exec(
		stmt,
		t.Id,
	)
	if err != nil {
		slog.Error(
			"delete failed",
			slog.String("table", t.TableName()),
			slog.Int64("id", t.Id),
			slog.String("error", err.Error()),
		)
	}
	return
}

// validate that SccHwInfoTelemetryDataRow implements TelemetryDataRow interface
var _ TelemetryDataRowHandler = (*SccHwInfoTelemetryDataRow)(nil)
