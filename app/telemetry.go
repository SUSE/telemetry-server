package app

import (
	"log/slog"
	"strings"

	"github.com/SUSE/telemetry-server/app/database"
	telemetrylib "github.com/SUSE/telemetry/pkg/lib"
)

const (
	ANONYMOUS_CUSTOMER_ID = "ANONYMOUS"
)

func (a *App) GetTagSetId(tagSet string) (tagSetId int64, err error) {
	//
	// create an telemetryDb transaction
	//
	tdbTx, err := a.TelemetryDB.StartTx()
	if err != nil {
		return
	}

	// defer a rollback of the telemetryDb transaction
	defer func() {
		a.TelemetryDB.RollbackTx(tdbTx, "AuthenticateClient")
	}()

	tsRow := new(database.TagSetRow)
	tsRow.SetupDB(a.TelemetryDB, tdbTx)

	tsRow.Init(tagSet)

	// if the tagSet entry doesn't already exist, add it
	if !tsRow.Exists() {
		err = tsRow.Insert()
		if err != nil {
			slog.Error("tagSet insert failed", slog.String("tagSet", tsRow.TagSet), slog.String("error", err.Error()))
		} else {
			slog.Info("tagSet added successfully", slog.String("tagSet", tsRow.TagSet), slog.Int64("id", tsRow.Id))
		}
	}

	// commit the transaction
	if err = a.TelemetryDB.CommitTx(tdbTx); err != nil {
		return
	}

	// save the tagSet's reference id if either already present or successfully inserted
	if err == nil {
		tagSetId = tsRow.Id
	}

	return
}

func (a *App) GetCustomerRefId(customerId string) (customerRefId int64, err error) {
	//
	// create an telemetryDb transaction
	//
	tdbTx, err := a.TelemetryDB.StartTx()
	if err != nil {
		return
	}

	// defer a rollback of the telemetryDb transaction
	defer func() {
		a.TelemetryDB.RollbackTx(tdbTx, "AuthenticateClient")
	}()

	cRow := new(database.CustomersRow)
	cRow.SetupDB(a.TelemetryDB, tdbTx)

	// determine actual customer id value to use
	realCustomerId := strings.TrimSpace(customerId)
	switch {
	case strings.ToUpper(realCustomerId) == ANONYMOUS_CUSTOMER_ID:
		fallthrough
	case realCustomerId == "":
		realCustomerId = ANONYMOUS_CUSTOMER_ID
		fallthrough
	case customerId != realCustomerId:
		slog.Debug(
			"Using modified customer id",
			slog.String("original", customerId),
			slog.String("updated", customerId),
		)
	}

	cRow.Init(realCustomerId)

	// if the customerId entry doesn't already exist, add it
	if !cRow.Exists() {
		err = cRow.Insert()
		if err != nil {
			slog.Error("customerId insert failed", slog.String("customerId", cRow.CustomerId), slog.String("error", err.Error()))
		} else {
			slog.Info("customerId added successfully", slog.String("customerId", cRow.CustomerId), slog.Int64("customerRefId", cRow.Id))
		}
	}

	// commit the transaction
	if err = a.TelemetryDB.CommitTx(tdbTx); err != nil {
		return
	}

	// save the customerId's reference id if either already present or successfully inserted
	if err == nil {
		customerRefId = cRow.Id
	}

	return
}

func (a *App) StoreTelemetry(
	dItm *telemetrylib.TelemetryDataItem,
	bHdr *telemetrylib.TelemetryBundleHeader,
) (err error) {
	// generate a tagSet from the bundle and data item tags
	tagSet := createTagSet(append(dItm.Header.TelemetryAnnotations, bHdr.BundleAnnotations...))

	// get the associated tagSet's id, creating a new one if needed
	tagSetId, err := a.GetTagSetId(tagSet)
	if err != nil {
		slog.Error(
			"failed to retrieve tagSetId",
			slog.String("tagSet", tagSet),
			slog.String("err", err.Error()),
		)
		return
	}

	// get the associated tagSet's id, creating a new one if needed
	customerRefId, err := a.GetCustomerRefId(bHdr.BundleCustomerId)
	if err != nil {
		slog.Error(
			"failed to retrieve customerRefId",
			slog.String("customerId", bHdr.BundleCustomerId),
			slog.String("err", err.Error()),
		)
		return
	}

	// store the telemetry
	err = a.StoreTelemetryData(dItm, bHdr, tagSetId, customerRefId)
	if err != nil {
		slog.Error(
			"telemetry store failed",
			slog.String("telemetryId", dItm.Header.TelemetryId),
			slog.String("error", err.Error()))
	}
	return
}

func (a *App) StoreTelemetryData(
	dItm *telemetrylib.TelemetryDataItem,
	bHdr *telemetrylib.TelemetryBundleHeader,
	tagSetId int64,
	customerRefId int64,
) (err error) {
	//
	// create an telemetryDb transaction
	//
	tdbTx, err := a.TelemetryDB.StartTx()
	if err != nil {
		return
	}

	// defer a rollback of the telemetryDb transaction
	defer func() {
		a.TelemetryDB.RollbackTx(tdbTx, "AuthenticateClient")
	}()

	tdRow := new(database.TelemetryDataRow)
	tdRow.SetupDB(a.TelemetryDB, tdbTx)

	err = tdRow.Init(dItm, bHdr, tagSetId, customerRefId)
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

	// commit the transaction
	if err = a.TelemetryDB.CommitTx(tdbTx); err != nil {
		return
	}

	return
}
