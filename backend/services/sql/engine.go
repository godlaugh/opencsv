package sql

import (
	"database/sql"
	"fmt"
	"opencsv/models"
	"strings"

	_ "modernc.org/sqlite"
)

// Execute loads session data into an in-memory SQLite DB and runs the query
func Execute(sess *models.FileSession, query string) (columns []string, rows [][]string, err error) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, nil, fmt.Errorf("cannot open sqlite: %w", err)
	}
	defer db.Close()

	// Create table
	tableName := "data"
	colDefs := make([]string, len(sess.Columns))
	for i, c := range sess.Columns {
		// Sanitize column name
		name := sanitizeColName(c.Name, i)
		colDefs[i] = fmt.Sprintf(`"%s" TEXT`, name)
	}
	createSQL := fmt.Sprintf(`CREATE TABLE "%s" (%s)`, tableName, strings.Join(colDefs, ", "))

	if _, err = db.Exec(createSQL); err != nil {
		return nil, nil, fmt.Errorf("create table error: %w", err)
	}

	// Insert data in batches
	if len(sess.Rows) > 0 {
		colCount := len(sess.Columns)
		placeholders := make([]string, colCount)
		for i := range placeholders {
			placeholders[i] = "?"
		}

		colNames := make([]string, colCount)
		for i, c := range sess.Columns {
			colNames[i] = fmt.Sprintf(`"%s"`, sanitizeColName(c.Name, i))
		}

		insertSQL := fmt.Sprintf(`INSERT INTO "%s" (%s) VALUES (%s)`,
			tableName,
			strings.Join(colNames, ", "),
			strings.Join(placeholders, ", "),
		)

		tx, _ := db.Begin()
		stmt, err := tx.Prepare(insertSQL)
		if err != nil {
			_ = tx.Rollback()
			return nil, nil, fmt.Errorf("prepare insert error: %w", err)
		}

		for _, row := range sess.Rows {
			vals := make([]interface{}, colCount)
			for i := range vals {
				if i < len(row) {
					vals[i] = row[i]
				} else {
					vals[i] = ""
				}
			}
			if _, err = stmt.Exec(vals...); err != nil {
				_ = tx.Rollback()
				return nil, nil, fmt.Errorf("insert error: %w", err)
			}
		}
		stmt.Close()
		_ = tx.Commit()
	}

	// Execute user query
	sqlRows, err := db.Query(query)
	if err != nil {
		return nil, nil, fmt.Errorf("query error: %w", err)
	}
	defer sqlRows.Close()

	columns, err = sqlRows.Columns()
	if err != nil {
		return nil, nil, err
	}

	for sqlRows.Next() {
		vals := make([]interface{}, len(columns))
		ptrs := make([]interface{}, len(columns))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err = sqlRows.Scan(ptrs...); err != nil {
			return nil, nil, err
		}
		row := make([]string, len(columns))
		for i, v := range vals {
			if v == nil {
				row[i] = ""
			} else {
				row[i] = fmt.Sprintf("%v", v)
			}
		}
		rows = append(rows, row)
	}

	return columns, rows, sqlRows.Err()
}

func sanitizeColName(name string, index int) string {
	if strings.TrimSpace(name) == "" {
		return fmt.Sprintf("col_%d", index)
	}
	return name
}
