package main

import (
	"denchokun-api/handlers"
	"denchokun-api/middleware"
	"denchokun-api/models"
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
	// デフォルト設定
	config = Config{
		Server: ServerConfig{
			Port: ":8080",
			Mode: "debug",
		},
		Database: DatabaseConfig{
			BasePath: "./data",
		},
	}

	// 環境変数から設定を取得
	if basePath := os.Getenv("DENCHOKUN_BASEPATH"); basePath != "" {
		config.Database.BasePath = basePath
		log.Printf("Using base path from environment variable: %s", basePath)
	} else {
		log.Printf("Using default base path: %s", config.Database.BasePath)
	}

	if port := os.Getenv("DENCHOKUN_PORT"); port != "" {
		// ポート番号だけの場合は : を付ける
		if port[0] != ':' {
			port = ":" + port
		}
		config.Server.Port = port
		log.Printf("Using port from environment variable: %s", port)
	} else {
		log.Printf("Using default port: %s", config.Server.Port)
	}

	if mode := os.Getenv("DENCHOKUN_MODE"); mode != "" {
		config.Server.Mode = mode
		log.Printf("Using mode from environment variable: %s", mode)
	} else {
		log.Printf("Using default mode: %s", config.Server.Mode)
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

	// プレビューハンドラーの初期化（preview-link API用）
	previewHandler, err := handlers.NewPreviewHandler(config.Database.BasePath)
	if err != nil {
		log.Printf("Warning: Failed to initialize preview handler: %v", err)
		// プレビュー機能は必須ではないので、エラーでも続行
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.LoggingMiddleware())
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.ErrorMiddleware())

	api := r.Group("/v1/api")
	{
		api.GET("/health", handlers.HealthCheck)

		api.GET("/periods", handlers.GetPeriods)
		api.GET("/periodinfo", handlers.GetPeriod)
		api.POST("/periods", handlers.CreatePeriod)
		api.PUT("/periods/dates", handlers.UpdatePeriodDates)
		api.PUT("/periods/name", handlers.UpdatePeriodName)
		api.POST("/periods/connect", handlers.ConnectPeriod)

		api.POST("/deals", handlers.CreateDeal)
		api.GET("/deals", handlers.GetDeals)
		api.POST("/all-deals", handlers.GetAllDeals)
		api.GET("/deals/:dealId", handlers.GetDeal)
		api.PUT("/deals/:dealId", handlers.UpdateDeal)
		api.PUT("/deals/:dealId/to-otherperiod", handlers.ChangeDealPeriod)
		api.DELETE("/deals/:dealId", handlers.DeleteDeal)
		api.GET("/deals/:dealId/download", handlers.DownloadDealFile)

		// プレビューAPI（別サーバーへのリンクを返す）
		if previewHandler != nil {
			// 取引のプレビューリンクを取得
			api.GET("/preview-link", previewHandler.GetDealPreviewLink)
		}

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