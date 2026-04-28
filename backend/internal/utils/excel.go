package utils

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

var TemplateHeaders = []string{"日期", "地点名称", "经度", "纬度", "活动描述"}

type ExcelImportRow struct {
	Date        string
	Location    string
	Longitude   float64
	Latitude    float64
	Description string
}

func BuildImportTemplate() ([]byte, error) {
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)

	for i, h := range TemplateHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := f.SetCellValue(sheet, cell, h); err != nil {
			return nil, err
		}
	}

	sample := []interface{}{"2025-04-01", "大理古城", 100.267, 25.693, "逛古城"}
	for i, v := range sample {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		if err := f.SetCellValue(sheet, cell, v); err != nil {
			return nil, err
		}
	}

	_ = f.SetColWidth(sheet, "A", "A", 14)
	_ = f.SetColWidth(sheet, "B", "B", 18)
	_ = f.SetColWidth(sheet, "C", "D", 12)
	_ = f.SetColWidth(sheet, "E", "E", 36)

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func ParseImportExcel(r io.Reader) ([]ExcelImportRow, error) {
	f, err := excelize.OpenReader(r)
	if err != nil {
		return nil, fmt.Errorf("open excel: %w", err)
	}
	defer func() { _ = f.Close() }()

	sheet := f.GetSheetName(0)
	if sheet == "" {
		return nil, fmt.Errorf("excel has no sheets")
	}

	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("read rows: %w", err)
	}
	if len(rows) < 2 {
		return []ExcelImportRow{}, nil
	}

	out := make([]ExcelImportRow, 0, len(rows)-1)
	for i := 1; i < len(rows); i++ {
		rowNum := i + 1
		row := rows[i]
		date := strings.TrimSpace(safeCell(row, 0))
		location := strings.TrimSpace(safeCell(row, 1))
		lonRaw := strings.TrimSpace(safeCell(row, 2))
		latRaw := strings.TrimSpace(safeCell(row, 3))
		desc := strings.TrimSpace(safeCell(row, 4))

		if date == "" && location == "" && lonRaw == "" && latRaw == "" && desc == "" {
			continue
		}
		if date == "" {
			return nil, fmt.Errorf("row %d: 日期不能为空", rowNum)
		}
		if location == "" {
			return nil, fmt.Errorf("row %d: 地点名称不能为空", rowNum)
		}

		normalizedDate, err := normalizeDate(date)
		if err != nil {
			return nil, fmt.Errorf("row %d: 日期格式错误，需 YYYY-MM-DD", rowNum)
		}

		lon, err := parseOptionalFloat(lonRaw)
		if err != nil {
			return nil, fmt.Errorf("row %d: 经度格式错误", rowNum)
		}
		lat, err := parseOptionalFloat(latRaw)
		if err != nil {
			return nil, fmt.Errorf("row %d: 纬度格式错误", rowNum)
		}

		out = append(out, ExcelImportRow{
			Date:        normalizedDate,
			Location:    location,
			Longitude:   lon,
			Latitude:    lat,
			Description: desc,
		})
	}

	return out, nil
}

func safeCell(row []string, idx int) string {
	if idx < len(row) {
		return row[idx]
	}
	return ""
}

func parseOptionalFloat(raw string) (float64, error) {
	if raw == "" {
		return 0, nil
	}
	return strconv.ParseFloat(raw, 64)
}

func normalizeDate(raw string) (string, error) {
	layouts := []string{"2006-01-02", "2006/01/02", "2006.01.02"}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.Format("2006-01-02"), nil
		}
	}
	return "", fmt.Errorf("invalid date")
}
