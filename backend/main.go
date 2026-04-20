package main

import (
	"log"
	"os"
	"path/filepath"

	"dataviewer/backend/config"
	"dataviewer/backend/handlers"
	"dataviewer/backend/services"

	"github.com/gin-gonic/gin"
)

func main() {
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
