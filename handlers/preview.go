package handlers

import (
	"denchokun-api/models"
	"denchokun-api/preview"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// PreviewHandler はプレビュー機能を処理するハンドラー
type PreviewHandler struct {
	generator    preview.Generator
	cache        *preview.Cache
	dataBasePath string
}

// NewPreviewHandler は新しいプレビューハンドラーを作成
func NewPreviewHandler(dataBasePath string) (*PreviewHandler, error) {
	// キャッシュディレクトリの作成
	cacheDir := filepath.Join(dataBasePath, ".cache", "previews")
	cache, err := preview.NewCache(cacheDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache: %w", err)
	}
	
	// Windows用のプレビュージェネレータを作成
	generator := preview.NewWindowsPreviewGenerator()
	
	return &PreviewHandler{
		generator:    generator,
		cache:        cache,
		dataBasePath: dataBasePath,
	}, nil
}

// GetDealPreview は取引に紐づくファイルのプレビューを取得
func (h *PreviewHandler) GetDealPreview(c *gin.Context) {
	period := c.Param("period")
	dealId := c.Param("dealId")
	
	// パラメータの検証
	if period == "" || dealId == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_parameters",
			"message": "Period and dealId are required",
		})
		return
	}
	
	// オプションパラメータの取得
	width := h.getIntParam(c, "width", 300)
	height := h.getIntParam(c, "height", 300)
	page := h.getIntParam(c, "page", 1)
	format := c.DefaultQuery("format", "jpeg")
	responseFormat := c.DefaultQuery("response", "binary") // binary or base64
	
	// サイズの制限
	if width > 1000 {
		width = 1000
	}
	if height > 1000 {
		height = 1000
	}
	
	// データベースから取引情報を取得してファイルパスを特定
	filePath, err := h.getFilePathFromDeal(period, dealId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "file_not_found",
			"message": "File not found for the specified deal",
		})
		return
	}
	
	// キャッシュをチェック
	if cachedData, exists := h.cache.Get(filePath, width, height, page); exists {
		if responseFormat == "base64" {
			h.sendBase64Response(c, cachedData, format)
		} else {
			h.sendImageResponse(c, cachedData, format)
		}
		return
	}
	
	// プレビューを生成
	fmt.Printf("DEBUG: Attempting to generate preview for: %s\n", filePath)
	if _, err := os.Stat(filePath); err != nil {
		fmt.Printf("DEBUG: File stat error: %v\n", err)
	} else {
		fmt.Printf("DEBUG: File exists at path\n")
	}
	imageData, contentType, err := h.generator.GeneratePreviewBytes(filePath, width, height, format)
	if err != nil {
		// エラーログ
		fmt.Printf("ERROR: Failed to generate preview for %s: %v\n", filePath, err)
		fmt.Printf("ERROR: Full error details: %+v\n", err)
		
		// デフォルトアイコンを返す
		fmt.Printf("DEBUG: Returning default icon for extension: %s\n", filepath.Ext(filePath))
		imageData, contentType = h.getDefaultIcon(filepath.Ext(filePath))
	} else {
		fmt.Printf("DEBUG: Preview generated successfully, size: %d bytes\n", len(imageData))
	}
	
	// キャッシュに保存
	if err := h.cache.Put(filePath, width, height, page, imageData); err != nil {
		// キャッシュエラーは無視（ログのみ）
		fmt.Printf("Failed to cache preview: %v\n", err)
	}
	
	// レスポンスを送信
	if responseFormat == "base64" {
		h.sendBase64Response(c, imageData, contentType)
	} else {
		h.sendImageResponse(c, imageData, contentType)
	}
}

