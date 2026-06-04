package excel

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

// Import reads the first sheet of an xlsx file and returns headers + rows
func Import(filePath string, hasHeader bool) (headers []string, rows [][]string, err error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot open xlsx: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, nil, fmt.Errorf("xlsx has no sheets")
	}

	all, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, nil, fmt.Errorf("cannot read sheet: %w", err)
	}

	if len(all) == 0 {
		return nil, nil, nil
	}

	if hasHeader {
		headers = all[0]
		rows = all[1:]
	} else {
		colCount := len(all[0])
		headers = make([]string, colCount)
		for i := range headers {
			headers[i] = fmt.Sprintf("Column %d", i+1)
		}
		rows = all
	}

	return headers, rows, nil
}

// Export writes headers + rows to an xlsx file
func Export(filePath string, headers []string, rows [][]string) error {
	f, err := build(headers, rows)
	if err != nil {
		return err
	}
	return f.SaveAs(filePath)
}

// ExportBytes serializes headers + rows as an xlsx document in memory
func ExportBytes(headers []string, rows [][]string) ([]byte, error) {
	f, err := build(headers, rows)
	if err != nil {
		return nil, err
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func build(headers []string, rows [][]string) (*excelize.File, error) {
	f := excelize.NewFile()
	sheet := "Sheet1"

	// Write header
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := f.SetCellValue(sheet, cell, h); err != nil {
			return nil, err
		}
	}

	// Write rows
	for ri, row := range rows {
		for ci, val := range row {
			cell, _ := excelize.CoordinatesToCellName(ci+1, ri+2)
			if err := f.SetCellValue(sheet, cell, val); err != nil {
				return nil, err
			}
		}
	}

	// Style the header row
	style, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"DAEEF3"}, Pattern: 1},
	})
	if len(headers) > 0 {
		endCell, _ := excelize.CoordinatesToCellName(len(headers), 1)
		_ = f.SetCellStyle(sheet, "A1", endCell, style)
	}

	return f, nil
}
