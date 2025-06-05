package database

import (
	"fmt"
	"log/slog"

	"slices"
)

type TableSpec struct {
	Name        string
	Columns     []TableSpecColumn
	ForeignKeys []TableSpecForeignKey
	Extras      []string

	// TODO: add a sync.Map to hold sql.Prepare()'d statements to
	// eliminate repeated generation of the same SQL statements
}

func (ts *TableSpec) CreateCmd(db *DbConnection) (string, error) {
	table := "CREATE TABLE IF NOT EXISTS " + ts.Name + " ("

	for i, column := range ts.Columns {
		if i > 0 {
			table += ", "
		}
		table += column.Create(db)
	}

	if len(ts.ForeignKeys) > 0 {
		table += ", "
	}

	for i, fk := range ts.ForeignKeys {
		if err := ts.CheckColumnNames([]string{fk.Column}); err != nil {
			return "", fmt.Errorf(
				"foreign key column %q not found in table %q",
				fk.Column,
				ts.Name,
			)
		}

		if i > 0 {
			table += ", "
		}
		table += fk.Create(db)
	}

	if len(ts.Extras) > 0 {
		table += ", "
	}

	for _, extra := range ts.Extras {
		table += " " + extra
	}

	table += ")"

	return table, nil
}

func (ts *TableSpec) ColumnName(ind int) (name string, err error) {
	switch {
	case ind < 0:
		fallthrough
	case ind >= len(ts.Columns):
		err = fmt.Errorf("invalid column index %d for table %q", ind, ts.Name)
	default:
		name = ts.Columns[ind].Name
	}
	return
}

func (ts *TableSpec) CheckColumnNames(colNames []string) (err error) {
	for _, colName := range colNames {
		matchInd := slices.IndexFunc(ts.Columns, func(cs TableSpecColumn) bool {
			return cs.Name == colName
		})
		if matchInd == -1 {
			err = fmt.Errorf("column %q not part of table %q", colName, ts.Name)
			break
		}
	}

	// log a debug message only if an error occurred
	if err != nil {
		slog.Debug("Invalid columns", slog.String("error", err.Error()))
	}
	return err
}