// GetFilePreview はファイルIDを指定してプレビューを取得
func (h *PreviewHandler) GetFilePreview(c *gin.Context) {
	fileId := c.Param("fileId")
	period := c.Query("period")
	
	// パラメータの検証
	if fileId == "" || period == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_parameters",
			"message": "FileId and period are required",
		})
		return
	}
	
	// オプションパラメータの取得
	width := h.getIntParam(c, "width", 300)
	height := h.getIntParam(c, "height", 300)
	page := h.getIntParam(c, "page", 1)
	format := c.DefaultQuery("format", "jpeg")
	responseFormat := c.DefaultQuery("response", "binary") // binary or base64
	
	// サイズの制限
	if width > 1000 {
		width = 1000
	}
	if height > 1000 {
		height = 1000
	}
	
	// ファイルパスを構築
	filePath := filepath.Join(h.dataBasePath, period, fileId)
	
	// ファイルの存在確認
	if _, err := os.Stat(filePath); err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "file_not_found",
			"message": "File not found or not accessible",
		})
		return
	}
	
	// キャッシュをチェック
	if cachedData, exists := h.cache.Get(filePath, width, height, page); exists {
		if responseFormat == "base64" {
			h.sendBase64Response(c, cachedData, format)
		} else {
			h.sendImageResponse(c, cachedData, format)
		}
		return
	}
	
	// プレビューを生成
	fmt.Printf("DEBUG: Attempting to generate preview for: %s\n", filePath)
	if _, err := os.Stat(filePath); err != nil {
		fmt.Printf("DEBUG: File stat error: %v\n", err)
	} else {
		fmt.Printf("DEBUG: File exists at path\n")
	}
	imageData, contentType, err := h.generator.GeneratePreviewBytes(filePath, width, height, format)
	if err != nil {
		// エラーログ
		fmt.Printf("ERROR: Failed to generate preview for %s: %v\n", filePath, err)
		fmt.Printf("ERROR: Full error details: %+v\n", err)
		
		// デフォルトアイコンを返す
		fmt.Printf("DEBUG: Returning default icon for extension: %s\n", filepath.Ext(filePath))
		imageData, contentType = h.getDefaultIcon(filepath.Ext(filePath))
	} else {
		fmt.Printf("DEBUG: Preview generated successfully, size: %d bytes\n", len(imageData))
	}
	
	// キャッシュに保存
	if err := h.cache.Put(filePath, width, height, page, imageData); err != nil {
		// キャッシュエラーは無視（ログのみ）
		fmt.Printf("Failed to cache preview: %v\n", err)
	}
	
	// レスポンスを送信
	if responseFormat == "base64" {
		h.sendBase64Response(c, imageData, contentType)
	} else {
		h.sendImageResponse(c, imageData, contentType)
	}
}

// getFilePathFromDeal はデータベースから取引のファイルパスを取得
func (h *PreviewHandler) getFilePathFromDeal(period, dealId string) (string, error) {
	// データベース接続を取得
	fmt.Printf("DEBUG: Connecting to period database: %s\n", period)
	db, err := models.ConnectPeriodDB(period)
	if err != nil {
		fmt.Printf("DEBUG: Database connection failed: %v\n", err)
		return "", err
	}
	
	var filePath string
	query := "SELECT FilePath FROM Deals WHERE NO = ?"
	fmt.Printf("DEBUG: Executing query: %s with dealId: %s\n", query, dealId)
	err = db.QueryRow(query, dealId).Scan(&filePath)
	if err != nil {
		fmt.Printf("DEBUG: Query failed: %v\n", err)
		return "", err
	}
	
	fmt.Printf("DEBUG: Retrieved filePath from DB: %s\n", filePath)
	
	// 相対パスの場合は絶対パスに変換
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(h.dataBasePath, period, filePath)
	}
	
	fmt.Printf("DEBUG: Final filePath: %s\n", filePath)
	
	// ファイルの存在確認
	if _, err := os.Stat(filePath); err != nil {
		fmt.Printf("DEBUG: File not found at path: %s, error: %v\n", filePath, err)
		return "", fmt.Errorf("file not found: %s", filePath)
	}
	
	return filePath, nil
}

