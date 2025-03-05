package app

import (
	"database/sql"
	"encoding/json"
	"log/slog"
)

// customers table specification
// The customers table records customer identifiers that have been received
// in telemetry submissions, and which will be referenced by a customerRefId
// foreign key reference in the telemetry data table.
// Additionally a customer identifier entry can be marked as deleted, with
// an associated deletedAt time. Optionally the associated customerId value
// can be cleared or changed to an anonymised value as part of marking an
// entry as deleted.
var customersTableSpec = TableSpec{
	Name: "customers",
	Columns: []TableSpecColumn{
		{Name: "id", Type: "INTEGER", PrimaryKey: true, Identity: true},
		{Name: "customerId", Type: "VARCHAR", Nullable: true},
		{Name: "deleted", Type: "BOOLEAN", Default: "false"},
		{Name: "deletedAt", Type: "VARCHAR", Nullable: true},
	},
}

type CustomersRow struct {
	// include common table row fields
	TableRowCommon

	Id         int64  `json:"id"`
	CustomerId string `json:"customerId"`
	Deleted    bool   `json:"deleted"`
	DeletedAt  string `json:"deletedAt"`
}

func (r *CustomersRow) Init(customerId string) {
	r.CustomerId = customerId
}

func (r *CustomersRow) SetupDB(db *DbConnection) (err error) {
	r.tableSpec = &customersTableSpec
	return r.TableRowCommon.SetupDB(db)
}

func (r *CustomersRow) TableName() string {
	return r.TableRowCommon.TableName()
}

func (r *CustomersRow) String() string {
	bytes, _ := json.Marshal(r)
	return string(bytes)
}

func (r *CustomersRow) RowId() int64 {
	return r.Id
}

func (r *CustomersRow) Exists() bool {
	stmt, err := r.SelectStmt(
		// select columns
		[]string{
			"id",
			"deletedAt",
		},
		// match columns
		[]string{
			"customerId",
			"deleted",
		},
		SelectOpts{}, // no special options
	)
	if err != nil {
		slog.Error(
			"exists statement generation failed",
			slog.String("table", r.TableName()),
			slog.String("error", err.Error()),
		)
		panic(err)
	}

	row := r.DB().QueryRow(stmt, r.CustomerId, r.Deleted)
	// if the entry was found, all fields not used to find the entry will have
	// been updated to match what is in the DB
	if err := row.Scan(
		&r.Id,
		&r.DeletedAt,
	); err != nil {
		if err != sql.ErrNoRows {
			slog.Error(
				"check for matching entry failed",
				slog.String("table", r.TableName()),
				slog.Int64("id", r.Id),
				slog.String("error", err.Error()),
			)
		}
		return false
	}
	return true
}

func (r *CustomersRow) IdExists() bool {
	stmt, err := r.SelectStmt(
		// select columns
		[]string{
			"customerId",
			"deleted",
			"deletedAt",
		},
		// match columns
		[]string{
			"id",
		},
		SelectOpts{}, // no special options
	)
	if err != nil {
		slog.Error(
			"exists statement generation failed",
			slog.String("table", r.TableName()),
			slog.String("error", err.Error()),
		)
		panic(err)
	}

	row := r.DB().QueryRow(stmt, r.Id)
	// if the entry was found, all fields not used to find the entry will have
	// been updated to match what is in the DB
	if err := row.Scan(
		&r.CustomerId,
		&r.Deleted,
		&r.DeletedAt,
	); err != nil {
		if err != sql.ErrNoRows {
			slog.Error(
				"check for matching entry failed",
				slog.String("table", r.TableName()),
				slog.Int64("id", r.Id),
				slog.String("error", err.Error()),
			)
		}
		return false
	}
	return true
}

func (r *CustomersRow) Insert() (err error) {
	stmt, err := r.InsertStmt(
		[]string{
			"customerId",
			"deleted",
			"deletedAt",
		},
		"id",
	)
	if err != nil {
		slog.Error(
			"insert statement generation failed",
			slog.String("table", r.TableName()),
			slog.String("error", err.Error()),
		)
		return
	}
	row := r.DB().QueryRow(
		stmt,
		r.CustomerId,
		r.Deleted,
		r.DeletedAt,
	)
	if err = row.Scan(
		&r.Id,
	); err != nil {
		slog.Error(
			"insert failed",
			slog.String("table", r.TableName()),
			slog.String("customerId", r.CustomerId),
			slog.Bool("deleted", r.Deleted),
			slog.String("deletedAt", r.DeletedAt),
			slog.String("error", err.Error()),
		)
	}

	return
}

func (r *CustomersRow) Update() (err error) {
	stmt, err := r.UpdateStmt(
		[]string{
			"customerId",
			"deleted",
			"deletedAt",
		},
		[]string{
			"id",
		},
	)
	if err != nil {
		slog.Error(
			"update statement generation failed",
			slog.String("table", r.TableName()),
			slog.String("error", err.Error()),
		)
		return
	}
	_, err = r.DB().Exec(
		stmt,
		r.CustomerId,
		r.Deleted,
		r.DeletedAt,
		r.Id,
	)
	if err != nil {
		slog.Error(
			"update failed",
			slog.String("table", r.TableName()),
			slog.Int64("id", r.Id),
			slog.String("error", err.Error()),
		)
	}

	return
}

func (r *CustomersRow) Delete() (err error) {
	stmt, err := r.DeleteStmt(
		[]string{
			"id",
		},
	)
	if err != nil {
		slog.Error(
			"delete statement generation failed",
			slog.String("table", r.TableName()),
			slog.String("error", err.Error()),
		)
		return
	}

	_, err = r.DB().Exec(
		stmt,
		r.Id,
	)
	if err != nil {
		slog.Error(
			"delete failed",
			slog.String("table", r.TableName()),
			slog.Int64("id", r.Id),
			slog.String("error", err.Error()),
		)
	}

	return
}

// verify that CustomersRow conforms to the TableRowHandler interface
var _ TableRowHandler = (*CustomersRow)(nil)
