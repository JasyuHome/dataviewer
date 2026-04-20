package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"dataviewer/backend/models"
	"dataviewer/backend/services"

	"github.com/gin-gonic/gin"
)

type UploadHandler struct {
	csvService  *services.CSVService
	dbService   *services.DatabaseService
	storagePath string
}

func NewUploadHandler(csvSvc *services.CSVService, dbSvc *services.DatabaseService, storagePath string) *UploadHandler {
	return &UploadHandler{
		csvService:  csvSvc,
		dbService:   dbSvc,
		storagePath: storagePath,
	}
}

// UploadFile handles CSV file upload
func (h *UploadHandler) UploadFile(c *gin.Context) {
	// Get file from form
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get file: " + err.Error()})
		return
	}
	defer file.Close()

	// Validate file extension
	ext := filepath.Ext(header.Filename)
	if ext != ".csv" && ext != ".CSV" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only CSV files are allowed"})
		return
	}

	// Create storage directory if not exists
	if err := os.MkdirAll(h.storagePath, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create storage directory"})
		return
	}

	// Save uploaded file
	timestamp := time.Now().Unix()
	tempPath := filepath.Join(h.storagePath, fmt.Sprintf("temp_%d_%s", timestamp, header.Filename))

	outFile, err := os.Create(tempPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file: " + err.Error()})
		return
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, file); err != nil {
		os.Remove(tempPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file content"})
		return
	}

	// Parse CSV
	result, err := h.csvService.ParseFile(tempPath)
	if err != nil {
		os.Remove(tempPath)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse CSV: " + err.Error()})
		return
	}

	// Debug logging
	log.Printf("Parsed CSV: headers=%v", result.Headers)
	log.Printf("Parsed CSV: rows count=%d", len(result.Rows))
	if len(result.Rows) > 0 {
		log.Printf("First row: %v", result.Rows[0])
	}
	log.Printf("Parsed CSV: columns=%v", result.Columns)

	// Generate table name
	tableName := h.csvService.GenerateTableName(header.Filename, timestamp)

	// Check if table already exists, add more uniqueness if needed
	if h.dbService.TableExists(tableName) {
		tableName = fmt.Sprintf("csv_%d", timestamp)
	}

	// Create database table
	columns := make([]models.ColumnDef, len(result.Columns))
	for i, col := range result.Columns {
		columns[i] = models.ColumnDef{Name: col.Name, Type: col.Type}
	}

	if err := h.dbService.CreateTable(tableName, columns); err != nil {
		os.Remove(tempPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create table: " + err.Error()})
		return
	}

	// Prepare data for insertion
	values := h.csvService.PrepareRowsForInsert(result.Rows, result.Columns)

	// Batch insert data
	colNames := make([]string, len(result.Columns))
	for i, col := range result.Columns {
		colNames[i] = col.Name
	}

	if err := h.dbService.BatchInsertData(tableName, colNames, values); err != nil {
		h.dbService.DropTable(tableName)
		os.Remove(tempPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert data: " + err.Error()})
		return
	}

	// Move file to final location
	finalPath := filepath.Join(h.storagePath, fmt.Sprintf("%s.csv", tableName))
	if err := os.Rename(tempPath, finalPath); err != nil {
		// Keep the data even if rename fails
		os.Remove(tempPath)
	}

	// Save metadata
	colDefsJSON, _ := json.Marshal(result.Columns)
	id, err := h.dbService.SaveFileMetadata(header.Filename, tableName, result.RowCount, result.ColCount, string(colDefsJSON))
	if err != nil {
		// Continue anyway
	}

	// Return response
	c.JSON(http.StatusOK, models.UploadResponse{
		ID:        id,
		Filename:  header.Filename,
		TableName: tableName,
		RowCount:  result.RowCount,
		Preview:   result.Preview,
		Rows:      result.Rows,
		Columns:   result.Columns,
	})
}

// GetFiles returns all uploaded files metadata
func (h *UploadHandler) GetFiles(c *gin.Context) {
	files, err := h.dbService.GetAllFileMetadata()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"files": files})
}

// DeleteFile deletes a file and its data table
func (h *UploadHandler) DeleteFile(c *gin.Context) {
	idStr := c.Param("id")
	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file ID"})
		return
	}

	// Get file metadata first
	metadata, err := h.dbService.GetFileMetadata(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	// Delete from database
	if err := h.dbService.DeleteFileMetadata(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file: " + err.Error()})
		return
	}

	// Delete physical file
	csvPath := filepath.Join(h.storagePath, fmt.Sprintf("%s.csv", metadata.TableName))
	os.Remove(csvPath)

	c.JSON(http.StatusOK, gin.H{"message": "File deleted successfully"})
}

// RenameFile renames a file/table
func (h *UploadHandler) RenameFile(c *gin.Context) {
	idStr := c.Param("id")
	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file ID"})
		return
	}

	var body struct {
		NewName string `json:"new_name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if body.NewName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "New name is required"})
		return
	}

	// Sanitize new table name
	newTableName := h.csvService.SanitizeTableName(body.NewName)

	// Check if table name already exists
	if h.dbService.TableExists(newTableName) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Table name already exists"})
		return
	}

	// Update metadata
	if err := h.dbService.RenameFile(id, newTableName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to rename file: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "File renamed successfully", "new_table_name": newTableName})
}
