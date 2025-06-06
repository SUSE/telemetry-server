package database

import (
	"strings"
)

type TableSpecForeignKey struct {
	Name             string
	Column           string
	ReferencedTable  string
	ReferencedColumn string
}

func (fk *TableSpecForeignKey) Create(db *DbConnection) string {
	elements := []string{}

	if fk.Name != "" {
		elements = append(elements, "CONSTRAINT", fk.Name)
	}

	elements = append(
		elements,
		"FOREIGN",
		"KEY",
		"("+fk.Column+")",
		"REFERENCES",
		fk.ReferencedTable,
		"("+fk.ReferencedColumn+")",
	)

	return strings.Join(elements, " ")
}
