package handlers

import (
	"crypto/sha256"
	"denchokun-api/models"
	"denchokun-api/utils"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type DealRequest struct {
	Period   string       `json:"period" binding:"required"`
	DealData models.Deal  `json:"dealData"`
	FileData *FileRequest `json:"fileData,omitempty"`
}

type FileRequest struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Size       int64  `json:"size"`
	Hash       string `json:"hash"`
	Base64Data string `json:"base64Data,omitempty"`
}

func CreateDeal(c *gin.Context) {
	var req DealRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	if err := models.ConnectToPeriod(req.Period); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "connection_error",
			"message": err.Error(),
		})
		return
	}

	if req.DealData.NO == "" {
		req.DealData.NO = generateDealNumber(c, "")
	} else {
		// Check if the deal number already exists
		existingDeal, err := models.GetDealByID(req.DealData.NO)
		if err == nil && existingDeal != nil {
			// If it exists, generate a sequence number
			req.DealData.NO = generateSequenceNumber(req.DealData.NO)
		}
	}

	if req.FileData != nil && req.FileData.Base64Data != "" {
		fileData, err := base64.StdEncoding.DecodeString(req.FileData.Base64Data)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "invalid_file_data",
				"message": "Failed to decode base64 file data",
			})
			return
		}

		hash := sha256.Sum256(fileData)
		req.DealData.Hash = hex.EncodeToString(hash[:])

		if req.FileData.Path == "" {
			ext := filepath.Ext(req.FileData.Name)
			fileName := fmt.Sprintf("%s_%s_%s_%d%s",
				req.DealData.NO,
				req.DealData.DealDate,
				strings.ReplaceAll(req.DealData.DealPartner, "/", "_"),
				req.DealData.DealPrice,
				ext)
			req.FileData.Path = fileName
		}

		filePath := filepath.Join("./data", req.Period, req.FileData.Path)
		if err := utils.SaveFileAtomic(filePath, fileData); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "file_save_error",
				"message": err.Error(),
			})
			return
		}

		req.DealData.FilePath = req.FileData.Path
	}

	if err := models.CreateDeal(&req.DealData); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusConflict, gin.H{
				"success": false,
				"error":   "resource_conflict",
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "database_error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"message":  "Deal created successfully",
		"dealId":   req.DealData.NO,
		"filePath": req.DealData.FilePath,
	})
}

func GetDeals(c *gin.Context) {
	var filter models.DealFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	if err := models.ConnectToPeriod(filter.Period); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "connection_error",
			"message": err.Error(),
		})
		return
	}

	deals, totalCount, err := models.GetDeals(&filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "database_error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"count":      len(deals),
		"totalCount": totalCount,
		"deals":      deals,
	})
}

func GetDeal(c *gin.Context) {
	dealID := c.Param("dealId")
	if dealID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"message": "Deal ID is required",
		})
		return
	}

	period := c.Query("period")
	if period != "" {
		if err := models.ConnectToPeriod(period); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "connection_error",
				"message": err.Error(),
			})
			return
		}
	}

	deal, err := models.GetDealByID(dealID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "not_found",
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "database_error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"deal":    deal,
	})
}

func UpdateDeal(c *gin.Context) {
	dealID := c.Param("dealId")
	if dealID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"message": "Deal ID is required",
		})
		return
	}

	var req DealRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	if err := models.ConnectToPeriod(req.Period); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "connection_error",
			"message": err.Error(),
		})
		return
	}

	if req.FileData != nil && req.FileData.Base64Data != "" {
		fileData, err := base64.StdEncoding.DecodeString(req.FileData.Base64Data)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "invalid_file_data",
				"message": "Failed to decode base64 file data",
			})
			return
		}

		hash := sha256.Sum256(fileData)
		req.DealData.Hash = hex.EncodeToString(hash[:])

		if req.FileData.Path == "" {
			ext := filepath.Ext(req.FileData.Name)
			fileName := fmt.Sprintf("%s_%s_%s_%d%s",
				dealID,
				req.DealData.DealDate,
				strings.ReplaceAll(req.DealData.DealPartner, "/", "_"),
				req.DealData.DealPrice,
				ext)
			req.FileData.Path = fileName
		}

		oldDeal, err := models.GetDealByID(dealID)
		if err == nil && oldDeal.FilePath != "" && oldDeal.FilePath != req.FileData.Path {
			oldFilePath := filepath.Join("./data", req.Period, oldDeal.FilePath)
			os.Remove(oldFilePath)
		}

		filePath := filepath.Join("./data", req.Period, req.FileData.Path)
		if err := utils.SaveFileAtomic(filePath, fileData); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "file_save_error",
				"message": err.Error(),
			})
			return
		}

		req.DealData.FilePath = req.FileData.Path
	}

	if err := models.UpdateDeal(dealID, &req.DealData); err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "not_found",
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "database_error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Deal updated successfully",
	})
}