// getIntParam はクエリパラメータから整数値を取得
func (h *PreviewHandler) getIntParam(c *gin.Context, name string, defaultValue int) int {
	if value := c.Query(name); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// sendImageResponse は画像レスポンスを送信
func (h *PreviewHandler) sendImageResponse(c *gin.Context, data []byte, contentType string) {
	// Content-Typeが文字列の場合の処理
	if contentType == "" || !strings.HasPrefix(contentType, "image/") {
		contentType = "image/jpeg"
	}
	
	// キャッシュヘッダーを設定
	c.Header("Cache-Control", "public, max-age=86400") // 24時間
	c.Header("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
	
	// 画像データを送信
	c.Data(http.StatusOK, contentType, data)
}

// sendBase64Response はBase64エンコードされた画像レスポンスを送信
func (h *PreviewHandler) sendBase64Response(c *gin.Context, data []byte, contentType string) {
	// Content-Typeが文字列の場合の処理
	if contentType == "" || !strings.HasPrefix(contentType, "image/") {
		contentType = "image/jpeg"
	}
	
	// Base64エンコード
	encodedData := base64.StdEncoding.EncodeToString(data)
	
	// JSONレスポンスとして送信
	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"contentType": contentType,
		"base64Data":  encodedData,
	})
}

// getDefaultIcon はデフォルトアイコンを取得
func (h *PreviewHandler) getDefaultIcon(ext string) ([]byte, string) {
	// 拡張子に応じたデフォルトアイコンを返す
	// 実際の実装では、事前に用意したアイコンファイルを読み込む
	iconPath := filepath.Join(h.dataBasePath, "assets", "icons", "default.png")
	
	// 拡張子別のアイコンがある場合
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))
	specificIconPath := filepath.Join(h.dataBasePath, "assets", "icons", ext+".png")
	if _, err := os.Stat(specificIconPath); err == nil {
		iconPath = specificIconPath
	}
	
	// アイコンファイルを読み込み
	data, err := os.ReadFile(iconPath)
	if err != nil {
		// 読み込みエラーの場合は空の画像を返す
		return []byte{}, "image/png"
	}
	
	return data, "image/png"
}

// GetCacheStats はキャッシュの統計情報を取得
func (h *PreviewHandler) GetCacheStats(c *gin.Context) {
	count, totalSize := h.cache.GetStats()
	
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"count":     count,
		"totalSize": totalSize,
		"sizeText":  formatBytes(totalSize),
	})
}

// ClearCache はキャッシュをクリア
func (h *PreviewHandler) ClearCache(c *gin.Context) {
	if err := h.cache.Clear(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "cache_clear_failed",
			"message": "Failed to clear cache",
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Cache cleared successfully",
	})
}

// GetDealPreviewLink は取引のプレビューリンクを返す
func (h *PreviewHandler) GetDealPreviewLink(c *gin.Context) {
	period := c.Query("period")
	dealId := c.Query("dealId")
	
	// パラメータの検証
	if period == "" || dealId == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_parameters",
			"message": "period and dealId query parameters are required",
		})
		return
	}
	
	// データベースから取引情報を取得してファイル名を取得
	if err := models.ConnectToPeriod(period); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "connection_error",
			"message": fmt.Sprintf("Failed to connect to period: %v", err),
		})
		return
	}
	
	deal, err := models.GetDealByID(dealId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "not_found",
			"message": fmt.Sprintf("Deal not found: %v", err),
		})
		return
	}
	
	// ファイルパスが存在しない場合
	if deal.FilePath == "" {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "no_file",
			"message": "No file associated with this deal",
		})
		return
	}
	
	// 環境変数からプレビューホストを取得（デフォルト値あり）
	previewHost := os.Getenv("DENCHOKUN_PREVIEW_HOST")
	if previewHost == "" {
		previewHost = "http://localhost:8081"
	}
	
	// プレビューURLを構築（修正版）
	// http://localhost:8081/v1/api/preview?period={period}&filename={filename}
	previewURL := fmt.Sprintf("%s/v1/api/preview?period=%s&filename=%s", 
		previewHost, 
		url.QueryEscape(period), 
		url.QueryEscape(deal.FilePath))
	
	// 元のリクエストから width, height などの追加パラメータを取得して追加
	for key, values := range c.Request.URL.Query() {
		// period と dealId は除外（既に処理済み）
		if key != "period" && key != "dealId" {
			for _, value := range values {
				previewURL = fmt.Sprintf("%s&%s=%s", previewURL, key, url.QueryEscape(value))
			}
		}
	}
	
	// レスポンスを返す
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"url": previewURL,
	})
}

// formatBytes はバイト数を人間が読みやすい形式に変換
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}