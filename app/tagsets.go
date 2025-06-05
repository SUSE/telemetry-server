package app

import (
	"fmt"
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
