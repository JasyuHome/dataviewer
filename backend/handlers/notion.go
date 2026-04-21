package handlers

import (
	"net/http"

	"dataviewer/backend/services"

	"github.com/gin-gonic/gin"
)

type NotionHandler struct {
	notionService *services.NotionService
	dbService     *services.DatabaseService
}

// NewNotionHandler creates a new Notion handler
func NewNotionHandler(notionService *services.NotionService, dbService *services.DatabaseService) *NotionHandler {
	return &NotionHandler{
		notionService: notionService,
		dbService:     dbService,
	}
}

// DatabaseInfo represents database information for list response
type DatabaseInfo struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

// ListDatabasesResponse represents the response for listing databases
type ListDatabasesResponse struct {
	Databases []DatabaseInfo `json:"databases"`
}

// QueryRequest represents a query request
type QueryRequest struct {
	DatabaseID string                 `json:"database_id"`
	Filter     map[string]interface{} `json:"filter,omitempty"`
	Sorts      []Sort                 `json:"sorts,omitempty"`
	PageSize   int                    `json:"page_size,omitempty"`
}

// Sort represents a sort configuration
type Sort struct {
	Property  string `json:"property"`
	Direction string `json:"direction"` // "ascending" or "descending"
}

// QueryResponse represents a query response
type QueryResponse struct {
	Results    []PageResult `json:"results"`
	HasMore    bool         `json:"has_more"`
	NextCursor string       `json:"next_cursor"`
}

// PageResult represents a page in the query result
type PageResult struct {
	ID         string                 `json:"id"`
	CreatedTime string                `json:"created_time"`
	Properties map[string]interface{} `json:"properties"`
	URL        string                 `json:"url"`
}

// CreatePageRequest represents a request to create a page
type CreatePageRequest struct {
	DatabaseID string                 `json:"database_id"`
	Properties map[string]interface{} `json:"properties"`
}

// CreatePageResponse represents the response for creating a page
type CreatePageResponse struct {
	ID         string                 `json:"id"`
	URL        string                 `json:"url"`
	Properties map[string]interface{} `json:"properties"`
}

// SearchRequest represents a search request
type SearchRequest struct {
	Query  string `json:"query"`
	Filter string `json:"filter,omitempty"` // "page" or "database"
}

// SaveNotionDataRequest represents a request to save Notion data
type SaveNotionDataRequest struct {
	DatabaseID   string `json:"database_id"`
	TableName    string `json:"table_name"`
}

// SyncNotionDataRequest represents a request to sync Notion data
type SyncNotionDataRequest struct {
	TableName string `json:"table_name"`
}

// NotionTableInfo represents Notion table info response
type NotionTableInfo struct {
	ID               int64                  `json:"id"`
	TableName        string                 `json:"table_name"`
	NotionDatabaseID string                 `json:"notion_database_id"`
	LastSyncTime     string                 `json:"last_sync_time"`
	RowCount         int64                  `json:"row_count"`
	CreatedAt        string                 `json:"created_at"`
}

// ListDatabases lists all accessible Notion databases
func (h *NotionHandler) ListDatabases(c *gin.Context) {
	databases, err := h.notionService.ListDatabases()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	result := ListDatabasesResponse{
		Databases: make([]DatabaseInfo, 0),
	}

	for _, db := range databases {
		title := ""
		if len(db.Title) > 0 {
			title = db.Title[0].PlainText
		}
		result.Databases = append(result.Databases, DatabaseInfo{
			ID:    db.ID,
			Title: title,
			URL:   db.URL,
		})
	}

	c.JSON(http.StatusOK, result)
}

// GetDatabase retrieves a specific database
func (h *NotionHandler) GetDatabase(c *gin.Context) {
	databaseID := c.Param("databaseID")
	if databaseID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "database_id is required",
		})
		return
	}

	db, err := h.notionService.GetDatabase(databaseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, db)
}

