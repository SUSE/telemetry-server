package database

import (
	"fmt"
	"strings"
)

type DbTables []*TableSpec

func (dts DbTables) String() string {
	var names []string
	for _, t := range dts {
		names = append(names, t.Name)
	}
	return fmt.Sprintf("DbTables<%s>", strings.Join(names, ","))
}
