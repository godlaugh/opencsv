package session

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"opencsv/models"
	csvparser "opencsv/services/csv"
	sqlengine "opencsv/services/sql"
)

// Store manages all open file sessions
type Store struct {
	mu       sync.RWMutex
	sessions map[string]*models.FileSession
}

var Global = &Store{
	sessions: make(map[string]*models.FileSession),
}

// Open loads a CSV file into memory and returns the session
func (s *Store) Open(filePath string, config models.CsvConfig) (*models.FileSession, error) {
	delimiter := config.Delimiter
	if delimiter == "" {
		delimiter = csvparser.DetectDelimiter(filePath)
		config.Delimiter = delimiter
	}

	headers, rows, err := csvparser.ParseFile(filePath, delimiter, config.Quote, config.Encoding, config.HasHeader)
	if err != nil {
		return nil, err
	}

	id := generateID()
	columns := make([]models.Column, len(headers))
	for i, h := range headers {
		columns[i] = models.Column{Index: i, Name: h}
	}

	sess := &models.FileSession{
		ID:        id,
		FilePath:  filePath,
		Config:    config,
		Columns:   columns,
		Rows:      rows,
		TotalRows: len(rows),
		Modified:  false,
	}

	s.mu.Lock()
	s.sessions[id] = sess
	s.mu.Unlock()

	return sess, nil
}

// CreateSession creates a session from pre-parsed data (e.g., Excel import)
func (s *Store) CreateSession(filePath string, config models.CsvConfig, headers []string, rows [][]string) *models.FileSession {
	id := generateID()
	columns := make([]models.Column, len(headers))
	for i, h := range headers {
		columns[i] = models.Column{Index: i, Name: h}
	}
	sess := &models.FileSession{
		ID:        id,
		FilePath:  filePath,
		Config:    config,
		Columns:   columns,
		Rows:      rows,
		TotalRows: len(rows),
		Modified:  false,
	}
	s.mu.Lock()
	s.sessions[id] = sess
	s.mu.Unlock()
	return sess
}

// Get retrieves a session by ID
func (s *Store) Get(id string) (*models.FileSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", id)
	}
	return sess, nil
}

// Close removes a session from memory
func (s *Store) Close(id string) {
	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()
	sqlengine.Invalidate(id) // drop the cached SQLite table
}

// Save writes the session data back to disk
func (s *Store) Save(id, filePath string) error {
	sess, err := s.Get(id)
	if err != nil {
		return err
	}

	if filePath == "" {
		filePath = sess.FilePath
	}

	// Auto-backup
	if filePath == sess.FilePath {
		backupPath := filePath + ".bak"
		_ = copyFile(filePath, backupPath)
	}

	headers := make([]string, len(sess.Columns))
	for i, c := range sess.Columns {
		headers[i] = c.Name
	}

	delimiter := sess.Config.Delimiter
	if delimiter == "" {
		delimiter = ","
	}

	err = csvparser.WriteFile(filePath, headers, sess.Rows, delimiter, sess.Config.Encoding)
	if err != nil {
		return err
	}

	s.mu.Lock()
	sess.FilePath = filePath
	sess.Modified = false
	s.mu.Unlock()

	return nil
}

// GetRows returns a slice of rows from offset with limit count
func (s *Store) GetRows(id string, offset, limit int) ([][]string, int, error) {
	sess, err := s.Get(id)
	if err != nil {
		return nil, 0, err
	}

	total := len(sess.Rows)
	if offset >= total {
		return [][]string{}, total, nil
	}

	end := offset + limit
	if end > total {
		end = total
	}

	return sess.Rows[offset:end], total, nil
}

// UpdateCells applies cell updates to the session
func (s *Store) UpdateCells(id string, cells []models.Cell) error {
	sess, err := s.Get(id)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, cell := range cells {
		if cell.Row < 0 || cell.Row >= len(sess.Rows) {
			continue
		}
		row := sess.Rows[cell.Row]
		// Expand row if needed
		for len(row) <= cell.Col {
			row = append(row, "")
		}
		row[cell.Col] = cell.Value
		sess.Rows[cell.Row] = row
	}
	sess.Modified = true
	sess.DataVersion++
	return nil
}

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