func DeleteDeal(c *gin.Context) {
	dealID := c.Param("dealId")
	if dealID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"message": "Deal ID is required",
		})
		return
	}

	period := c.Query("period")
	if period != "" {
		if err := models.ConnectToPeriod(period); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "connection_error",
				"message": err.Error(),
			})
			return
		}
	}

	deal, err := models.GetDealByID(dealID)
	if err == nil && deal.FilePath != "" {
		currentPeriod := models.GetCurrentPeriod()
		if currentPeriod != "" {
			filePath := filepath.Join("./data", currentPeriod, deal.FilePath)
			os.Remove(filePath)
		}
	}

	if err := models.DeleteDeal(dealID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "not_found",
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "database_error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Deal deleted successfully",
	})
}

// generateDealNumber generates a new deal number with machine ID and optional sequence
// Format: YYYYMMDDHHmmssPCXXX or YYYYMMDDHHmmssPCXXX-NN
func generateDealNumber(c *gin.Context, existingNo string) string {
	if existingNo == "" {
		// New deal number with machine ID
		machineID := getMachineID(c)
		timestamp := time.Now().Format("20060102150405")
		return fmt.Sprintf("%s%s", timestamp, machineID)
	} else {
		// Generate sequence number for existing base number
		return generateSequenceNumber(existingNo)
	}
}

// getMachineID generates a machine ID from client IP
// Example: 192.168.1.105 -> PC105
func getMachineID(c *gin.Context) string {
	ip := c.ClientIP()
	parts := strings.Split(ip, ".")
	if len(parts) > 0 {
		lastPart := parts[len(parts)-1]
		if num, err := strconv.Atoi(lastPart); err == nil {
			return fmt.Sprintf("PC%03d", num)
		}
	}
	// Fallback to PC000 if IP parsing fails
	return "PC000"
}

// generateSequenceNumber generates the next sequence number
// Example: 20240115143025PC01 -> 20240115143025PC01-01
//          20240115143025PC01-01 -> 20240115143025PC01-02
func generateSequenceNumber(baseNo string) string {
	// Check if already has sequence
	if strings.Contains(baseNo, "-") {
		parts := strings.Split(baseNo, "-")
		if len(parts) == 2 {
			mainNo := parts[0]
			if subNo, err := strconv.Atoi(parts[1]); err == nil {
				return fmt.Sprintf("%s-%02d", mainNo, subNo+1)
			}
		}
	}
	// First sequence for base number
	return fmt.Sprintf("%s-01", baseNo)
}

// isValidDealNumber checks if a deal number is valid (supports old and new formats)
func isValidDealNumber(dealNo string) bool {
	// UUID format (Da1b2c3d4e5)
	if regexp.MustCompile(`^D[a-z0-9]{10}$`).MatchString(dealNo) {
		return true
	}
	// Old timestamp format (20240115143025)
	if regexp.MustCompile(`^\d{14}$`).MatchString(dealNo) {
		return true
	}
	// Old timestamp with sequence (20240115143025-01)
	if regexp.MustCompile(`^\d{14}-\d{2}$`).MatchString(dealNo) {
		return true
	}
	// New format with machine ID (20240115143025PC001)
	if regexp.MustCompile(`^\d{14}PC\d{3}$`).MatchString(dealNo) {
		return true
	}
	// New format with machine ID and sequence (20240115143025PC001-01)
	if regexp.MustCompile(`^\d{14}PC\d{3}-\d{2}$`).MatchString(dealNo) {
		return true
	}
	return false
}