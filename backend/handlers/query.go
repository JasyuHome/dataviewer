package handlers

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"dataviewer/backend/models"
	"dataviewer/backend/services"

	"github.com/gin-gonic/gin"
)

type QueryHandler struct {
	dbService *services.DatabaseService
}

func NewQueryHandler(dbSvc *services.DatabaseService) *QueryHandler {
	return &QueryHandler{dbService: dbSvc}
}

// QueryData handles data query with conditions
func (h *QueryHandler) QueryData(c *gin.Context) {
	var params models.QueryParams
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Validate table name
	if params.TableName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Table name is required"})
		return
	}

	if !h.dbService.TableExists(params.TableName) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Table not found"})
		return
	}

	// Set default pagination
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 50
	}
	if params.PageSize > 1000 {
		params.PageSize = 1000
	}

	// Build WHERE clause if conditions exist
	whereClause := ""
	args := []interface{}{}

	if len(params.Conditions) > 0 {
		logic := "AND"
		if strings.ToUpper(params.Logic) == "OR" {
			logic = "OR"
		}

		conditions, condArgs := h.buildConditions(params.Conditions)
		if len(conditions) > 0 {
			whereClause = " WHERE " + strings.Join(conditions, " "+logic+" ")
			args = condArgs
		}
	}

	// Get total count with conditions
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM \"%s\"%s", params.TableName, whereClause)
	var total int64
	if err := h.dbService.GetDB().QueryRow(countSQL, args...).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count records: " + err.Error()})
		return
	}

	// Validate sort field
	sortField := h.validateColumnName(params.SortField)
	sortOrder := "ASC"
	if strings.ToUpper(params.SortOrder) == "DESC" {
		sortOrder = "DESC"
	}

	// Ensure sortOrder is safe
	if sortOrder != "ASC" && sortOrder != "DESC" {
		sortOrder = "ASC"
	}

	offset := (params.Page - 1) * params.PageSize

	// Build query SQL - sortOrder is safe as it's validated above
	querySQL := fmt.Sprintf("SELECT * FROM \"%s\"%s ORDER BY %s %s LIMIT ? OFFSET ?",
		params.TableName, whereClause, sortField, sortOrder)

	// Add limit and offset to args
	args = append(args, params.PageSize, offset)

	rows, err := h.dbService.GetDB().Query(querySQL, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Query failed: " + err.Error()})
		return
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

	totalPages := int(total) / params.PageSize
	if int(total)%params.PageSize != 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, models.QueryResult{
		Data:       data,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	})
}

// buildConditions builds WHERE clause conditions
func (h *QueryHandler) buildConditions(conditions []models.Condition) ([]string, []interface{}) {
	var clauses []string
	var args []interface{}

	for _, cond := range conditions {
		// Skip conditions with empty field or empty value
		if cond.Field == "" || cond.Value == nil || cond.Value == "" {
			continue
		}

		field := h.validateColumnName(cond.Field)
		if field == "id" && cond.Field != "id" {
			// Invalid field name
			continue
		}

		switch cond.Operator {
		case "eq":
			clauses = append(clauses, fmt.Sprintf(`"%s" = ?`, field))
			args = append(args, cond.Value)
		case "ne":
			clauses = append(clauses, fmt.Sprintf(`"%s" != ?`, field))
			args = append(args, cond.Value)
		case "gt":
			clauses = append(clauses, fmt.Sprintf(`"%s" > ?`, field))
			args = append(args, cond.Value)
		case "lt":
			clauses = append(clauses, fmt.Sprintf(`"%s" < ?`, field))
			args = append(args, cond.Value)
		case "gte":
			clauses = append(clauses, fmt.Sprintf(`"%s" >= ?`, field))
			args = append(args, cond.Value)
		case "lte":
			clauses = append(clauses, fmt.Sprintf(`"%s" <= ?`, field))
			args = append(args, cond.Value)
		case "like":
			clauses = append(clauses, fmt.Sprintf(`"%s" LIKE ?`, field))
			args = append(args, fmt.Sprintf("%%%v%%", cond.Value))
		case "in":
			// Handle array values
			if arr, ok := cond.Value.([]interface{}); ok {
				placeholders := make([]string, len(arr))
				for i, v := range arr {
					placeholders[i] = "?"
					args = append(args, v)
				}
				clauses = append(clauses, fmt.Sprintf(`"%s" IN (%s)`, field, strings.Join(placeholders, ", ")))
			}
		case "between":
			// Handle range values
			if arr, ok := cond.Value.([]interface{}); ok && len(arr) == 2 {
				clauses = append(clauses, fmt.Sprintf(`"%s" BETWEEN ? AND ?`, field))
				args = append(args, arr[0], arr[1])
			}
		}
	}

	return clauses, args
}

