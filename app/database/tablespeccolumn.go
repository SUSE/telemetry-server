package database

import (
	"strings"
)

type TableSpecColumn struct {
	Name       string
	Type       string
	Nullable   bool
	Default    string
	PrimaryKey bool
	Identity   bool
	Unique     bool
}

func (c *TableSpecColumn) Create(db *DbConnection) string {
	elements := []string{
		c.Name, c.Type,
	}
	if !c.Nullable {
		elements = append(elements, "NOT")
	}
	elements = append(elements, "NULL")
	if len(c.Default) > 0 {
		elements = append(elements, "DEFAULT", c.Default)
	}
	if c.PrimaryKey {
		elements = append(elements, "PRIMARY", "KEY")
	}
	if c.Unique {
		elements = append(elements, "UNIQUE")
	}
	if c.Identity {
		switch {
		case db.dbMgr.Type().IsPostgres():
			elements = append(elements, "GENERATED", "BY", "DEFAULT", "AS", "IDENTITY")
		}
	}
	return strings.Join(elements, " ")
}
