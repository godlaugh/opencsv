package sql

import (
	"database/sql"
	"fmt"
	"opencsv/models"
	"strings"
	"sync"

	_ "modernc.org/sqlite"
)

// cachedDB holds a built in-memory SQLite table for a session, tagged with the
// session's DataVersion so we can detect when the underlying data changed.
type cachedDB struct {
	db      *sql.DB
	version uint64
}

var (
	cacheMu    sync.Mutex
	cache      = map[string]*cachedDB{}
	buildLocks sync.Map // sessionID -> *sync.Mutex (avoid concurrent rebuilds)
)

func sessionLock(id string) *sync.Mutex {
	m, _ := buildLocks.LoadOrStore(id, &sync.Mutex{})
	return m.(*sync.Mutex)
}

// Execute runs the query against the session's SQLite table, building (and
// caching) that table on first use and reusing it until the data changes.
func Execute(sess *models.FileSession, query string) (columns []string, rows [][]string, err error) {
	db, err := getOrBuild(sess)
	if err != nil {
		return nil, nil, err
	}

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

// getOrBuild returns a cached SQLite DB for the session, rebuilding it only if
// the session's data changed (DataVersion bumped) since it was last built.
func getOrBuild(sess *models.FileSession) (*sql.DB, error) {
	lock := sessionLock(sess.ID)
	lock.Lock()
	defer lock.Unlock()

	cacheMu.Lock()
	cd := cache[sess.ID]
	cacheMu.Unlock()

	if cd != nil && cd.version == sess.DataVersion {
		return cd.db, nil // fresh — reuse
	}
	if cd != nil {
		cd.db.Close() // stale — drop and rebuild
	}

	db, err := build(sess)
	if err != nil {
		return nil, err
	}
	cacheMu.Lock()
	cache[sess.ID] = &cachedDB{db: db, version: sess.DataVersion}
	cacheMu.Unlock()
	return db, nil
}

// build loads the session data into a fresh in-memory SQLite table.
func build(sess *models.FileSession) (*sql.DB, error) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("cannot open sqlite: %w", err)
	}
	// A ":memory:" database is scoped to a single connection. Pin the pool to
	// one connection so the table persists across cached queries.
	db.SetMaxOpenConns(1)
	// Speed up the bulk load (safe for a throwaway in-memory table).
	_, _ = db.Exec("PRAGMA journal_mode=OFF")
	_, _ = db.Exec("PRAGMA synchronous=OFF")
	_, _ = db.Exec("PRAGMA temp_store=MEMORY")

	tableName := "data"
	colCount := len(sess.Columns)
	colDefs := make([]string, colCount)
	colNames := make([]string, colCount)
	for i, c := range sess.Columns {
		name := sanitizeColName(c.Name, i)
		colDefs[i] = fmt.Sprintf(`"%s" TEXT`, name)
		colNames[i] = fmt.Sprintf(`"%s"`, name)
	}
	createSQL := fmt.Sprintf(`CREATE TABLE "%s" (%s)`, tableName, strings.Join(colDefs, ", "))
	if _, err = db.Exec(createSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("create table error: %w", err)
	}

	if len(sess.Rows) > 0 && colCount > 0 {
		// One prepared statement reused for every row: modernc compiles the SQL
		// once, which is faster than re-parsing a big multi-row INSERT per batch.
		placeholders := strings.TrimSuffix(strings.Repeat("?,", colCount), ",")
		insertSQL := fmt.Sprintf(`INSERT INTO "%s" (%s) VALUES (%s)`,
			tableName, strings.Join(colNames, ", "), placeholders)

		tx, _ := db.Begin()
		stmt, err := tx.Prepare(insertSQL)
		if err != nil {
			_ = tx.Rollback()
			db.Close()
			return nil, fmt.Errorf("prepare insert error: %w", err)
		}
		vals := make([]interface{}, colCount)
		for _, row := range sess.Rows {
			for i := range vals {
				if i < len(row) {
					vals[i] = row[i]
				} else {
					vals[i] = ""
				}
			}
			if _, err = stmt.Exec(vals...); err != nil {
				stmt.Close()
				_ = tx.Rollback()
				db.Close()
				return nil, fmt.Errorf("insert error: %w", err)
			}
		}
		stmt.Close()
		_ = tx.Commit()
	}

	return db, nil
}

// Invalidate drops and closes any cached table for a session (call on close).
func Invalidate(id string) {
	cacheMu.Lock()
	cd := cache[id]
	delete(cache, id)
	cacheMu.Unlock()
	if cd != nil {
		cd.db.Close()
	}
	buildLocks.Delete(id)
}

func sanitizeColName(name string, index int) string {
	if strings.TrimSpace(name) == "" {
		return fmt.Sprintf("col_%d", index)
	}
	return name
}