// QueryDatabase queries a Notion database
func (h *NotionHandler) QueryDatabase(c *gin.Context) {
	databaseID := c.Param("databaseID")
	if databaseID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "database_id is required",
		})
		return
	}

	var req QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Convert sorts to interface{} for service layer
	var sorts interface{}
	if len(req.Sorts) > 0 {
		sortsArr := make([]map[string]interface{}, len(req.Sorts))
		for i, sort := range req.Sorts {
			sortsArr[i] = map[string]interface{}{
				"property":  sort.Property,
				"direction": sort.Direction,
			}
		}
		sorts = sortsArr
	}

	pages, err := h.notionService.QueryDatabase(databaseID, req.Filter, sorts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	queryResult := QueryResponse{
		Results: make([]PageResult, 0),
	}

	for _, page := range pages {
		queryResult.Results = append(queryResult.Results, PageResult{
			ID:          page.ID,
			CreatedTime: page.CreatedTime,
			Properties:  page.Properties,
			URL:         page.URL,
		})
	}

	c.JSON(http.StatusOK, queryResult)
}

// CreatePage creates a new page in a Notion database
func (h *NotionHandler) CreatePage(c *gin.Context) {
	var req CreatePageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if req.DatabaseID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "database_id is required",
		})
		return
	}

	page, err := h.notionService.CreatePage(req.DatabaseID, req.Properties)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, CreatePageResponse{
		ID:         page.ID,
		URL:        page.URL,
		Properties: page.Properties,
	})
}

// UpdatePage updates an existing page
func (h *NotionHandler) UpdatePage(c *gin.Context) {
	pageID := c.Param("pageID")
	if pageID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "page_id is required",
		})
		return
	}

	var req CreatePageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	page, err := h.notionService.UpdatePage(pageID, req.Properties)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, CreatePageResponse{
		ID:         page.ID,
		URL:        page.URL,
		Properties: page.Properties,
	})
}

// DeletePage archives (soft deletes) a page
func (h *NotionHandler) DeletePage(c *gin.Context) {
	pageID := c.Param("pageID")
	if pageID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "page_id is required",
		})
		return
	}

	if err := h.notionService.ArchivePage(pageID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Page archived successfully",
	})
}

// Search searches for pages in Notion
func (h *NotionHandler) Search(c *gin.Context) {
	query := c.Query("q")
	filter := c.Query("filter") // "page" or "database"

	pages, err := h.notionService.Search(query, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	results := make([]PageResult, 0)
	for _, page := range pages {
		results = append(results, PageResult{
			ID:          page.ID,
			CreatedTime: page.CreatedTime,
			Properties:  page.Properties,
			URL:         page.URL,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
	})
}

// SaveNotionData saves Notion data to local database
func (h *NotionHandler) SaveNotionData(c *gin.Context) {
	var req SaveNotionDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if req.DatabaseID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "database_id is required",
		})
		return
	}

	if req.TableName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "table_name is required",
		})
		return
	}

	// Query data from Notion
	pages, err := h.notionService.QueryDatabase(req.DatabaseID, nil, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Save to local database
	if err := h.dbService.ImportNotionData(req.TableName, req.DatabaseID, pages); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Data saved successfully",
		"table_name": req.TableName,
		"row_count":  len(pages),
	})
}

// SyncNotionData syncs Notion data to local database
func (h *NotionHandler) SyncNotionData(c *gin.Context) {
	var req SyncNotionDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if req.TableName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "table_name is required",
		})
		return
	}

	// Get metadata to find the database ID
	meta, err := h.dbService.GetNotionTableMeta(req.TableName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Query data from Notion
	pages, err := h.notionService.QueryDatabase(meta.NotionDatabaseID, nil, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Save to local database
	if err := h.dbService.ImportNotionData(req.TableName, meta.NotionDatabaseID, pages); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Data synced successfully",
		"table_name": req.TableName,
		"row_count":  len(pages),
	})
}

// ListNotionTables lists all cached Notion tables
func (h *NotionHandler) ListNotionTables(c *gin.Context) {
	tables, err := h.dbService.GetAllNotionTables()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	result := make([]NotionTableInfo, 0)
	for _, table := range tables {
		result = append(result, NotionTableInfo{
			ID:               table.ID,
			TableName:        table.TableName,
			NotionDatabaseID: table.NotionDatabaseID,
			LastSyncTime:     table.LastSyncTime,
			RowCount:         table.RowCount,
			CreatedAt:        table.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"tables": result,
	})
}

// DeleteNotionTable deletes a cached Notion table
func (h *NotionHandler) DeleteNotionTable(c *gin.Context) {
	tableName := c.Param("tableName")
	if tableName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "table_name is required",
		})
		return
	}

	if err := h.dbService.DeleteNotionTable(tableName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Table deleted successfully",
	})
}
