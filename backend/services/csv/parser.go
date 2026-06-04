package csv

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// ParseFile reads a CSV file with the given configuration
// Returns headers (if hasHeader) and all data rows.
func ParseFile(filePath string, delimiter, quote, encoding string, hasHeader bool) (headers []string, rows [][]string, err error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot open file: %w", err)
	}
	defer f.Close()

	var reader io.Reader = f

	// Apply encoding transform
	reader, err = applyEncoding(reader, encoding)
	if err != nil {
		return nil, nil, fmt.Errorf("encoding error: %w", err)
	}

	r := csv.NewReader(bufio.NewReaderSize(reader, 1<<20)) // 1MB buffer
	r.LazyQuotes = true
	r.TrimLeadingSpace = false
	r.FieldsPerRecord = -1 // allow variable field count

	if len(delimiter) > 0 {
		r.Comma = rune(delimiter[0])
	}
	if len(quote) > 0 {
		// standard csv.Reader uses '"' by default; we can override
		_ = quote // Go's csv.Reader doesn't expose Quote field; we handle common cases
	}

	all, err := r.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("csv parse error: %w", err)
	}

	if len(all) == 0 {
		return nil, nil, nil
	}

	if hasHeader {
		headers = all[0]
		rows = all[1:]
	} else {
		// Generate numeric headers
		colCount := len(all[0])
		headers = make([]string, colCount)
		for i := range headers {
			headers[i] = fmt.Sprintf("Column %d", i+1)
		}
		rows = all
	}

	return headers, rows, nil
}

// WriteFile writes headers + rows to a CSV file
func WriteFile(filePath string, headers []string, rows [][]string, delimiter, encoding string) error {
	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("cannot create file: %w", err)
	}
	defer f.Close()

	return WriteCSV(f, headers, rows, delimiter, encoding)
}

// WriteCSV serializes headers + rows as CSV into the given writer
func WriteCSV(writer io.Writer, headers []string, rows [][]string, delimiter, encoding string) error {
	// Apply encoding if needed
	if strings.ToLower(encoding) == "utf-8-bom" {
		if _, err := writer.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
			return err
		}
	}

	w := csv.NewWriter(writer)
	if len(delimiter) > 0 {
		w.Comma = rune(delimiter[0])
	}

	// Write header
	if len(headers) > 0 {
		if err := w.Write(headers); err != nil {
			return err
		}
	}

	// Write rows
	for _, row := range rows {
		if err := w.Write(row); err != nil {
			return err
		}
	}

	w.Flush()
	return w.Error()
}

// DetectDelimiter tries to detect the most likely delimiter from file content
func DetectDelimiter(filePath string) string {
	f, err := os.Open(filePath)
	if err != nil {
		return ","
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Scan()
	line := scanner.Text()

	candidates := []struct {
		char  string
		count int
	}{
		{",", strings.Count(line, ",")},
		{"\t", strings.Count(line, "\t")},
		{";", strings.Count(line, ";")},
		{"|", strings.Count(line, "|")},
	}

	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.count > best.count {
			best = c
		}
	}
	return best.char
}

func applyEncoding(r io.Reader, encoding string) (io.Reader, error) {
	switch strings.ToLower(encoding) {
	case "utf-8", "utf8", "":
		return r, nil
	case "utf-8-bom":
		// Skip BOM if present
		br := bufio.NewReader(r)
		b, _ := br.Peek(3)
		if len(b) == 3 && b[0] == 0xEF && b[1] == 0xBB && b[2] == 0xBF {
			_, _ = br.Discard(3)
		}
		return br, nil
	case "gbk", "gb2312", "gb18030":
		return transform.NewReader(r, charmap.Windows1252.NewDecoder()), nil
	case "utf-16", "utf-16le":
		return transform.NewReader(r, unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewDecoder()), nil
	case "utf-16be":
		return transform.NewReader(r, unicode.UTF16(unicode.BigEndian, unicode.UseBOM).NewDecoder()), nil
	case "latin-1", "iso-8859-1":
		return transform.NewReader(r, charmap.ISO8859_1.NewDecoder()), nil
	default:
		return r, nil
	}
}
