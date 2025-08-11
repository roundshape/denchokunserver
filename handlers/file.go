package handlers

import (
	"crypto/sha256"
	"denchokun-api/models"
	"denchokun-api/utils"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

func UploadFile(c *gin.Context) {
	period := c.PostForm("period")
	if period == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"message": "Period is required",
		})
		return
	}

	dealID := c.PostForm("dealId")
	if dealID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"message": "Deal ID is required",
		})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"message": "File is required",
		})
		return
	}
	defer file.Close()

	if header.Size > 100*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "file_too_large",
			"message": "File size exceeds 100MB limit",
		})
		return
	}

	if err := models.ConnectToPeriod(period); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "connection_error",
			"message": err.Error(),
		})
		return
	}

	deal, err := models.GetDealByID(dealID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "not_found",
			"message": "Deal not found",
		})
		return
	}

	fileData, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "file_read_error",
			"message": err.Error(),
		})
		return
	}

	hash := sha256.Sum256(fileData)
	hashStr := hex.EncodeToString(hash[:])

	ext := filepath.Ext(header.Filename)
	fileName := fmt.Sprintf("%s_%s_%s_%d%s",
		dealID,
		deal.DealDate,
		strings.ReplaceAll(deal.DealPartner, "/", "_"),
		deal.DealPrice,
		ext)

	periodPath := filepath.Join("./data", period)
	if err := os.MkdirAll(periodPath, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "directory_error",
			"message": err.Error(),
		})
		return
	}

	filePath := filepath.Join(periodPath, fileName)
	if err := utils.SaveFileAtomic(filePath, fileData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "file_save_error",
			"message": err.Error(),
		})
		return
	}

	deal.FilePath = fileName
	deal.Hash = hashStr
	if err := models.UpdateDeal(dealID, deal); err != nil {
		os.Remove(filePath)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "database_error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"message":  "File uploaded successfully",
		"filePath": fileName,
		"hash":     hashStr,
		"size":     header.Size,
	})
}

func DownloadFile(c *gin.Context) {
	fileID := c.Param("fileId")
	if fileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"message": "File ID is required",
		})
		return
	}

	period := c.Query("period")
	if period == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"message": "Period is required",
		})
		return
	}

	filePath := filepath.Join("./data", period, fileID)
	
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "not_found",
			"message": "File not found",
		})
		return
	}

	c.File(filePath)
}