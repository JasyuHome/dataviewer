package services

import (
	"dataviewer/backend/models"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type CSVService struct{}

func NewCSVService() *CSVService {
	return &CSVService{}
}

// ParseResult contains parsed CSV data
type ParseResult struct {
	Headers   []string
	Rows      [][]string
	Preview   [][]string
	Columns   []models.ColumnDef
	RowCount  int
	ColCount  int
	Delimiter string
}

// ParseFile parses a CSV file
func (s *CSVService) ParseFile(filePath string) (*ParseResult, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return s.ParseReader(file)
}

// ParseReader parses CSV from a reader
func (s *CSVService) ParseReader(reader io.Reader) (*ParseResult, error) {
	// Detect delimiter
	delimiter := s.detectDelimiter(reader)
	log.Printf("Detected delimiter: '%c' (code: %d)", delimiter, delimiter)

	// Reset reader
	if seeker, ok := reader.(io.Seeker); ok {
		seeker.Seek(0, io.SeekStart)
	}

	// Read content and remove BOM if present
	content, _ := io.ReadAll(reader)
	// Remove UTF-8 BOM
	if len(content) >= 3 && content[0] == 0xEF && content[1] == 0xBB && content[2] == 0xBF {
		content = content[3:]
	}

	// Create new reader from cleaned content
	reader = strings.NewReader(string(content))

	csvReader := csv.NewReader(reader)
	csvReader.Comma = delimiter
	csvReader.FieldsPerRecord = -1 // Allow variable fields
	csvReader.TrimLeadingSpace = true
	csvReader.LazyQuotes = true    // Allow lazy quotes for malformed CSV
	// Don't use Comment feature as #92 etc are valid data, not comments
	// csvReader.Comment = '#'

	// Read all records
	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	log.Printf("Total records read: %d", len(records))
	for i, rec := range records {
		if i < 3 {
			log.Printf("Record %d: %v (len=%d)", i, rec, len(rec))
		}
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("empty CSV file")
	}

	// First row is headers
	headers := records[0]
	numColumns := len(headers)

	// Process remaining rows, ensuring consistent column count
	rows := make([][]string, 0)
	for _, record := range records[1:] {
		// Skip comment lines or empty lines
		if len(record) == 0 || (len(record) == 1 && strings.TrimSpace(record[0]) == "") {
			continue
		}

		// Pad or trim record to match header column count
		row := make([]string, numColumns)
		for i := 0; i < numColumns; i++ {
			if i < len(record) {
				row[i] = strings.TrimSpace(record[i])
			} else {
				row[i] = "" // Pad with empty string if missing columns
			}
		}
		rows = append(rows, row)
	}

	// Create preview (first 10 rows)
	preview := rows
	if len(preview) > 10 {
		preview = preview[:10]
	}

	// Analyze column types
	columns := s.analyzeColumns(rows, headers)

	return &ParseResult{
		Headers:   headers,
		Rows:      rows,
		Preview:   preview,
		Columns:   columns,
		RowCount:  len(rows),
		ColCount:  numColumns,
		Delimiter: string(delimiter),
	}, nil
}

// detectDelimiter detects the CSV delimiter
func (s *CSVService) detectDelimiter(reader io.Reader) rune {
	// Read first 4KB for detection
	buf := make([]byte, 4096)
	n, _ := reader.Read(buf)
	content := string(buf[:n])

	// Check both Chinese and English commas
	hasChineseComma := strings.Contains(content, "，")
	hasEnglishComma := strings.Contains(content, ",")

	// If both exist, prefer English comma (data rows usually use English)
	if hasEnglishComma {
		return ','
	}

	// If only Chinese comma exists, use it
	if hasChineseComma {
		return '，'
	}

	delimitters := []rune{';', '\t', '|'}
	scores := make(map[rune]int)

	for _, delim := range delimitters {
		// Count occurrences in first few lines (skip header and comment lines)
		lines := strings.Split(content, "\n")
		if len(lines) > 5 {
			lines = lines[:5]
		}

		counts := make([]int, len(lines))
		for i, line := range lines {
			// Skip empty lines and comment lines
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				counts[i] = -1 // Mark as invalid
				continue
			}
			counts[i] = strings.Count(line, string(delim))
		}

		// Check consistency (only count non-empty, non-comment lines)
		validCounts := make([]int, 0)
		for _, c := range counts {
			if c >= 0 {
				validCounts = append(validCounts, c)
			}
		}

		if len(validCounts) > 0 {
			consistent := true
			firstCount := validCounts[0]
			if firstCount > 0 {
				for _, c := range validCounts[1:] {
					if c != firstCount {
						consistent = false
						break
					}
				}
				if consistent {
					scores[delim] = firstCount
				}
			}
		}
	}

	// Return best delimiter
	bestDelim := ','
	bestScore := 0
	for delim, score := range scores {
		if score > bestScore {
			bestScore = score
			bestDelim = delim
		}
	}

	return bestDelim
}

