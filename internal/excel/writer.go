package excel

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"db2xlsx/internal/db"
	"github.com/xuri/excelize/v2"
)

type Sheet struct {
	Name   string
	Result db.Result
}

func Write(path string, sheets []Sheet) error {
	if len(sheets) == 0 {
		return fmt.Errorf("no sheets to write")
	}
	f := excelize.NewFile()
	defaultSheet := f.GetSheetName(0)

	usedNames := map[string]bool{}
	styles, err := newStyles(f)
	if err != nil {
		return err
	}

	for i, sheet := range sheets {
		name := SafeSheetName(sheet.Name, i+1, usedNames)
		var sheetName string
		if i == 0 {
			sheetName = defaultSheet
			if err := f.SetSheetName(defaultSheet, name); err != nil {
				return err
			}
		} else {
			idx, err := f.NewSheet(name)
			if err != nil {
				return err
			}
			f.SetActiveSheet(idx)
		}
		sheetName = name
		if err := writeSheet(f, sheetName, sheet.Result, styles); err != nil {
			return fmt.Errorf("write sheet %q: %w", sheetName, err)
		}
	}
	f.SetActiveSheet(0)
	return f.SaveAs(path)
}

func SafeSheetName(name string, index int, used map[string]bool) string {
	base := strings.TrimSpace(name)
	if base == "" {
		base = fmt.Sprintf("SQL_%03d", index)
	}
	replacer := strings.NewReplacer(":", "_", "\\", "_", "/", "_", "?", "_", "*", "_", "[", "_", "]", "_")
	base = replacer.Replace(base)
	base = strings.TrimSpace(strings.Trim(base, "'"))
	if base == "" {
		base = fmt.Sprintf("SQL_%03d", index)
	}
	base = truncateRunes(base, 31)
	candidate := base
	for n := 2; used[candidate]; n++ {
		suffix := fmt.Sprintf("_%d", n)
		candidate = truncateRunes(base, 31-len(suffix)) + suffix
	}
	used[candidate] = true
	return candidate
}

type styles struct {
	header   int
	text     int
	integer  int
	number   int
	date     int
	dateTime int
	bool     int
}

func newStyles(f *excelize.File) (styles, error) {
	header, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"4472C4"}, Pattern: 1},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})
	if err != nil {
		return styles{}, err
	}
	text, err := f.NewStyle(&excelize.Style{NumFmt: 49})
	if err != nil {
		return styles{}, err
	}
	integer, err := f.NewStyle(&excelize.Style{NumFmt: 1})
	if err != nil {
		return styles{}, err
	}
	number, err := f.NewStyle(&excelize.Style{NumFmt: 4})
	if err != nil {
		return styles{}, err
	}
	date, err := f.NewStyle(&excelize.Style{CustomNumFmt: stringPtr("yyyy-mm-dd")})
	if err != nil {
		return styles{}, err
	}
	dateTime, err := f.NewStyle(&excelize.Style{CustomNumFmt: stringPtr("yyyy-mm-dd hh:mm:ss")})
	if err != nil {
		return styles{}, err
	}
	boolStyle, err := f.NewStyle(&excelize.Style{NumFmt: 49})
	if err != nil {
		return styles{}, err
	}
	return styles{header: header, text: text, integer: integer, number: number, date: date, dateTime: dateTime, bool: boolStyle}, nil
}

