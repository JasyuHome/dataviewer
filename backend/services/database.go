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

	-- Create indexes
	CREATE INDEX IF NOT EXISTS idx_file_metadata_status ON file_metadata(status);
	CREATE INDEX IF NOT EXISTS idx_file_metadata_upload_time ON file_metadata(upload_time);
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

	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
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
	err := s.db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&total)
	if err != nil {
		return nil, err
	}

	// Validate sort field
	sortField = s.validateColumnName(sortField)
	if sortOrder != "ASC" && sortOrder != "DESC" {
		sortOrder = "ASC"
	}

	offset := (page - 1) * pageSize
	sql := fmt.Sprintf("SELECT * FROM %s ORDER BY %s %s LIMIT ? OFFSET ?",
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
