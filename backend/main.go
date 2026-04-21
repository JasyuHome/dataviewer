package main

import (
	"log"
	"os"
	"path/filepath"

	"dataviewer/backend/config"
	"dataviewer/backend/handlers"
	"dataviewer/backend/services"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Load configuration
	cfg := config.Load()

	// Initialize services
	dbService, err := services.NewDatabaseService(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer dbService.Close()

	csvService := services.NewCSVService()

	// Ensure storage directory exists
	if err := os.MkdirAll(cfg.StoragePath, 0755); err != nil {
		log.Fatalf("Failed to create storage directory: %v", err)
	}

	// Initialize handlers
	uploadHandler := handlers.NewUploadHandler(csvService, dbService, cfg.StoragePath)
	queryHandler := handlers.NewQueryHandler(dbService)
	chartHandler := handlers.NewChartHandler(dbService)

	// Initialize Notion service if configured
	var notionHandler *handlers.NotionHandler
	if cfg.NotionAPIKey != "" {
		notionService := services.NewNotionService(&services.NotionConfig{
			APIKey:        cfg.NotionAPIKey,
			NotionVersion: cfg.NotionVersion,
		})
		notionHandler = handlers.NewNotionHandler(notionService, dbService)
		log.Println("Notion integration enabled")
	} else {
		log.Println("Notion integration not configured (set NOTION_API_KEY to enable)")
	}

	// Setup Gin router
	r := gin.Default()

	// Enable CORS
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// API routes
	api := r.Group("/api")
	{
		// File upload endpoints
		api.POST("/upload", uploadHandler.UploadFile)
		api.GET("/files", uploadHandler.GetFiles)
		api.DELETE("/files/:id", uploadHandler.DeleteFile)
		api.PUT("/files/:id/rename", uploadHandler.RenameFile)

		// Query endpoints
		api.POST("/query", queryHandler.QueryData)
		api.GET("/tables/:tableName/data", queryHandler.GetTableData)
		api.GET("/tables/:tableName/structure", queryHandler.GetTableStructure)
		api.GET("/tables/:tableName/export", queryHandler.ExportToCSV)

		// Chart endpoints
		api.POST("/charts/generate", chartHandler.GenerateChart)
		api.GET("/charts/:tableName/data", chartHandler.GetChartData)

		// Notion endpoints (if configured)
		if notionHandler != nil {
			notion := api.Group("/notion")
			{
				notion.GET("/databases", notionHandler.ListDatabases)
				notion.GET("/databases/:databaseID", notionHandler.GetDatabase)
				notion.POST("/databases/:databaseID/query", notionHandler.QueryDatabase)
				notion.POST("/pages", notionHandler.CreatePage)
				notion.PUT("/pages/:pageID", notionHandler.UpdatePage)
				notion.DELETE("/pages/:pageID", notionHandler.DeletePage)
				notion.GET("/search", notionHandler.Search)
				// Notion table cache endpoints
				notion.POST("/save", notionHandler.SaveNotionData)
				notion.POST("/sync", notionHandler.SyncNotionData)
				notion.GET("/tables", notionHandler.ListNotionTables)
				notion.DELETE("/tables/:tableName", notionHandler.DeleteNotionTable)
			}
		}
	}

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
			"version": "1.0.0",
		})
	})

	// Serve static files (for production)
	staticPath := filepath.Join(filepath.Dir(cfg.StoragePath), "frontend", "build")
	if _, err := os.Stat(staticPath); err == nil {
		r.Static("/static", staticPath)
		r.StaticFile("/", filepath.Join(staticPath, "index.html"))
	}

	// Start server
	log.Printf("Starting server on port %s", cfg.ServerPort)
	log.Printf("Database: %s", cfg.DatabasePath)
	log.Printf("Storage: %s", cfg.StoragePath)

	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