func writeSheet(f *excelize.File, sheetName string, result db.Result, styles styles) error {
	widths := make([]int, len(result.Columns))
	for i, col := range result.Columns {
		cell, err := excelize.CoordinatesToCellName(i+1, 1)
		if err != nil {
			return err
		}
		if err := f.SetCellValue(sheetName, cell, col.Name); err != nil {
			return err
		}
		widths[i] = displayWidth(col.Name)
	}
	if len(result.Columns) > 0 {
		end, err := excelize.CoordinatesToCellName(len(result.Columns), 1)
		if err != nil {
			return err
		}
		if err := f.SetCellStyle(sheetName, "A1", end, styles.header); err != nil {
			return err
		}
		if err := f.SetPanes(sheetName, &excelize.Panes{
			Freeze:      true,
			Split:       false,
			XSplit:      0,
			YSplit:      1,
			TopLeftCell: "A2",
			ActivePane:  "bottomLeft",
		}); err != nil {
			return err
		}
	}

	for rowIdx, row := range result.Rows {
		for colIdx, value := range row {
			cell, err := excelize.CoordinatesToCellName(colIdx+1, rowIdx+2)
			if err != nil {
				return err
			}
			kind := inferKind(result.Columns[colIdx].DatabaseTypeName, value)
			if err := setCell(f, sheetName, cell, value, kind); err != nil {
				return err
			}
			if err := f.SetCellStyle(sheetName, cell, cell, styleFor(styles, kind)); err != nil {
				return err
			}
			widths[colIdx] = max(widths[colIdx], displayWidth(fmt.Sprint(value)))
		}
	}

	for i, width := range widths {
		col, err := excelize.ColumnNumberToName(i + 1)
		if err != nil {
			return err
		}
		adjusted := min(max(width+2, 10), 60)
		if err := f.SetColWidth(sheetName, col, col, float64(adjusted)); err != nil {
			return err
		}
	}
	return nil
}

type valueKind int

const (
	kindText valueKind = iota
	kindInteger
	kindNumber
	kindDate
	kindDateTime
	kindBool
)

func inferKind(databaseType string, value any) valueKind {
	switch value.(type) {
	case nil:
		return kindText
	case time.Time:
		t := value.(time.Time)
		if t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 && t.Nanosecond() == 0 {
			return kindDate
		}
		return kindDateTime
	case bool:
		return kindBool
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return kindInteger
	case float32, float64:
		return kindNumber
	}
	upper := strings.ToUpper(databaseType)
	switch {
	case strings.Contains(upper, "DATE") && !strings.Contains(upper, "TIME"):
		return kindDate
	case strings.Contains(upper, "TIME"):
		return kindDateTime
	case strings.Contains(upper, "BOOL"):
		return kindBool
	case strings.Contains(upper, "INT"):
		return kindInteger
	case strings.Contains(upper, "NUMBER") || strings.Contains(upper, "NUMERIC") || strings.Contains(upper, "DECIMAL") || strings.Contains(upper, "FLOAT") || strings.Contains(upper, "DOUBLE") || strings.Contains(upper, "REAL"):
		return kindNumber
	default:
		return kindText
	}
}

func setCell(f *excelize.File, sheet, cell string, value any, kind valueKind) error {
	if value == nil {
		return nil
	}
	switch kind {
	case kindInteger:
		if s, ok := value.(string); ok {
			if i, err := strconv.ParseInt(s, 10, 64); err == nil {
				return f.SetCellInt(sheet, cell, int(i))
			}
		}
		return f.SetCellValue(sheet, cell, value)
	case kindNumber:
		if s, ok := value.(string); ok {
			if n, err := strconv.ParseFloat(s, 64); err == nil && !math.IsNaN(n) && !math.IsInf(n, 0) {
				return f.SetCellFloat(sheet, cell, n, -1, 64)
			}
		}
		return f.SetCellValue(sheet, cell, value)
	case kindDate, kindDateTime:
		return f.SetCellValue(sheet, cell, value)
	case kindBool:
		if s, ok := value.(string); ok {
			if b, err := strconv.ParseBool(strings.ToLower(s)); err == nil {
				return f.SetCellBool(sheet, cell, b)
			}
		}
		return f.SetCellValue(sheet, cell, value)
	default:
		return f.SetCellStr(sheet, cell, fmt.Sprint(value))
	}
}

func styleFor(styles styles, kind valueKind) int {
	switch kind {
	case kindInteger:
		return styles.integer
	case kindNumber:
		return styles.number
	case kindDate:
		return styles.date
	case kindDateTime:
		return styles.dateTime
	case kindBool:
		return styles.bool
	default:
		return styles.text
	}
}

func displayWidth(value string) int {
	width := 0
	for _, r := range value {
		if r > 127 {
			width += 2
		} else {
			width++
		}
	}
	return width
}

func truncateRunes(value string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= maxLen {
		return value
	}
	return string(runes[:maxLen])
}

func stringPtr(value string) *string {
	return &value
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
