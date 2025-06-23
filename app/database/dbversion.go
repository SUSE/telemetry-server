package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-playground/validator/v10"
)

var dvVersionTableSpec = TableSpec{
	Name: "dbVersion",
	Columns: []TableSpecColumn{
		{Name: "id", Type: "INTEGER", PrimaryKey: true, Identity: true},
		{Name: "version", Type: "VARCHAR", Unique: true},
		{Name: "date", Type: "VARCHAR", Nullable: true},
	},
}

func GetDbVersionTableSpec() *TableSpec {
	return &dvVersionTableSpec
}

type DbVersionRowHandler interface {
	TableRowHandler

	Init(version, date string)
}

type DbVersionRow struct {
	TableRowCommon

	Id      int64  `json:"id"`
	Version string `json:"version" validate:"required,uuid|uuid_rfc4122"`
	Date    string `json:"date" validate:"required,datetime=2006-01-02,dateonly"`
}

func NewDbVersionRow(version, date string) (dv *DbVersionRow) {
	dv = &DbVersionRow{}
	dv.Init(version, date)
	return
}

func (dv *DbVersionRow) Init(version, date string) {
	dv.Version = version
	dv.Date = date
}

func dateOnlyValidator(fl validator.FieldLevel) bool {
	dateOnly := fl.Field().String()
	date, err := time.Parse("2006-01-02", dateOnly)
	if err != nil {
		return false
	}

	// a valid dateOnly value should have zero for hours, minutes and seconds
	return ((date.Hour() == 0) &&
		(date.Minute() == 0) &&
		(date.Second() == 0))
}

func (dv *DbVersionRow) Validate() (err error) {
	validate := validator.New()
	validate.RegisterValidation("dateonly", dateOnlyValidator)

	if err = validate.Struct(dv); err != nil {
		err = fmt.Errorf("dbVersion validation check failed: %w", err)
	}

	return
}

func (dv *DbVersionRow) String() string {
	bytes, _ := json.Marshal(dv)
	return string(bytes)
}

func (dv *DbVersionRow) RowId() int64 {
	return dv.Id
}

func (dv *DbVersionRow) SetupDB(adb *AppDb) (err error) {
	dv.SetTableSpec(GetDbVersionTableSpec())
	return dv.TableRowCommon.SetupDB(adb)
}

func (dv *DbVersionRow) Exists() bool {
	// generate a SELECT statement
	stmt, err := dv.SelectStmt(
		// columns to be retrieved
		[]string{
			"id",
			"date",
		},
		// columns to match against
		[]string{
			"version",
		},
		SelectOpts{},
	)
	if err != nil {
		slog.Error(
			"exists statement generation failed",
			slog.String("table", dv.TableName()),
			slog.String("error", err.Error()),
		)
		panic(err)
	}

	// attempt to retrieve a matching row for the provided version
	row := dv.DB().QueryRow(
		stmt,
		// match values
		dv.Version,
	)
	if err := row.Scan(&dv.Id, &dv.Date); err != nil {
		if err != sql.ErrNoRows {
			slog.Error(
				"row.Scan() failed for existence of version",
				slog.String("version", dv.Version),
				slog.String("table", dv.TableName()),
				slog.String("error", err.Error()),
			)
		}
		return false
	}

	return true
}

func (dv *DbVersionRow) LastRow() (err error) {
	// generate a SELECT statement
	stmt, err := dv.SelectStmt(
		// columns to be retrieved
		[]string{
			"id",
			"version",
			"date",
		},
		// columns to match against
		[]string{},
		SelectOpts{
			OrderBy:    "id",
			Descending: true,
			Limit:      1,
		},
	)
	if err != nil {
		slog.Error(
			"exists statement generation failed",
			slog.String("table", dv.TableName()),
			slog.String("error", err.Error()),
		)
		panic(err)
	}

	// attempt to retrieve last row
	row := dv.DB().QueryRow(
		stmt,
	)
	if err = row.Scan(&dv.Id, &dv.Version, &dv.Date); err != nil {
		if err == sql.ErrNoRows {
			// ensure version is empty and clear the error
			dv.Version = ""
			err = nil
		} else {
			slog.Error(
				"row.Scan() failed for last row",
				slog.String("version", dv.Version),
				slog.String("table", dv.TableName()),
				slog.String("error", err.Error()),
			)
		}
	}

	return
}

func (dv *DbVersionRow) Insert() (err error) {
	// generate an INSERT statement
	stmt, err := dv.InsertStmt(
		// inserted fields
		[]string{
			"version",
			"date",
		},
		// returning field
		"id",
	)
	if err != nil {
		slog.Error(
			"insert statement generation failed",
			slog.String("table", dv.TableName()),
			slog.String("error", err.Error()),
		)
		panic(err)
	}

	// insert a row, querying to retrieve the inserted row's id
	row := dv.DB().QueryRow(
		stmt,
		// insert values
		dv.Version,
		dv.Date,
	)
	if err = row.Scan(
		// returning value
		&dv.Id,
	); err != nil {
		slog.Error(
			"insert failed",
			slog.String("version", dv.Version),
			slog.String("date", dv.Date),
			slog.String("table", dv.TableName()),
			slog.String("error", err.Error()),
		)
	}
	return
}

func (dv *DbVersionRow) Update() (err error) {
	// generate an UPDATE statement
	stmt, err := dv.UpdateStmt(
		// updated fields
		[]string{
			"version",
			"date",
		},
		// match fields
		[]string{
			"id",
		},
	)
	if err != nil {
		slog.Error(
			"update statement generation failed",
			slog.String("table", dv.TableName()),
			slog.String("error", err.Error()),
		)
		panic(err)
	}

	// update the row
	_, err = dv.DB().Exec(
		stmt,
		// update values
		dv.Version,
		dv.Date,
		// match values
		dv.Id,
	)
	if err != nil {
		slog.Error(
			"update failed",
			slog.String("version", dv.Version),
			slog.String("date", dv.Date),
			slog.Int64("id", dv.Id),
			slog.String("table", dv.TableName()),
			slog.String("error", err.Error()),
		)
	}
	return
}

func (dv *DbVersionRow) Delete() (err error) {
	// generate a DELETE statement
	stmt, err := dv.DeleteStmt(
		// match fields
		[]string{
			"id",
		},
	)
	if err != nil {
		slog.Error(
			"delete generation failed",
			slog.String("table", dv.TableName()),
			slog.String("error", err.Error()),
		)
		panic(err)
	}

	// delete the row
	_, err = dv.DB().Exec(
		stmt,
		// match values
		dv.Id,
	)
	if err != nil {
		slog.Error(
			"delete failed",
			slog.Int64("id", dv.Id),
			slog.String("table", dv.TableName()),
			slog.String("error", err.Error()),
		)
	}
	return
}

// verify that DbVersionRow conforms to the TableRowHandler interface
var _ DbVersionRowHandler = (*DbVersionRow)(nil)
