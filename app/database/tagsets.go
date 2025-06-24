package database

import (
	"database/sql"
	"encoding/json"
	"log/slog"
)

var tagSetsTableSpec = TableSpec{
	Name: "tagSets",
	Columns: []TableSpecColumn{
		{Name: "id", Type: "INTEGER", PrimaryKey: true, Identity: true},
		{Name: "tagSet", Type: "VARCHAR"},
	},
}

func GetTagSetsTableSpec() *TableSpec {
	return &tagSetsTableSpec
}

type TagSetRow struct {
	TableRowCommon

	Id     int64  `json:"id"`
	TagSet string `json:"tagSet"`
}

func (t *TagSetRow) Init(tagSet string) {
	t.TagSet = tagSet
}

func (t *TagSetRow) String() string {
	bytes, _ := json.Marshal(t)
	return string(bytes)
}

func (t *TagSetRow) SetupDB(adb *AppDb) error {
	t.SetTableSpec(GetTagSetsTableSpec())
	return t.TableRowCommon.SetupDB(adb)
}

func (t *TagSetRow) RowId() int64 {
	return t.Id
}

func (t *TagSetRow) Exists() bool {
	stmt, err := t.SelectStmt(
		[]string{
			"id",
		},
		[]string{
			"tagSet",
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

	row := t.DB().QueryRow(stmt, t.TagSet)
	if err := row.Scan(&t.Id); err != nil {
		if err != sql.ErrNoRows {
			slog.Error("tagSet existence check failed", slog.String("tagSet", t.TagSet), slog.String("error", err.Error()))
		}
		return false
	}
	return true
}

func (t *TagSetRow) Insert() (err error) {
	stmt, err := t.InsertStmt(
		[]string{
			"tagSet",
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
		t.TagSet,
	)
	if err = row.Scan(
		&t.Id,
	); err != nil {
		slog.Error(
			"insert failed",
			slog.String("table", t.TableName()),
			slog.String("tagSet", t.TagSet),
			slog.String("error", err.Error()),
		)
	}

	return
}

func (t *TagSetRow) Update() (err error) {
	stmt, err := t.UpdateStmt(
		[]string{
			"tagSet",
		},
		[]string{
			"id",
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
		t.TagSet,
		t.Id,
	)
	if err != nil {
		slog.Error(
			"update failed",
			slog.String("table", t.TableName()),
			slog.String("tagSet", t.TagSet),
			slog.String("error", err.Error()),
		)
	}

	return
}

func (t *TagSetRow) Delete() (err error) {
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

// verify that TagSetRow conforms to the TableRowHandler interface
var _ TableRowHandler = (*TagSetRow)(nil)
