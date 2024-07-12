package app

import (
	"log/slog"

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
			slog.Error("tagSet insert failed", slog.String("tagSet", tsRow.TagSet), slog.String("error", err.Error()))
			return err
		}

		slog.Info("tagSet added successfully", slog.String("tagSet", tsRow.TagSet), slog.Int64("id", tsRow.Id))
	}

	// store the telemetry
	err = a.StoreTelemetryData(dItm, bHeader, tsRow.Id)
	if err != nil {
		slog.Error("telemetry store failed", slog.String("telemetryId", dItm.Header.TelemetryId), slog.String("error", err.Error()))
	}
	return
}

func (a *App) StoreTelemetryData(dItm *telemetrylib.TelemetryDataItem, bHdr *telemetrylib.TelemetryBundleHeader, tagSetId int64) (err error) {
	tdRow := a.Xformers.Get(dItm.Header.TelemetryType)

	err = tdRow.Init(dItm, bHdr, tagSetId)
	if err != nil {
		slog.Error(
			"unstructured tdRow init failed",
			slog.String("telemetryId", dItm.Header.TelemetryId),
			slog.String("error", err.Error()),
		)
		return
	}

	if !tdRow.Exists() {
		if err := tdRow.Insert(); err != nil {
			slog.Error(
				"unstructured tdRow insert failed",
				slog.String("tableName", tdRow.TableName()),
				slog.String("telemetryId", dItm.Header.TelemetryId),
				slog.String("error", err.Error()),
			)
			return err
		}

		slog.Info(
			"unstructured tdRow insert success",
			slog.String("tableName", tdRow.TableName()),
			slog.String("telemetryId", dItm.Header.TelemetryId),
		)
	}

	return
}

type TelemetryDataRowHandler interface {
	// TelemetryDataRow is a superset of TableRow
	TableRowHandler

	// Initialise the row fields
	Init(dItm *telemetrylib.TelemetryDataItem, bHdr *telemetrylib.TelemetryBundleHeader, tagSetId int64) error
}

type TelemetryDataCommon struct {
	TableRowCommon

	// public table fields
	Id            int64  `json:"id"`
	ClientId      int64  `json:"clientId"`
	CustomerId    string `json:"customerId"`
	TelemetryId   string `json:"telemetryId"`
	TelemetryType string `json:"telemetryType"`
	Timestamp     string `json:"timestamp"`
	TagSetId      int64  `json:"tagSetId"`
}

func (t *TelemetryDataCommon) SetupDB(db *DbConnection) (err error) {
	// save DB reference
	return t.TableRowCommon.SetupDB(db)
}

func (t *TelemetryDataCommon) TableName() string {
	return t.TableRowCommon.TableName()
}

// verify that TelemetryDataCommom provides TableRowCommonHandler interfaces
var _ TableRowCommonHandler = (*TableRowCommon)(nil)

func (t *TelemetryDataCommon) Init(dItm *telemetrylib.TelemetryDataItem, bHdr *telemetrylib.TelemetryBundleHeader, tagSetId int64) {
	// init common telemetry data fields
	t.ClientId = bHdr.BundleClientId
	t.CustomerId = bHdr.BundleCustomerId
	t.TelemetryId = dItm.Header.TelemetryId
	t.TelemetryType = dItm.Header.TelemetryType
	t.Timestamp = dItm.Header.TelemetryTimeStamp
	t.TagSetId = tagSetId
}
