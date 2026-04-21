package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"dataviewer/backend/models"
	_ "github.com/mattn/go-sqlite3"
)

type DatabaseService struct {
	db *sql.DB
}

// GetDB returns the underlying sql.DB connection
func (s *DatabaseService) GetDB() *sql.DB {
	return s.db
}

func NewDatabaseService(dbPath string) (*DatabaseService, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	svc := &DatabaseService{db: db}
	if err := svc.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return svc, nil
}

func (s *DatabaseService) initSchema() error {
	schema := `
	-- File metadata table
	CREATE TABLE IF NOT EXISTS file_metadata (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		filename TEXT NOT NULL,
		table_name TEXT UNIQUE NOT NULL,
		upload_time DATETIME DEFAULT CURRENT_TIMESTAMP,
		row_count INTEGER,
		column_count INTEGER,
		status TEXT DEFAULT 'active',
		column_defs TEXT
	);

	-- Data tables info
	CREATE TABLE IF NOT EXISTS data_tables (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		table_name TEXT UNIQUE NOT NULL,
		column_definitions TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Notion tables metadata
	CREATE TABLE IF NOT EXISTS notion_tables (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		table_name TEXT UNIQUE NOT NULL,
		notion_database_id TEXT NOT NULL,
		last_sync_time DATETIME DEFAULT CURRENT_TIMESTAMP,
		row_count INTEGER DEFAULT 0,
		column_defs TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Create indexes
	CREATE INDEX IF NOT EXISTS idx_file_metadata_status ON file_metadata(status);
	CREATE INDEX IF NOT EXISTS idx_file_metadata_upload_time ON file_metadata(upload_time);
	CREATE INDEX IF NOT EXISTS idx_notion_tables_table_name ON notion_tables(table_name);
	`

	_, err := s.db.Exec(schema)
	return err
}

func (s *DatabaseService) Close() error {
	return s.db.Close()
}

