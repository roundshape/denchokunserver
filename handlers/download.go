package handlers

import (
	"denchokun-api/models"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// DownloadDealFile handles file download for a specific deal
func DownloadDealFile(c *gin.Context) {
	dealId := c.Param("dealId")
	period := c.Query("period")

	log.Printf("DownloadDealFile: dealId=%s, period=%s", dealId, period)

	// Validate parameters
	if dealId == "" {
		log.Println("DownloadDealFile: Deal ID is required")
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "missing_deal_id",
			"message": "Deal ID is required",
		})
		return
	}

	if period == "" {
		log.Println("DownloadDealFile: Period is required")
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "missing_period",
			"message": "Period is required",
		})
		return
	}

	// Connect to the period database
	db, err := models.ConnectPeriodDB(period)
	if err != nil {
		log.Printf("DownloadDealFile: Failed to connect to period %s: %v", period, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "database_error",
			"message": fmt.Sprintf("Failed to connect to period: %v", err),
		})
		return
	}

	// Get deal from database
	var deal models.Deal
	query := `SELECT NO, DealType, DealDate, DealName, DealPartner, DealPrice,
	          DealRemark, RecUpdate, RegDate, RecStatus, FilePath, Hash,
	          nextNO, prevNO
	          FROM Deals WHERE NO = ?`

	err = db.QueryRow(query, dealId).Scan(
		&deal.NO, &deal.DealType, &deal.DealDate, &deal.DealName,
		&deal.DealPartner, &deal.DealPrice, &deal.DealRemark,
		&deal.RecUpdate, &deal.RegDate, &deal.RecStatus,
		&deal.FilePath, &deal.Hash, &deal.NextNO, &deal.PrevNO,
	)

	if err != nil {
		log.Printf("DownloadDealFile: Deal not found: %v", err)
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "deal_not_found",
			"message": "Deal not found",
		})
		return
	}

	// Check if file path exists
	if deal.FilePath == "" {
		log.Printf("DownloadDealFile: No file associated with deal %s", dealId)
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "no_file",
			"message": "No file associated with this deal",
		})
		return
	}

	// Build full file path
	basePath := models.GetBasePath()
	fullPath := filepath.Join(basePath, period, deal.FilePath)
	log.Printf("DownloadDealFile: Attempting to serve file: %s", fullPath)

	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		log.Printf("DownloadDealFile: File not found at path: %s", fullPath)
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "file_not_found",
			"message": "File not found on disk",
		})
		return
	}

	// Get file extension to determine content type
	ext := strings.ToLower(filepath.Ext(deal.FilePath))
	contentType := getContentType(ext)

	// Set appropriate headers
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(deal.FilePath)))

	// If it's a PDF or image that might be displayed inline, also allow inline display
	if contentType == "application/pdf" || strings.HasPrefix(contentType, "image/") {
		c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filepath.Base(deal.FilePath)))
	}

	// Serve the file
	c.File(fullPath)
	log.Printf("DownloadDealFile: Successfully served file for deal %s", dealId)
}

// getContentType returns the appropriate content type based on file extension
func getContentType(ext string) string {
	switch ext {
	case ".pdf":
		return "application/pdf"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".bmp":
		return "image/bmp"
	case ".webp":
		return "image/webp"
	case ".txt":
		return "text/plain; charset=utf-8"
	case ".csv":
		return "text/csv; charset=utf-8"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".xls":
		return "application/vnd.ms-excel"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".doc":
		return "application/msword"
	case ".zip":
		return "application/zip"
	default:
		return "application/octet-stream"
	}
}