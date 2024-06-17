package app

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
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

const tagSetsTableColumns = `(
	id INTEGER NOT NULL PRIMARY KEY,
	tagSet VARCHAR NOT NULL UNIQUE
)`

type TagSetRow struct {
	Id     int64  `json:"id"`
	TagSet string `json:"tagSet"`
}

func (t *TagSetRow) String() string {
	bytes, _ := json.Marshal(t)
	return string(bytes)
}

func (t *TagSetRow) Exists(DB *sql.DB) bool {
	row := DB.QueryRow(`SELECT id FROM tagSets WHERE tagSet = ?`, t.TagSet)
	if err := row.Scan(&t.Id); err != nil {
		if err != sql.ErrNoRows {
			log.Printf("ERR: failed when checking for existence of tagSet %q: %s", t.TagSet, err.Error())
		}
		return false
	}
	return true
}

func (t *TagSetRow) Insert(DB *sql.DB) (err error) {
	res, err := DB.Exec(
		`INSERT INTO tagSets(tagSet) VALUES(?)`,
		t.TagSet,
	)
	if err != nil {
		log.Printf("ERR: failed to add tagSet %q: %s", t.TagSet, err.Error())
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Printf("ERR: failed to retrieve id for inserted tagSet %q: %s", t.TagSet, err.Error())
		return err
	}
	t.Id = id

	return
}