// SaveFileMetadata saves file metadata to database
func (s *DatabaseService) SaveFileMetadata(filename, tableName string, rowCount, colCount int, columnDefs string) (int64, error) {
	result, err := s.db.Exec(
		`INSERT INTO file_metadata (filename, table_name, row_count, column_count, column_defs, status)
		 VALUES (?, ?, ?, ?, ?, 'active')`,
		filename, tableName, rowCount, colCount, columnDefs,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetFileMetadata retrieves file metadata by ID
func (s *DatabaseService) GetFileMetadata(id int64) (*models.FileMetadata, error) {
	var m models.FileMetadata
	var columnDefsJSON string
	err := s.db.QueryRow(
		`SELECT id, filename, table_name, upload_time, row_count, column_count, status, column_defs
		 FROM file_metadata WHERE id = ?`,
		id,
	).Scan(&m.ID, &m.Filename, &m.TableName, &m.UploadTime, &m.RowCount, &m.ColumnCount, &m.Status, &columnDefsJSON)
	if err != nil {
		return nil, err
	}
	if columnDefsJSON != "" {
		if err := json.Unmarshal([]byte(columnDefsJSON), &m.ColumnDefs); err != nil {
			log.Printf("Failed to parse column_defs: %v", err)
		}
	}
	return &m, nil
}

// GetAllFileMetadata retrieves all file metadata
func (s *DatabaseService) GetAllFileMetadata() ([]*models.FileMetadata, error) {
	rows, err := s.db.Query(
		`SELECT id, filename, table_name, upload_time, row_count, column_count, status, column_defs
		 FROM file_metadata ORDER BY upload_time DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.FileMetadata
	for rows.Next() {
		var m models.FileMetadata
		var columnDefsJSON string
		err := rows.Scan(&m.ID, &m.Filename, &m.TableName, &m.UploadTime, &m.RowCount, &m.ColumnCount, &m.Status, &columnDefsJSON)
		if err != nil {
			return nil, err
		}
		if columnDefsJSON != "" {
			if err := json.Unmarshal([]byte(columnDefsJSON), &m.ColumnDefs); err != nil {
				log.Printf("Failed to parse column_defs: %v", err)
			}
		}
		result = append(result, &m)
	}
	return result, rows.Err()
}

// DeleteFileMetadata deletes file metadata by ID
func (s *DatabaseService) DeleteFileMetadata(id int64) error {
	// Get the table name first
	m, err := s.GetFileMetadata(id)
	if err != nil {
		return err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	// Drop the data table
	_, err = tx.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", m.TableName))
	if err != nil {
		tx.Rollback()
		return err
	}

	// Delete metadata
	_, err = tx.Exec("DELETE FROM file_metadata WHERE id = ?", id)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// UpdateFileMetadataStatus updates file status
func (s *DatabaseService) UpdateFileMetadataStatus(id int64, status string) error {
	_, err := s.db.Exec("UPDATE file_metadata SET status = ? WHERE id = ?", status, id)
	return err
}

// RenameFile updates filename and table_name
func (s *DatabaseService) RenameFile(id int64, newTableName string) error {
	_, err := s.db.Exec("UPDATE file_metadata SET table_name = ? WHERE id = ?", newTableName, id)
	return err
}

// CreateTable creates a new data table dynamically
func (s *DatabaseService) CreateTable(tableName string, columns []models.ColumnDef) error {
	colDefs := make([]string, 0)
	hasIDColumn := false

	for _, col := range columns {
		// Skip empty column names
		if col.Name == "" {
			continue
		}
		// Check if there's already an 'id' column
		if strings.ToLower(col.Name) == "id" {
			hasIDColumn = true
		}
		colDefs = append(colDefs, fmt.Sprintf(`"%s" %s`, col.Name, col.Type))
	}

	// Ensure we have at least one column
	if len(colDefs) == 0 {
		return fmt.Errorf("no valid columns to create table")
	}

	// Add internal id as primary key only if there's no 'id' column in CSV
	var sql string
	if hasIDColumn {
		// Use _id as internal primary key
		sql = fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (_id INTEGER PRIMARY KEY AUTOINCREMENT, %s)",
			tableName, strings.Join(colDefs, ", "))
	} else {
		// Use id as primary key
		sql = fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (id INTEGER PRIMARY KEY AUTOINCREMENT, %s)",
			tableName, strings.Join(colDefs, ", "))
	}

	log.Printf("Creating table with SQL: %s", sql)
	_, err := s.db.Exec(sql)
	return err
}

// InsertData inserts a row into a data table
func (s *DatabaseService) InsertData(tableName string, columns []string, values []interface{}) error {
	placeholders := make([]string, len(values))
	for i := range placeholders {
		placeholders[i] = "?"
	}

	quotedCols := make([]string, len(columns))
	for i, col := range columns {
		quotedCols[i] = fmt.Sprintf(`"%s"`, col)
	}

	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tableName, strings.Join(quotedCols, ", "), strings.Join(placeholders, ", "))

	_, err := s.db.Exec(sql, values...)
	return err
}

// BatchInsertData inserts multiple rows using transaction
func (s *DatabaseService) BatchInsertData(tableName string, columns []string, rows [][]interface{}) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	placeholders := make([]string, len(columns))
	for i := range placeholders {
		placeholders[i] = "?"
	}

	quotedCols := make([]string, len(columns))
	for i, col := range columns {
		quotedCols[i] = fmt.Sprintf(`"%s"`, col)
	}

	sql := fmt.Sprintf("INSERT INTO \"%s\" (%s) VALUES (%s)",
		tableName, strings.Join(quotedCols, ", "), strings.Join(placeholders, ", "))

	stmt, err := tx.Prepare(sql)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, row := range rows {
		_, err = stmt.Exec(row...)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

// QueryData queries data from a table with pagination
func (s *DatabaseService) QueryData(tableName string, page, pageSize int, sortField, sortOrder string) (*models.QueryResult, error) {
	// Get total count
	var total int64
	err := s.db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM \"%s\"", tableName)).Scan(&total)
	if err != nil {
		return nil, err
	}

	// Validate sort field
	sortField = s.validateColumnName(sortField)
	if sortOrder != "ASC" && sortOrder != "DESC" {
		sortOrder = "ASC"
	}

	offset := (page - 1) * pageSize
	sql := fmt.Sprintf("SELECT * FROM \"%s\" ORDER BY %s %s LIMIT ? OFFSET ?",
		tableName, sortField, sortOrder)

	rows, err := s.db.Query(sql, pageSize, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, _ := rows.Columns()
	data := make([]map[string]interface{}, 0)

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}

		rowMap := make(map[string]interface{})
		for i, col := range columns {
			rowMap[col] = values[i]
		}
		data = append(data, rowMap)
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPages++
	}

	return &models.QueryResult{
		Data:       data,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// validateColumnName prevents SQL injection
func (s *DatabaseService) validateColumnName(name string) string {
	if name == "" {
		return "id"
	}
	// Simple validation - only allow alphanumeric and underscore
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return "id"
		}
	}
	return name
}

// TableExists checks if a table exists
func (s *DatabaseService) TableExists(tableName string) bool {
	var count int
	err := s.db.QueryRow(
		"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?",
		tableName,
	).Scan(&count)
	return err == nil && count > 0
}

// GetTableStructure returns column definitions for a table
func (s *DatabaseService) GetTableStructure(tableName string) ([]models.ColumnDef, error) {
	rows, err := s.db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []models.ColumnDef
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull int
		var defaultVal, pk interface{}

		err := rows.Scan(&cid, &name, &typ, &notNull, &defaultVal, &pk)
		if err != nil {
			continue
		}
		columns = append(columns, models.ColumnDef{Name: name, Type: typ})
	}
	return columns, rows.Err()
}

// DropTable drops a table
func (s *DatabaseService) DropTable(tableName string) error {
	_, err := s.db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName))
	return err
}

// CountRows returns the number of rows in a table
func (s *DatabaseService) CountRows(tableName string) (int64, error) {
	var count int64
	err := s.db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&count)
	return count, err
}

// NotionTableMeta represents Notion table metadata
type NotionTableMeta struct {
	ID             int64                  `json:"id"`
	TableName      string                 `json:"table_name"`
	NotionDatabaseID string               `json:"notion_database_id"`
	LastSyncTime   string                 `json:"last_sync_time"`
	RowCount       int64                  `json:"row_count"`
	ColumnDefs     []models.ColumnDef     `json:"column_defs"`
	CreatedAt      string                 `json:"created_at"`
}

// SaveNotionTableMeta saves Notion table metadata
func (s *DatabaseService) SaveNotionTableMeta(tableName, notionDatabaseID string, columnDefs []models.ColumnDef) error {
	columnDefsJSON, err := json.Marshal(columnDefs)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(
		`INSERT INTO notion_tables (table_name, notion_database_id, column_defs, row_count, last_sync_time)
		 VALUES (?, ?, ?, 0, CURRENT_TIMESTAMP)
		 ON CONFLICT(table_name) DO UPDATE SET
		 notion_database_id = excluded.notion_database_id,
		 column_defs = excluded.column_defs,
		 last_sync_time = CURRENT_TIMESTAMP`,
		tableName, notionDatabaseID, string(columnDefsJSON),
	)
	return err
}

// GetNotionTableMeta retrieves Notion table metadata by table name
func (s *DatabaseService) GetNotionTableMeta(tableName string) (*NotionTableMeta, error) {
	var meta NotionTableMeta
	var columnDefsJSON string

	err := s.db.QueryRow(
		`SELECT id, table_name, notion_database_id, last_sync_time, row_count, column_defs, created_at
		 FROM notion_tables WHERE table_name = ?`,
		tableName,
	).Scan(&meta.ID, &meta.TableName, &meta.NotionDatabaseID, &meta.LastSyncTime, &meta.RowCount, &columnDefsJSON, &meta.CreatedAt)
	if err != nil {
		return nil, err
	}

	if columnDefsJSON != "" {
		if err := json.Unmarshal([]byte(columnDefsJSON), &meta.ColumnDefs); err != nil {
			log.Printf("Failed to parse notion_tables column_defs: %v", err)
		}
	}

	return &meta, nil
}

// GetAllNotionTables retrieves all Notion tables metadata
func (s *DatabaseService) GetAllNotionTables() ([]*NotionTableMeta, error) {
	rows, err := s.db.Query(
		`SELECT id, table_name, notion_database_id, last_sync_time, row_count, column_defs, created_at
		 FROM notion_tables ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*NotionTableMeta
	for rows.Next() {
		var meta NotionTableMeta
		var columnDefsJSON string
		err := rows.Scan(&meta.ID, &meta.TableName, &meta.NotionDatabaseID, &meta.LastSyncTime, &meta.RowCount, &columnDefsJSON, &meta.CreatedAt)
		if err != nil {
			return nil, err
		}
		if columnDefsJSON != "" {
			if err := json.Unmarshal([]byte(columnDefsJSON), &meta.ColumnDefs); err != nil {
				log.Printf("Failed to parse notion_tables column_defs: %v", err)
			}
		}
		result = append(result, &meta)
	}
	return result, rows.Err()
}

// DeleteNotionTable deletes Notion table metadata and drops the data table
func (s *DatabaseService) DeleteNotionTable(tableName string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	// Drop the data table
	_, err = tx.Exec(fmt.Sprintf("DROP TABLE IF EXISTS \"%s\"", tableName))
	if err != nil {
		tx.Rollback()
		return err
	}

	// Delete metadata
	_, err = tx.Exec("DELETE FROM notion_tables WHERE table_name = ?", tableName)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// UpdateNotionTableSyncTime updates the last sync time for a Notion table
func (s *DatabaseService) UpdateNotionTableSyncTime(tableName string) error {
	_, err := s.db.Exec("UPDATE notion_tables SET last_sync_time = CURRENT_TIMESTAMP WHERE table_name = ?", tableName)
	return err
}

// UpdateNotionTableRowCount updates the row count for a Notion table
func (s *DatabaseService) UpdateNotionTableRowCount(tableName string, rowCount int64) error {
	_, err := s.db.Exec("UPDATE notion_tables SET row_count = ? WHERE table_name = ?", rowCount, tableName)
	return err
}

// ImportNotionData imports Notion pages into a local table
func (s *DatabaseService) ImportNotionData(tableName string, notionDatabaseID string, pages []Page) error {
	if len(pages) == 0 {
		return nil
	}

	// Extract column definitions from first page
	columnDefs := s.extractNotionColumnDefs(pages[0])

	// Create table if not exists
	if err := s.CreateNotionTable(tableName, columnDefs); err != nil {
		return err
	}

	// Clear existing data
	if err := s.clearTable(tableName); err != nil {
		return err
	}

	// Check if we need to add _id or id column
	hasIDColumn := false
	for _, col := range columnDefs {
		if col.Name == "id" {
			hasIDColumn = true
			break
		}
	}

	// Prepare columns for batch insert (add internal id column)
	var columns []string
	if hasIDColumn {
		columns = append([]string{"_id"}, make([]string, len(columnDefs))...)
		for i, col := range columnDefs {
			columns[i+1] = col.Name
		}
	} else {
		columns = append([]string{"id"}, make([]string, len(columnDefs))...)
		for i, col := range columnDefs {
			columns[i+1] = col.Name
		}
	}

	// Convert pages to rows
	rows := make([][]interface{}, len(pages))
	for i, page := range pages {
		values, err := s.extractPageValues(page, columnDefs, hasIDColumn)
		if err != nil {
			log.Printf("Failed to extract values from page %s: %v", page.ID, err)
			continue
		}
		rows[i] = values
	}

	// Batch insert
	if err := s.BatchInsertData(tableName, columns, rows); err != nil {
		return err
	}

	// Update metadata
	if err := s.SaveNotionTableMeta(tableName, notionDatabaseID, columnDefs); err != nil {
		return err
	}

	if err := s.UpdateNotionTableRowCount(tableName, int64(len(rows))); err != nil {
		return err
	}

	return nil
}

// extractNotionColumnDefs extracts column definitions from a Notion page
func (s *DatabaseService) extractNotionColumnDefs(page Page) []models.ColumnDef {
	var columns []models.ColumnDef
	seenNames := make(map[string]bool)

	for propName, propValue := range page.Properties {
		if propValue == nil {
			continue
		}

		propMap, ok := propValue.(map[string]interface{})
		if !ok {
			continue
		}

		// Determine type based on Notion property structure
		colType := "TEXT"
		if propMap["title"] != nil {
			colType = "TEXT"
		} else if propMap["rich_text"] != nil {
			colType = "TEXT"
		} else if propMap["number"] != nil {
			colType = "REAL"
		} else if propMap["select"] != nil {
			colType = "TEXT"
		} else if propMap["multi_select"] != nil {
			colType = "TEXT"
		} else if propMap["checkbox"] != nil {
			colType = "INTEGER"
		} else if propMap["date"] != nil {
			colType = "TEXT"
		}

		// Sanitize column name
		safeName := s.sanitizeColumnName(propName)

		// Skip duplicate column names
		if seenNames[safeName] {
			continue
		}
		seenNames[safeName] = true

		columns = append(columns, models.ColumnDef{
			Name:     safeName,
			Type:     colType,
			Original: propName,
		})
	}

	return columns
}

// CreateNotionTable creates a table for Notion data
func (s *DatabaseService) CreateNotionTable(tableName string, columns []models.ColumnDef) error {
	colDefs := make([]string, 0)
	hasIDColumn := false

	for _, col := range columns {
		if col.Name == "" {
			continue
		}
		if col.Name == "id" {
			hasIDColumn = true
		}
		colDefs = append(colDefs, fmt.Sprintf(`"%s" %s`, col.Name, col.Type))
	}

	if len(colDefs) == 0 {
		return fmt.Errorf("no valid columns to create table")
	}

	// Add id as primary key if not already present
	var sql string
	if hasIDColumn {
		sql = fmt.Sprintf("CREATE TABLE IF NOT EXISTS \"%s\" (_id TEXT PRIMARY KEY, %s)",
			tableName, strings.Join(colDefs, ", "))
	} else {
		sql = fmt.Sprintf("CREATE TABLE IF NOT EXISTS \"%s\" (id TEXT PRIMARY KEY, %s)",
			tableName, strings.Join(colDefs, ", "))
	}

	_, err := s.db.Exec(sql)
	return err
}

// clearTable deletes all data from a table
func (s *DatabaseService) clearTable(tableName string) error {
	_, err := s.db.Exec(fmt.Sprintf("DELETE FROM \"%s\"", tableName))
	return err
}

// extractPageValues extracts values from a Notion page for insertion
func (s *DatabaseService) extractPageValues(page Page, columnDefs []models.ColumnDef, hasIDColumn bool) ([]interface{}, error) {
	// Create values array with space for internal id column
	values := make([]interface{}, len(columnDefs)+1)

	// First column is the internal id (_id or id)
	values[0] = page.ID

	// Fill in property values
	for i, col := range columnDefs {
		propValue, ok := page.Properties[col.Original]
		if !ok || propValue == nil {
			values[i+1] = nil
			continue
		}

		propMap, ok := propValue.(map[string]interface{})
		if !ok {
			values[i+1] = nil
			continue
		}

		// Extract value based on type
		switch col.Type {
		case "TEXT":
			values[i+1] = s.extractTextValue(propMap)
		case "REAL":
			values[i+1] = s.extractNumberValue(propMap)
		case "INTEGER":
			values[i+1] = s.extractCheckboxValue(propMap)
		default:
			values[i+1] = s.extractTextValue(propMap)
		}
	}

	return values, nil
}

// extractTextValue extracts text from various Notion property types
func (s *DatabaseService) extractTextValue(propMap map[string]interface{}) string {
	if title, ok := propMap["title"].([]interface{}); ok {
		return s.extractRichText(title)
	}
	if richText, ok := propMap["rich_text"].([]interface{}); ok {
		return s.extractRichText(richText)
	}
	if selectVal, ok := propMap["select"].(map[string]interface{}); ok {
		if name, ok := selectVal["name"].(string); ok {
			return name
		}
	}
	if multiSelect, ok := propMap["multi_select"].([]interface{}); ok {
		names := make([]string, 0)
		for _, item := range multiSelect {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if name, ok := itemMap["name"].(string); ok {
					names = append(names, name)
				}
			}
		}
		return strings.Join(names, ", ")
	}
	if date, ok := propMap["date"].(map[string]interface{}); ok {
		if start, ok := date["start"].(string); ok {
			return start
		}
	}
	if email, ok := propMap["email"].(string); ok {
		return email
	}
	if url, ok := propMap["url"].(string); ok {
		return url
	}
	if phone, ok := propMap["phone_number"].(string); ok {
		return phone
	}
	return ""
}

// extractRichText extracts plain text from rich text array
func (s *DatabaseService) extractRichText(richText []interface{}) string {
	var parts []string
	for _, item := range richText {
		if itemMap, ok := item.(map[string]interface{}); ok {
			if text, ok := itemMap["plain_text"].(string); ok {
				parts = append(parts, text)
			}
		}
	}
	return strings.Join(parts, " ")
}

// extractNumberValue extracts number from Notion property
func (s *DatabaseService) extractNumberValue(propMap map[string]interface{}) interface{} {
	if num, ok := propMap["number"].(float64); ok {
		return num
	}
	return nil
}

// extractCheckboxValue extracts boolean from Notion checkbox
func (s *DatabaseService) extractCheckboxValue(propMap map[string]interface{}) interface{} {
	if checked, ok := propMap["checkbox"].(bool); ok {
		if checked {
			return 1
		}
		return 0
	}
	return 0
}

// sanitizeColumnName converts a string to a valid SQL column name
func (s *DatabaseService) sanitizeColumnName(name string) string {
	// Allow letters, numbers, and Chinese/Japanese/Korean characters
	result := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		// Allow CJK characters
		if r >= '\u4e00' && r <= '\u9fff' {
			return r
		}
		return '_'
	}, name)

	// Ensure it doesn't start with a number
	if len(result) > 0 && result[0] >= '0' && result[0] <= '9' {
		result = "_" + result
	}

	// Limit length
	if len(result) > 63 {
		result = result[:63]
	}

	if result == "" {
		return "column"
	}

	return result
}
