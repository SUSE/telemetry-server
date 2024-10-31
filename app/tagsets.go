package app

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"strings"
)

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

var tagSetsTableSpec = TableSpec{
	Name: "tagSets",
	Columns: []TableSpecColumn{
		{Name: "id", Type: "INTEGER", PrimaryKey: true, Identity: true},
		{Name: "tagSet", Type: "VARCHAR"},
	},
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

func (t *TagSetRow) SetupDB(db *DbConnection) error {
	t.tableSpec = &tagSetsTableSpec
	return t.TableRowCommon.SetupDB(db)
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
			"exists statement generation failed",
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