// analyzeColumns analyzes column data types
func (s *CSVService) analyzeColumns(rows [][]string, headers []string) []models.ColumnDef {
	if len(rows) == 0 || len(headers) == 0 {
		return nil
	}

	columns := make([]models.ColumnDef, len(headers))

	// Sample up to 100 rows
	sampleSize := 100
	if len(rows) < sampleSize {
		sampleSize = len(rows)
	}

	for i := range headers {
		columns[i] = s.inferColumnType(rows, i, sampleSize, headers)
	}

	return columns
}

// sanitizeColumnName sanitizes a column name for SQL use
func (s *CSVService) sanitizeColumnName(name string, colIndex int) string {
	// Replace common special characters for SQL safety
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, ".", "_")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	// Remove parentheses and other special chars
	name = strings.ReplaceAll(name, "(", "")
	name = strings.ReplaceAll(name, ")", "")
	name = strings.ReplaceAll(name, "[", "")
	name = strings.ReplaceAll(name, "]", "")
	name = strings.ReplaceAll(name, "{", "")
	name = strings.ReplaceAll(name, "}", "")
	name = strings.ReplaceAll(name, "<", "")
	name = strings.ReplaceAll(name, ">", "")
	name = strings.ReplaceAll(name, "=", "")
	name = strings.ReplaceAll(name, "+", "")
	name = strings.ReplaceAll(name, "*", "")
	name = strings.ReplaceAll(name, "%", "")
	name = strings.ReplaceAll(name, "&", "")
	name = strings.ReplaceAll(name, "|", "")
	name = strings.ReplaceAll(name, "@", "")
	name = strings.ReplaceAll(name, "#", "")
	name = strings.ReplaceAll(name, "$", "dollar")
	name = strings.ReplaceAll(name, "¥", "yuan")
	name = strings.ReplaceAll(name, "€", "euro")
	name = strings.ReplaceAll(name, "£", "pound")

	// Remove any remaining non-alphanumeric characters except underscore
	reg := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	name = reg.ReplaceAllString(name, "")

	// Ensure name starts with a letter
	if len(name) == 0 || (name[0] >= '0' && name[0] <= '9') {
		name = "col_" + name
	}

	// Ensure unique column name by appending column index if needed
	// This handles cases where multiple Chinese names all sanitize to the same value
	return fmt.Sprintf("%s_%d", name, colIndex)
}

