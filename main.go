package main

import (
	"denchokun-api/handlers"
	"denchokun-api/middleware"
	"denchokun-api/models"
	"encoding/json"
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

type Config struct {
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
}

type ServerConfig struct {
	Port string `json:"port"`
	Mode string `json:"mode"`
}

type DatabaseConfig struct {
	BasePath string `json:"basePath"`
}

var config Config

func loadConfig() error {
	configFile := "config.json"
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		config = Config{
			Server: ServerConfig{
				Port: ":8080",
				Mode: "debug",
			},
			Database: DatabaseConfig{
				BasePath: "./data",
			},
		}
		return nil
	}

	file, err := os.Open(configFile)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	err := loadConfig()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	if config.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	err = models.InitDB(config.Database.BasePath)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// Migrate DealPartners and System tables to System.db if needed
	log.Println("Checking for table migration to System.db...")
	if err := models.MigrateToSystemDB(); err != nil {
		log.Printf("Warning: Table migration failed: %v", err)
		// Don't fatal error, just log the warning
	} else {
		log.Println("Table migration completed successfully")
	}

	r := gin.Default()
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.ErrorMiddleware())

	api := r.Group("/api/v1")
	{
		api.GET("/health", handlers.HealthCheck)

		api.GET("/periods", handlers.GetPeriods)
		api.POST("/periods", handlers.CreatePeriod)
		api.PUT("/periods/:period/dates", handlers.UpdatePeriodDates)
		api.PUT("/periods/:period/name", handlers.UpdatePeriodName)
		api.POST("/periods/:period/connect", handlers.ConnectPeriod)

		api.POST("/deals", handlers.CreateDeal)
		api.GET("/deals", handlers.GetDeals)
		api.GET("/deals/:dealId", handlers.GetDeal)
		api.PUT("/deals/:dealId", handlers.UpdateDeal)
		api.DELETE("/deals/:dealId", handlers.DeleteDeal)

		api.POST("/files", handlers.UploadFile)
		api.GET("/files/:fileId", handlers.DownloadFile)

		api.GET("/deal-partners", handlers.GetDealPartners)
		api.POST("/deal-partners", handlers.CreateDealPartner)
		api.PUT("/deal-partners/:name", handlers.UpdateDealPartner)
		api.DELETE("/deal-partners/:name", handlers.DeleteDealPartner)

		api.GET("/system", handlers.GetSystemInfo)
		api.PUT("/system", handlers.UpdateSystemInfo)

		api.POST("/query", handlers.ExecuteQuery)
	}

	log.Printf("Starting server on %s\n", config.Server.Port)
	if err := r.Run(config.Server.Port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}