package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	ServerPort   string
	DatabasePath string
	StoragePath  string
	UploadLimit  int64 // in bytes
}

func Load() *Config {
	// Get executable directory
	execPath, err := os.Executable()
	if err != nil {
		execPath = "."
	}
	execDir := filepath.Dir(execPath)

	// Use relative paths for database and storage
	dbPath := filepath.Join(execDir, "storage", "dataviewer.db")
	storagePath := filepath.Join(execDir, "storage", "uploads")

	// For development, use current working directory
	if wd, err := os.Getwd(); err == nil {
		dbPath = filepath.Join(wd, "storage", "dataviewer.db")
		storagePath = filepath.Join(wd, "storage", "uploads")
	}

	return &Config{
		ServerPort:   getEnv("SERVER_PORT", "9999"),
		DatabasePath: getEnv("DATABASE_PATH", dbPath),
		StoragePath:  getEnv("STORAGE_PATH", storagePath),
		UploadLimit:  100 * 1024 * 1024, // 100MB default
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