// inferColumnType infers the data type of a column
func (s *CSVService) inferColumnType(rows [][]string, colIndex, sampleSize int, headers []string) models.ColumnDef {
	// Get original column name from header
	original := ""
	if colIndex < len(headers) {
		original = headers[colIndex]
	}

	// Generate safe SQL column name
	name := "column_" + strconv.Itoa(colIndex)
	if colIndex < len(headers) {
		name = s.sanitizeColumnName(headers[colIndex], colIndex)
	}

	// Type counters
	intCount := 0
	realCount := 0
	dateCount := 0
	textCount := 0
	emptyCount := 0

	dateRegex := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}`)
	// Enhanced integer regex to handle currency symbols and commas
	intRegex := regexp.MustCompile(`^-?[¥$€£]?\d{1,3}(,\d{3})*(\.\d+)?$|^-?\d+$`)
	// Enhanced real regex to handle currency symbols and commas
	realRegex := regexp.MustCompile(`^-?[¥$€£]?\d{1,3}(,\d{3})*(\.\d+)?$`)

	for i := 0; i < sampleSize && i < len(rows); i++ {
		if colIndex >= len(rows[i]) {
			emptyCount++
			continue
		}

		val := strings.TrimSpace(rows[i][colIndex])
		if val == "" || val == "NULL" || val == "null" {
			emptyCount++
			continue
		}

		// Remove currency symbols and commas for numeric detection
		cleanVal := strings.ReplaceAll(val, "¥", "")
		cleanVal = strings.ReplaceAll(cleanVal, ",", "")
		cleanVal = strings.TrimSpace(cleanVal)

		// Check integer
		if intRegex.MatchString(cleanVal) {
			// Try to parse as integer
			if _, err := strconv.ParseInt(cleanVal, 10, 64); err == nil {
				intCount++
				continue
			}
		}

		// Check real/float
		if realRegex.MatchString(cleanVal) {
			if _, err := strconv.ParseFloat(cleanVal, 64); err == nil {
				realCount++
				continue
			}
		}

		// Check date/datetime
		if dateRegex.MatchString(val) {
			// Try to parse as date
			_, err := time.Parse("2006-01-02", val)
			if err == nil {
				dateCount++
				continue
			}
			_, err = time.Parse("2006/01/02", val)
			if err == nil {
				dateCount++
				continue
			}
		}

		textCount++
	}

	// Determine type based on majority (excluding empty values)
	total := intCount + realCount + dateCount + textCount
	if total == 0 {
		return models.ColumnDef{Name: name, Type: "TEXT", Original: original}
	}

	// If all or most are integers
	if intCount > total*7/10 {
		return models.ColumnDef{Name: name, Type: "INTEGER", Original: original}
	}

	// If integers or reals
	if intCount+realCount > total*7/10 {
		return models.ColumnDef{Name: name, Type: "REAL", Original: original}
	}

	// If dates
	if dateCount > total*7/10 {
		return models.ColumnDef{Name: name, Type: "TEXT", Original: original} // SQLite stores dates as TEXT
	}

	return models.ColumnDef{Name: name, Type: "TEXT", Original: original}
}

// ConvertValue converts a string value to the appropriate type
func (s *CSVService) ConvertValue(value string, columnType string) interface{} {
	value = strings.TrimSpace(value)

	if value == "" || value == "NULL" || value == "null" {
		return nil
	}

	switch columnType {
	case "INTEGER":
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			return i
		}
		return value
	case "REAL":
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f
		}
		return value
	default:
		return value
	}
}

// PrepareRowsForInsert converts string rows to typed values
func (s *CSVService) PrepareRowsForInsert(rows [][]string, columns []models.ColumnDef) [][]interface{} {
	result := make([][]interface{}, len(rows))

	for i, row := range rows {
		values := make([]interface{}, len(columns))
		for j, col := range columns {
			if j < len(row) {
				values[j] = s.ConvertValue(row[j], col.Type)
			} else {
				values[j] = nil
			}
		}
		result[i] = values
	}

	return result
}

// GenerateTableName generates a safe table name from filename
func (s *CSVService) GenerateTableName(filename string, timestamp int64) string {
	// Remove extension
	name := strings.TrimSuffix(filename, ".csv")
	name = strings.TrimSuffix(name, ".CSV")

	// Replace invalid characters
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, ".", "_")

	// Only keep alphanumeric and underscore
	reg := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	name = reg.ReplaceAllString(name, "")

	// Ensure starts with letter
	if len(name) == 0 || (name[0] >= '0' && name[0] <= '9') {
		name = "table_" + name
	}

	return fmt.Sprintf("csv_%s_%d", name, timestamp)
}

// SanitizeTableName sanitizes a table name for SQL use
func (s *CSVService) SanitizeTableName(name string) string {
	reg := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	name = reg.ReplaceAllString(name, "")
	if len(name) == 0 {
		name = "unknown_table"
	}
	return name
}
