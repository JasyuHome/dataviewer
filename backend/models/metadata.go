package models

import "time"

// FileMetadata represents uploaded file metadata
type FileMetadata struct {
	ID           int64     `json:"id"`
	Filename     string    `json:"filename"`
	TableName    string    `json:"table_name"`
	UploadTime   time.Time `json:"upload_time"`
	RowCount     int       `json:"row_count"`
	ColumnCount  int       `json:"column_count"`
	Status       string    `json:"status"`
	ColumnDefs   []ColumnDef `json:"column_defs,omitempty"`
}

// ColumnDef represents a column definition
type ColumnDef struct {
	Name     string `json:"name"`      // Safe SQL column name
	Type     string `json:"type"`
	Original string `json:"original"`  // Original column name from CSV
}

// DataTableInfo represents table information
type DataTableInfo struct {
	ID              int64     `json:"id"`
	TableName       string    `json:"table_name"`
	ColumnDefinitions string  `json:"column_definitions"`
	CreatedAt       time.Time `json:"created_at"`
}

// QueryParams represents query parameters for data search
type QueryParams struct {
	TableName string        `json:"table_name"`
	Page      int           `json:"page"`
	PageSize  int           `json:"page_size"`
	SortField string        `json:"sort_field"`
	SortOrder string        `json:"sort_order"` // "ASC" or "DESC"
	Conditions []Condition  `json:"conditions"`
	Logic     string        `json:"logic"` // "AND" or "OR"
}

// Condition represents a single query condition
type Condition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // eq, ne, gt, lt, gte, lte, like, in
	Value    interface{} `json:"value"`
}

// QueryResult represents query result with pagination
type QueryResult struct {
	Data      []map[string]interface{} `json:"data"`
	Total     int64                    `json:"total"`
	Page      int                      `json:"page"`
	PageSize  int                      `json:"page_size"`
	TotalPages int                     `json:"total_pages"`
}

// ChartParams represents chart generation parameters
type ChartParams struct {
	TableName string   `json:"table_name"`
	ChartType string   `json:"chart_type"` // line, bar, pie
	XField    string   `json:"x_field"`
	YField    string   `json:"y_field"`
	Series    string   `json:"series,omitempty"`
	Limit     int      `json:"limit"`
}

// UploadResponse represents file upload response
type UploadResponse struct {
	ID        int64       `json:"id"`
	Filename  string      `json:"filename"`
	TableName string      `json:"table_name"`
	RowCount  int         `json:"row_count"`
	Preview   [][]string  `json:"preview"`
	Rows      [][]string  `json:"rows,omitempty"`
	Columns   []ColumnDef `json:"columns"`
}