// validateColumnName validates column name to prevent SQL injection
func (h *QueryHandler) validateColumnName(name string) string {
	if name == "" {
		return "id"
	}
	// Allow letters, numbers, underscores, and CJK characters
	for _, c := range name {
		// Allow alphanumeric and underscore
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			continue
		}
		// Allow CJK characters (Chinese, Japanese, Korean)
		if c >= 0x4E00 && c <= 0x9FFF {
			continue
		}
		// Invalid character
		return "id"
	}
	return name
}

// GetTableData gets paginated data from a table
func (h *QueryHandler) GetTableData(c *gin.Context) {
	tableName := c.Param("tableName")

	if !h.dbService.TableExists(tableName) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Table not found"})
		return
	}

	page := 1
	pageSize := 50
	sortField := "id"
	sortOrder := "ASC"

	if p := c.Query("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}
	if ps := c.Query("page_size"); ps != "" {
		fmt.Sscanf(ps, "%d", &pageSize)
	}
	if sf := c.Query("sort_field"); sf != "" {
		sortField = sf
	}
	if so := c.Query("sort_order"); so != "" {
		sortOrder = so
	}

	result, err := h.dbService.QueryData(tableName, page, pageSize, sortField, sortOrder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetTableStructure gets the structure of a table
func (h *QueryHandler) GetTableStructure(c *gin.Context) {
	tableName := c.Param("tableName")

	if !h.dbService.TableExists(tableName) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Table not found"})
		return
	}

	// Try to get column definitions from file_metadata table first
	var columnDefsJSON string
	err := h.dbService.GetDB().QueryRow(
		"SELECT column_defs FROM file_metadata WHERE table_name = ?", tableName,
	).Scan(&columnDefsJSON)

	if err != nil || columnDefsJSON == "" {
		// Try to get from notion_tables
		err = h.dbService.GetDB().QueryRow(
			"SELECT column_defs FROM notion_tables WHERE table_name = ?", tableName,
		).Scan(&columnDefsJSON)
	}

	if err != nil || columnDefsJSON == "" {
		// Fallback to PRAGMA table_info if no metadata found
		columns, err := h.dbService.GetTableStructure(tableName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"table_name": tableName, "columns": columns})
		return
	}

	// Parse column definitions from JSON
	var columns []models.ColumnDef
	if err := json.Unmarshal([]byte(columnDefsJSON), &columns); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse column definitions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"table_name": tableName, "columns": columns})
}

// ExportToCSV exports query results to CSV file
func (h *QueryHandler) ExportToCSV(c *gin.Context) {
	tableName := c.Param("tableName")

	if !h.dbService.TableExists(tableName) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Table not found"})
		return
	}

	// Get all data
	querySQL := fmt.Sprintf("SELECT * FROM %s", tableName)
	rows, err := h.dbService.GetDB().Query(querySQL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	columns, _ := rows.Columns()

	// Set response headers
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s.csv", tableName))

	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	// Write header
	if err := writer.Write(columns); err != nil {
		return
	}

	// Write data
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}

		strValues := make([]string, len(columns))
		for i, v := range values {
			if v == nil {
				strValues[i] = ""
			} else {
				strValues[i] = fmt.Sprintf("%v", v)
			}
		}

		writer.Write(strValues)
	}
}
