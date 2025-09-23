package handlers

import (
	"crypto/sha256"
	"denchokun-api/models"
	"denchokun-api/utils"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
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
	log.Println("CreateDeal: Starting request processing")
	
	// Check content type to determine how to parse the request
	contentType := c.GetHeader("Content-Type")
	log.Printf("CreateDeal: Content-Type = %s", contentType)
	
	var req DealRequest
	var fileData []byte
	var fileName string
	var fileSize int64
	
	if strings.Contains(contentType, "multipart/form-data") {
		log.Println("CreateDeal: Processing multipart/form-data request")
		// Handle multipart/form-data request
		form, err := c.MultipartForm()
		if err != nil {
			log.Printf("CreateDeal: Failed to parse multipart form: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "invalid_request",
				"message": "Failed to parse multipart form",
			})
			return
		}
		
		// Parse JSON dealData from form
		dealDataStr := c.PostForm("dealData")
		log.Printf("CreateDeal: dealData = %s", dealDataStr)
		if dealDataStr == "" {
			log.Println("CreateDeal: dealData is empty")
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "invalid_request",
				"message": "dealData is required",
			})
			return
		}
		
		// Create a temporary structure for parsing multipart data
		type MultipartDealData struct {
			Period      string `json:"period"`
			DealType    string `json:"DealType"`
			DealDate    string `json:"DealDate"`
			DealName    string `json:"DealName"`
			DealPartner string `json:"DealPartner"`
			DealPrice   int    `json:"DealPrice"`
			DealRemark  string `json:"DealRemark"`
			RecStatus   string `json:"RecStatus"`
		}
		
		var multipartData MultipartDealData
		if err := json.Unmarshal([]byte(dealDataStr), &multipartData); err != nil {
			log.Printf("CreateDeal: Failed to unmarshal dealData JSON: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "invalid_request",
				"message": "Invalid dealData JSON",
			})
			return
		}
		log.Printf("CreateDeal: Parsed multipartData: %+v", multipartData)
		
		// Convert to DealRequest
		req.Period = multipartData.Period
		// If period is still empty, try to get it from query parameter
		if req.Period == "" {
			req.Period = c.Query("period")
			log.Printf("CreateDeal: Period from query parameter: %s", req.Period)
		}
		// Period is required
		if req.Period == "" {
			log.Println("CreateDeal: Period is required")
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "invalid_request",
				"message": "Period is required",
			})
			return
		}
		req.DealData = models.Deal{
			DealType:    multipartData.DealType,
			DealDate:    multipartData.DealDate,
			DealName:    multipartData.DealName,
			DealPartner: multipartData.DealPartner,
			DealPrice:   multipartData.DealPrice,
			DealRemark:  multipartData.DealRemark,
			RecStatus:   multipartData.RecStatus,
		}
		
		// Handle file upload if present
		files := form.File["file"]
		if len(files) > 0 {
			file := files[0]
			fileName = file.Filename
			fileSize = file.Size
			
			// Check file size (max 100MB)
			const maxFileSize = 100 * 1024 * 1024 // 100MB
			if fileSize > maxFileSize {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"error":   "file_too_large",
					"message": "ファイルサイズが100MBを超えています",
					"maxSize": maxFileSize,
				})
				return
			}
			
			// Read file content
			f, err := file.Open()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"error":   "file_read_error",
					"message": "Failed to open uploaded file",
				})
				return
			}
			defer f.Close()
			
			// Use io.ReadAll to read the entire file
			fileData, err = io.ReadAll(f)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"error":   "file_read_error",
					"message": "Failed to read uploaded file",
				})
				return
			}
		}
	} else {
		// Handle JSON request (existing code)
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "invalid_request",
				"message": err.Error(),
			})
			return
		}

		// Validate file data consistency
		if req.FileData != nil {
			// Check if size is declared but no data provided
			if req.FileData.Size > 0 && req.FileData.Base64Data == "" {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"error":   "invalid_file_data",
					"message": fmt.Sprintf("File size is %d bytes but no base64 data provided", req.FileData.Size),
				})
				return
			}
		}

		// If base64 file data is provided in JSON
		if req.FileData != nil && req.FileData.Base64Data != "" {
			var err error
			fileData, err = base64.StdEncoding.DecodeString(req.FileData.Base64Data)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"error":   "invalid_file_data",
					"message": "Failed to decode base64 file data",
				})
				return
			}
			fileName = req.FileData.Name
			fileSize = int64(len(fileData))
		}
	}

	log.Printf("CreateDeal: Connecting to period: %s", req.Period)
	if err := models.ConnectToPeriod(req.Period); err != nil {
		log.Printf("CreateDeal: Failed to connect to period %s: %v", req.Period, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "connection_error",
			"message": err.Error(),
		})
		return
	}

	// Always generate deal number on server side
	log.Println("CreateDeal: Generating deal number on server")
	req.DealData.NO = generateDealNumber(c, "")
	log.Printf("CreateDeal: Generated deal number: %s", req.DealData.NO)
	
	// Check if the generated deal number already exists (extremely rare)
	existingDeal, err := models.GetDealByID(req.DealData.NO)
	if err == nil && existingDeal != nil {
		// If it exists, generate a sequence number
		req.DealData.NO = generateSequenceNumber(req.DealData.NO)
		log.Printf("CreateDeal: Deal number already exists, generated sequence: %s", req.DealData.NO)
	}

	// Process file if present (from either multipart or JSON base64)
	if len(fileData) > 0 {
		log.Printf("CreateDeal: Processing file data, size: %d bytes", len(fileData))
		// Calculate hash
		hash := sha256.Sum256(fileData)
		req.DealData.Hash = hex.EncodeToString(hash[:])
		log.Printf("CreateDeal: File hash calculated: %s", req.DealData.Hash)

		// Always generate file path with server-generated deal number
		// Ignore client-provided path to ensure consistency
		var filePath string
		ext := filepath.Ext(fileName)
		if ext == "" && req.FileData != nil && req.FileData.Path != "" {
			// Extract extension from client path if fileName doesn't have one
			ext = filepath.Ext(req.FileData.Path)
		}
		generatedFileName := fmt.Sprintf("%s_%s_%s_%d%s",
			req.DealData.NO,
			req.DealData.DealDate,
			strings.ReplaceAll(req.DealData.DealPartner, "/", "_"),
			req.DealData.DealPrice,
			ext)
		filePath = generatedFileName
		log.Printf("CreateDeal: Generated file path: %s", filePath)

		// Ensure period directory exists
		periodDir := filepath.Join(models.GetBasePath(), req.Period)
		log.Printf("CreateDeal: Creating directory: %s", periodDir)
		if err := os.MkdirAll(periodDir, 0755); err != nil {
			log.Printf("CreateDeal: Failed to create directory: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "directory_create_error",
				"message": "Failed to create period directory",
			})
			return
		}

		// Save file
		fullPath := filepath.Join(periodDir, filePath)
		log.Printf("CreateDeal: Saving file to: %s", fullPath)
		if err := utils.SaveFileAtomic(fullPath, fileData); err != nil {
			log.Printf("CreateDeal: Failed to save file: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "file_save_error",
				"message": err.Error(),
			})
			return
		}

		req.DealData.FilePath = filePath
		log.Println("CreateDeal: File processing completed")

		// Check for duplicate hash across all periods (unless force flag is set)
		forceUpload := c.Query("force") == "true"
		log.Printf("CreateDeal: Checking for duplicate hash across all periods, force=%v", forceUpload)

		// Check all periods for duplicates
		allDuplicates, err := models.GetDealsByHashAllPeriods(req.DealData.Hash)
		if err != nil {
			log.Printf("CreateDeal: Failed to check duplicate hash: %v", err)
			// Continue without duplicate check on error
		} else if len(allDuplicates) > 0 && !forceUpload {
			log.Printf("CreateDeal: Duplicate file detected across periods, %d existing deals found", len(allDuplicates))

			// Prepare simplified duplicate info for response
			var duplicateInfo []gin.H
			for _, dup := range allDuplicates {
				duplicateInfo = append(duplicateInfo, gin.H{
					"NO":          dup.NO,
					"DealDate":    dup.DealDate,
					"DealPartner": dup.DealPartner,
					"DealPrice":   dup.DealPrice,
					"DealPeriod":  dup.Period, // Use the period where the duplicate was found
				})
			}

			c.JSON(http.StatusConflict, gin.H{
				"success":    false,
				"error":      "duplicate_file",
				"message":    "このファイルは既に登録されています。強制登録する場合は?force=trueを付けてください",
				"duplicates": duplicateInfo,
			})
			return
		} else if len(allDuplicates) > 0 && forceUpload {
			log.Printf("CreateDeal: Duplicate file detected but force flag is set, proceeding with registration")
		}

		// After duplicate check, connect back to the target period for actual registration
		log.Printf("CreateDeal: Reconnecting to target period: %s", req.Period)
		if err := models.ConnectToPeriod(req.Period); err != nil {
			log.Printf("CreateDeal: Failed to reconnect to period %s: %v", req.Period, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "connection_error",
				"message": err.Error(),
			})
			return
		}
	} else {
		log.Println("CreateDeal: No file data to process")
	}

	// Set timestamps
	log.Println("CreateDeal: Setting timestamps")
	now := time.Now().Format("2006-01-02T15:04:05Z")
	if req.DealData.RegDate == "" {
		req.DealData.RegDate = now
	}
	if req.DealData.RecUpdate == "" {
		req.DealData.RecUpdate = now
	}

	log.Printf("CreateDeal: Creating deal in database: %+v", req.DealData)
	if err := models.CreateDeal(&req.DealData); err != nil {
		log.Printf("CreateDeal: Database create failed: %v", err)
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

	// Build response
	response := gin.H{
		"success": true,
		"message": "Deal created successfully",
		"dealNo":  req.DealData.NO,
	}

	// Add file information if file was uploaded
	if req.DealData.FilePath != "" {
		response["filePath"] = req.DealData.FilePath
		response["fileSize"] = fileSize
		response["fileHash"] = req.DealData.Hash

		// Add warning if duplicate was found but force flag was used
		if forceUpload := c.Query("force") == "true"; forceUpload && len(fileData) > 0 {
			allDuplicates, _ := models.GetDealsByHashAllPeriods(req.DealData.Hash)
			if len(allDuplicates) > 0 {
				var duplicateWarnings []gin.H
				for _, dup := range allDuplicates {
					// Include all duplicates from all periods
					duplicateWarnings = append(duplicateWarnings, gin.H{
						"NO": dup.NO,
						"Period": dup.Period,
					})
				}
				if len(duplicateWarnings) > 0 {
					response["warning"] = "duplicate_file"
					response["duplicates"] = duplicateWarnings
				}
			}
		}
	}

	c.JSON(http.StatusOK, response)
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

	// Default to "flat" view if not specified
	if filter.View == "" {
		filter.View = "flat"
	}

	// Handle different view types
	if filter.View == "history" {
		// Get deals with history
		dealsWithHistory, count, err := models.GetDealsWithHistory(&filter)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "database_error",
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"view":    "history",
			"count":   count,
			"deals":   dealsWithHistory,
		})
	} else {
		// Get flat view (default)
		deals, count, err := models.GetDeals(&filter)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "database_error",
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"view":    "flat",
			"count":   count,
			"deals":   deals,
		})
	}
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

	log.Printf("UpdateDeal: Starting request processing for dealId: %s", dealID)
	
	// Check content type to determine how to parse the request
	contentType := c.GetHeader("Content-Type")
	log.Printf("UpdateDeal: Content-Type = %s", contentType)
	
	var req DealRequest
	var fileData []byte
	var fileName string
	var fileSize int64
	
	if strings.Contains(contentType, "multipart/form-data") {
		log.Println("UpdateDeal: Processing multipart/form-data request")
		// Handle multipart/form-data request
		form, err := c.MultipartForm()
		if err != nil {
			log.Printf("UpdateDeal: Failed to parse multipart form: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "invalid_request",
				"message": "Failed to parse multipart form",
			})
			return
		}
		
		// Parse JSON dealData from form
		dealDataStr := c.PostForm("dealData")
		log.Printf("UpdateDeal: dealData = %s", dealDataStr)
		if dealDataStr == "" {
			log.Println("UpdateDeal: dealData is empty")
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "invalid_request",
				"message": "dealData is required",
			})
			return
		}
		
		// Create a temporary structure for parsing multipart data
		type MultipartDealData struct {
			Period      string `json:"period"`
			DealType    string `json:"DealType"`
			DealDate    string `json:"DealDate"`
			DealName    string `json:"DealName"`
			DealPartner string `json:"DealPartner"`
			DealPrice   int    `json:"DealPrice"`
			DealRemark  string `json:"DealRemark"`
			RecStatus   string `json:"RecStatus"`
		}
		
		var multipartData MultipartDealData
		if err := json.Unmarshal([]byte(dealDataStr), &multipartData); err != nil {
			log.Printf("UpdateDeal: Failed to unmarshal dealData JSON: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "invalid_request",
				"message": "Invalid dealData JSON",
			})
			return
		}
		log.Printf("UpdateDeal: Parsed multipartData: %+v", multipartData)
		
		// Convert to DealRequest
		req.Period = multipartData.Period
		// If period is empty in multipart data, try to get it from query parameter
		if req.Period == "" {
			req.Period = c.Query("period")
			log.Printf("UpdateDeal: Period from query parameter: %s", req.Period)
		}
		if req.Period == "" {
			log.Println("UpdateDeal: Period is required")
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "invalid_request",
				"message": "Period is required",
			})
			return
		}
		req.DealData = models.Deal{
			DealType:    multipartData.DealType,
			DealDate:    multipartData.DealDate,
			DealName:    multipartData.DealName,
			DealPartner: multipartData.DealPartner,
			DealPrice:   multipartData.DealPrice,
			DealRemark:  multipartData.DealRemark,
		}
		
		// Handle file upload if present
		files := form.File["file"]
		if len(files) > 0 {
			file := files[0]
			fileName = file.Filename
			fileSize = file.Size
			
			// Check file size (max 100MB)
			const maxFileSize = 100 * 1024 * 1024 // 100MB
			if fileSize > maxFileSize {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"error":   "file_too_large",
					"message": "ファイルサイズが100MBを超えています",
					"maxSize": maxFileSize,
				})
				return
			}
			
			// Read file content
			f, err := file.Open()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"error":   "file_read_error",
					"message": "Failed to open uploaded file",
				})
				return
			}
			defer f.Close()
			
			// Use io.ReadAll to read the entire file
			fileData, err = io.ReadAll(f)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"error":   "file_read_error",
					"message": "Failed to read uploaded file",
				})
				return
			}
		}
	} else {
		// Handle JSON request (existing code)
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "invalid_request",
				"message": err.Error(),
			})
			return
		}

		// Validate file data consistency
		if req.FileData != nil {
			// Check if size is declared but no data provided
			if req.FileData.Size > 0 && req.FileData.Base64Data == "" {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"error":   "invalid_file_data",
					"message": fmt.Sprintf("File size is %d bytes but no base64 data provided", req.FileData.Size),
				})
				return
			}
		}

		// If period is empty in JSON data, try to get it from query parameter
		if req.Period == "" {
			req.Period = c.Query("period")
			log.Printf("UpdateDeal: Period from query parameter: %s", req.Period)
		}
		if req.Period == "" {
			log.Println("UpdateDeal: Period is required")
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "invalid_request",
				"message": "Period is required",
			})
			return
		}
		
		// If base64 file data is provided in JSON
		if req.FileData != nil && req.FileData.Base64Data != "" {
			var err error
			fileData, err = base64.StdEncoding.DecodeString(req.FileData.Base64Data)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"error":   "invalid_file_data",
					"message": "Failed to decode base64 file data",
				})
				return
			}
			fileName = req.FileData.Name
			fileSize = int64(len(fileData))
		}
	}

	log.Printf("UpdateDeal: Connecting to period: %s", req.Period)
	if err := models.ConnectToPeriod(req.Period); err != nil {
		log.Printf("UpdateDeal: Failed to connect to period %s: %v", req.Period, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "connection_error",
			"message": err.Error(),
		})
		return
	}

	// Get the original deal
	oldDeal, err := models.GetDealByID(dealID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "not_found",
				"message": "Deal not found: " + dealID,
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

	// Generate new deal number with branch suffix
	newDealNo := generateBranchNumber(dealID)
	log.Printf("UpdateDeal: Generated new deal number: %s", newDealNo)

	// Process file if present (from either multipart or JSON base64)
	if len(fileData) > 0 {
		log.Printf("UpdateDeal: Processing file data, size: %d bytes", len(fileData))
		// Calculate hash
		hash := sha256.Sum256(fileData)
		req.DealData.Hash = hex.EncodeToString(hash[:])
		log.Printf("UpdateDeal: File hash calculated: %s", req.DealData.Hash)

		// Always generate file path with server-generated deal number
		// Ignore client-provided path to ensure consistency
		var filePath string
		ext := filepath.Ext(fileName)
		if ext == "" && req.FileData != nil && req.FileData.Path != "" {
			// Extract extension from client path if fileName doesn't have one
			ext = filepath.Ext(req.FileData.Path)
		}
		generatedFileName := fmt.Sprintf("%s_%s_%s_%d%s",
			newDealNo,
			req.DealData.DealDate,
			strings.ReplaceAll(req.DealData.DealPartner, "/", "_"),
			req.DealData.DealPrice,
			ext)
		filePath = generatedFileName
		log.Printf("UpdateDeal: Generated file path: %s", filePath)

		// Ensure period directory exists
		periodDir := filepath.Join(models.GetBasePath(), req.Period)
		log.Printf("UpdateDeal: Creating directory: %s", periodDir)
		if err := os.MkdirAll(periodDir, 0755); err != nil {
			log.Printf("UpdateDeal: Failed to create directory: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "directory_create_error",
				"message": "Failed to create period directory",
			})
			return
		}

		// Save new file
		fullPath := filepath.Join(periodDir, filePath)
		log.Printf("UpdateDeal: Saving file to: %s", fullPath)
		if err := utils.SaveFileAtomic(fullPath, fileData); err != nil {
			log.Printf("UpdateDeal: Failed to save file: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "file_save_error",
				"message": err.Error(),
			})
			return
		}

		req.DealData.FilePath = filePath
		log.Println("UpdateDeal: File processing completed")

		// Check for duplicate hash across all periods (only checks RecStatus='NEW' records)
		forceUpload := c.Query("force") == "true"
		log.Printf("UpdateDeal: Checking for duplicate hash across all periods, force=%v", forceUpload)

		// Check all periods for duplicates
		allDuplicates, err := models.GetDealsByHashAllPeriods(req.DealData.Hash)
		if err != nil {
			log.Printf("UpdateDeal: Failed to check duplicate hash: %v", err)
			// Continue without duplicate check on error
		} else if len(allDuplicates) > 0 && !forceUpload {
			log.Printf("UpdateDeal: Duplicate file detected across periods, %d existing deals found", len(allDuplicates))

			// Prepare simplified duplicate info for response
			var duplicateInfo []gin.H
			for _, dup := range allDuplicates {
				duplicateInfo = append(duplicateInfo, gin.H{
					"NO":          dup.NO,
					"DealDate":    dup.DealDate,
					"DealPartner": dup.DealPartner,
					"DealPrice":   dup.DealPrice,
					"DealPeriod":  dup.Period, // Use the period where the duplicate was found
				})
			}

			c.JSON(http.StatusConflict, gin.H{
				"success":    false,
				"error":      "duplicate_file",
				"message":    "このファイルは既に登録されています。強制登録する場合は?force=trueを付けてください",
				"duplicates": duplicateInfo,
			})
			return
		} else if len(allDuplicates) > 0 && forceUpload {
			log.Printf("UpdateDeal: Duplicate file detected but force flag is set, proceeding with update")
		}

		// After duplicate check, connect back to the target period for actual update
		log.Printf("UpdateDeal: Reconnecting to target period: %s", req.Period)
		if err := models.ConnectToPeriod(req.Period); err != nil {
			log.Printf("UpdateDeal: Failed to reconnect to period %s: %v", req.Period, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "connection_error",
				"message": err.Error(),
			})
			return
		}
	} else {
		// No new file, but update the file path for the new deal number if exists
		if oldDeal.FilePath != "" {
			// Copy old file with new name
			oldFilePath := filepath.Join(models.GetBasePath(), req.Period, oldDeal.FilePath)
			ext := filepath.Ext(oldDeal.FilePath)
			newFileName := fmt.Sprintf("%s_%s_%s_%d%s",
				newDealNo,
				req.DealData.DealDate,
				strings.ReplaceAll(req.DealData.DealPartner, "/", "_"),
				req.DealData.DealPrice,
				ext)
			newFilePath := filepath.Join(models.GetBasePath(), req.Period, newFileName)
			
			// Read old file and save as new file
			if fileContent, err := os.ReadFile(oldFilePath); err == nil {
				if err := utils.SaveFileAtomic(newFilePath, fileContent); err == nil {
					req.DealData.FilePath = newFileName
					req.DealData.Hash = oldDeal.Hash
				}
			}
		}
		log.Println("UpdateDeal: No new file data to process")
	}

	// Create new record with updated data
	now := time.Now().Format("2006-01-02T15:04:05Z")
	req.DealData.NO = newDealNo
	prevNO := dealID
	req.DealData.PrevNO = &prevNO
	req.DealData.NextNO = nil
	req.DealData.RecStatus = "NEW"
	req.DealData.RecUpdate = now
	req.DealData.RegDate = now

	// Use single transaction to update old record and create new record
	log.Printf("UpdateDeal: Creating new deal record with history: old=%s, new=%s", dealID, newDealNo)
	if err := models.CreateDealWithHistory(dealID, &req.DealData); err != nil {
		log.Printf("UpdateDeal: Failed to create deal with history: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "database_error",
			"message": "Failed to update deal with history",
		})
		return
	}

	// Build response
	response := gin.H{
		"success":    true,
		"message":    "Deal updated successfully with history",
		"dealNo":     newDealNo,
		"previousNo": dealID,
	}
	
	// Add file information if file was uploaded
	if req.DealData.FilePath != "" {
		response["filePath"] = req.DealData.FilePath
		response["fileSize"] = fileSize
		response["fileHash"] = req.DealData.Hash

		// Add warning if duplicate was found but force flag was used
		if forceUpload := c.Query("force") == "true"; forceUpload && len(fileData) > 0 {
			allDuplicates, _ := models.GetDealsByHashAllPeriods(req.DealData.Hash)
			if len(allDuplicates) > 0 {
				var duplicateWarnings []gin.H
				for _, dup := range allDuplicates {
					// Include all duplicates from all periods
					duplicateWarnings = append(duplicateWarnings, gin.H{
						"NO": dup.NO,
						"Period": dup.Period,
					})
				}
				if len(duplicateWarnings) > 0 {
					response["warning"] = "duplicate_file"
					response["duplicates"] = duplicateWarnings
				}
			}
		}
	}

	log.Printf("UpdateDeal: Successfully created update record %s for original %s", newDealNo, dealID)
	c.JSON(http.StatusOK, response)
}

// generateBranchNumber generates a branch number for update history
// Example: D240115001 -> D240115001-1
//          D240115001-1 -> D240115001-2
func generateBranchNumber(baseNo string) string {
	// Check if already has branch suffix
	if idx := strings.LastIndex(baseNo, "-"); idx != -1 {
		mainNo := baseNo[:idx]
		if branchNo, err := strconv.Atoi(baseNo[idx+1:]); err == nil {
			return fmt.Sprintf("%s-%d", mainNo, branchNo+1)
		}
	}
	// First branch for base number
	return fmt.Sprintf("%s-1", baseNo)
}

func GetAllDeals(c *gin.Context) {
	// Create a custom filter struct without the required Period field
	type AllDealsFilter struct {
		FromDate  string   `json:"from_date"`
		ToDate    string   `json:"to_date"`
		Partner   string   `json:"partner"`
		Type      string   `json:"type"`
		Keyword   string   `json:"keyword"`
		Limit     int      `json:"limit"`
		Offset    int      `json:"offset"`
		Periods   []string `json:"periods"`  // New: array of period names to search
		View      string   `json:"view"`      // "flat" or "history"
	}
	
	var queryFilter AllDealsFilter
	if err := c.ShouldBindJSON(&queryFilter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}
	
	// Set default view if not specified
	if queryFilter.View == "" {
		queryFilter.View = "flat"
	}
	
	// Validate view parameter
	if queryFilter.View != "flat" && queryFilter.View != "history" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"message": "Invalid view parameter. Must be 'flat' or 'history'",
		})
		return
	}
	
	// Convert to models.DealFilter for use with existing GetDeals function
	filter := models.DealFilter{
		FromDate: queryFilter.FromDate,
		ToDate:   queryFilter.ToDate,
		Partner:  queryFilter.Partner,
		Type:     queryFilter.Type,
		Keyword:  queryFilter.Keyword,
		Limit:    queryFilter.Limit,
		Offset:   queryFilter.Offset,
		View:     queryFilter.View,
	}

	var periodsToSearch []string
	
	// If periods are specified, use them. Otherwise, get all available periods
	if len(queryFilter.Periods) > 0 {
		// Use specified periods
		periodsToSearch = queryFilter.Periods
		log.Printf("Searching specified periods: %v", periodsToSearch)
	} else {
		// Get all available periods (directory names with Denchokun.db)
		availablePeriods, err := models.GetAvailablePeriods()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "database_error",
				"message": "Failed to get available periods: " + err.Error(),
			})
			return
		}
		periodsToSearch = availablePeriods
		log.Printf("Searching all available periods: %v", periodsToSearch)
	}

	if len(periodsToSearch) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"count":   0,
			"deals":   []models.Deal{},
			"periods": []string{},
		})
		return
	}

	// Handle different view types
	if filter.View == "history" {
		// Collect all deals with history from specified/all periods
		var allDealsWithHistory []models.DealWithHistory
		periodsSearched := []string{}
		totalCount := 0

		for _, periodName := range periodsToSearch {
			// Connect to the period database
			if err := models.ConnectToPeriod(periodName); err != nil {
				// Log error but continue with other periods
				log.Printf("Failed to connect to period %s: %v", periodName, err)
				continue
			}

			// Set the period in filter temporarily
			originalPeriod := filter.Period
			filter.Period = periodName

			// Get deals with history from this period
			dealsWithHistory, count, err := models.GetDealsWithHistory(&filter)
			if err != nil {
				// Log error but continue with other periods
				log.Printf("Failed to get deals from period %s: %v", periodName, err)
				continue
			}

			// Add period information to each deal and its children
			for i := range dealsWithHistory {
				// Add to main deal
				if dealsWithHistory[i].DealRemark != "" {
					dealsWithHistory[i].DealRemark = fmt.Sprintf("[%s] %s", periodName, dealsWithHistory[i].DealRemark)
				} else {
					dealsWithHistory[i].DealRemark = fmt.Sprintf("[%s]", periodName)
				}
				
				// Add to children if exists
				for j := range dealsWithHistory[i].Children {
					if dealsWithHistory[i].Children[j].DealRemark != "" {
						dealsWithHistory[i].Children[j].DealRemark = fmt.Sprintf("[%s] %s", periodName, dealsWithHistory[i].Children[j].DealRemark)
					} else {
						dealsWithHistory[i].Children[j].DealRemark = fmt.Sprintf("[%s]", periodName)
					}
				}
			}

			allDealsWithHistory = append(allDealsWithHistory, dealsWithHistory...)
			totalCount += count
			periodsSearched = append(periodsSearched, periodName)

			// Restore original period value
			filter.Period = originalPeriod
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"view":    "history",
			"count":   totalCount,
			"deals":   allDealsWithHistory,
			"periods": periodsSearched,
		})
	} else {
		// Collect all deals (flat view) from specified/all periods
		var allDeals []models.Deal
		periodsSearched := []string{}
		totalCount := 0

		for _, periodName := range periodsToSearch {
			// Connect to the period database
			if err := models.ConnectToPeriod(periodName); err != nil {
				// Log error but continue with other periods
				log.Printf("Failed to connect to period %s: %v", periodName, err)
				continue
			}

			// Set the period in filter temporarily
			originalPeriod := filter.Period
			filter.Period = periodName

			// Get deals from this period using API request's fromDate and toDate
			deals, count, err := models.GetDeals(&filter)
			if err != nil {
				// Log error but continue with other periods
				log.Printf("Failed to get deals from period %s: %v", periodName, err)
				continue
			}

			// Add period information to each deal
			for i := range deals {
				// Store the period information in a field or use existing field
				if deals[i].DealRemark != "" {
					deals[i].DealRemark = fmt.Sprintf("[%s] %s", periodName, deals[i].DealRemark)
				} else {
					deals[i].DealRemark = fmt.Sprintf("[%s]", periodName)
				}
			}

			allDeals = append(allDeals, deals...)
			totalCount += count
			periodsSearched = append(periodsSearched, periodName)

			// Restore original period value
			filter.Period = originalPeriod
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"view":    "flat",
			"count":   totalCount,
			"deals":   allDeals,
			"periods": periodsSearched,
		})
	}
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

	// Logical delete: Keep the file, don't physically delete it
	// deal, err := models.GetDealByID(dealID)
	// if err == nil && deal.FilePath != "" {
	// 	currentPeriod := models.GetCurrentPeriod()
	// 	if currentPeriod != "" {
	// 		filePath := filepath.Join("./data", currentPeriod, deal.FilePath)
	// 		os.Remove(filePath)
	// 	}
	// }

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

// ChangeDealPeriod moves a deal from one period to another
// It marks the original deal as DELETE and creates a new deal in the target period
func ChangeDealPeriod(c *gin.Context) {
	dealID := c.Param("dealId")
	if dealID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"message": "Deal ID is required",
		})
		return
	}

	// Parse request body
	var req struct {
		FromPeriod string `json:"fromPeriod" binding:"required"`
		ToPeriod   string `json:"toPeriod" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}
	
	log.Printf("ChangeDealPeriod: Moving deal %s from period %s to %s", dealID, req.FromPeriod, req.ToPeriod)
	
	// Validate that periods are different
	if req.FromPeriod == req.ToPeriod {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"message": "Source and target periods must be different",
		})
		return
	}
	
	// Step 1: Connect to source period to get the original deal
	if err := models.ConnectToPeriod(req.FromPeriod); err != nil {
		log.Printf("ChangeDealPeriod: Failed to connect to source period %s: %v", req.FromPeriod, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "connection_error",
			"message": fmt.Sprintf("Failed to connect to source period: %v", err),
		})
		return
	}
	
	// Get the original deal (but don't modify it yet)
	originalDeal, err := models.GetDealByID(dealID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "not_found",
				"message": fmt.Sprintf("Deal not found in period %s: %s", req.FromPeriod, dealID),
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
	
	// Step 2: Connect to target period and create new deal first
	if err := models.ConnectToPeriod(req.ToPeriod); err != nil {
		log.Printf("ChangeDealPeriod: Failed to connect to target period %s: %v", req.ToPeriod, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "connection_error",
			"message": fmt.Sprintf("Failed to connect to target period: %v", err),
		})
		return
	}
	
	// Create new deal in target period
	newDeal := models.Deal{
		NO:          generateDealNumber(c, ""),  // Generate new deal number
		DealType:    originalDeal.DealType,
		DealDate:    originalDeal.DealDate,
		DealName:    originalDeal.DealName,
		DealPartner: originalDeal.DealPartner,
		DealPrice:   originalDeal.DealPrice,
		DealRemark:  originalDeal.DealRemark,
		RecStatus:   "NEW",
		RegDate:     time.Now().Format("2006-01-02T15:04:05Z"),
		RecUpdate:   time.Now().Format("2006-01-02T15:04:05Z"),
		Hash:        originalDeal.Hash,
	}
	
	// Handle file if exists
	var fileData []byte
	var originalFileInfo os.FileInfo
	if originalDeal.FilePath != "" {
		// Read the original file from source period
		originalFilePath := filepath.Join(models.GetBasePath(), req.FromPeriod, originalDeal.FilePath)
		
		// Get file info to preserve timestamps
		originalFileInfo, err = os.Stat(originalFilePath)
		if err != nil {
			log.Printf("ChangeDealPeriod: Warning - could not stat original file: %v", err)
		}
		
		fileData, err = os.ReadFile(originalFilePath)
		if err != nil {
			log.Printf("ChangeDealPeriod: Warning - could not read original file: %v", err)
			// Continue without file - not a critical error
		}
	}
	
	// Save file in new period if exists
	if len(fileData) > 0 && originalDeal.FilePath != "" {
		// Generate new file path
		ext := filepath.Ext(originalDeal.FilePath)
		newFileName := fmt.Sprintf("%s_%s_%s_%d%s",
			newDeal.NO,
			newDeal.DealDate,
			strings.ReplaceAll(newDeal.DealPartner, "/", "_"),
			newDeal.DealPrice,
			ext)
		
		// Ensure target period directory exists
		targetPeriodDir := filepath.Join(models.GetBasePath(), req.ToPeriod)
		if err := os.MkdirAll(targetPeriodDir, 0755); err != nil {
			log.Printf("ChangeDealPeriod: Failed to create target directory: %v", err)
			
			// Rollback: restore original deal status
			models.ConnectToPeriod(req.FromPeriod)
			originalDeal.RecStatus = "NEW"
			models.UpdateDeal(dealID, originalDeal)
			
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "directory_create_error",
				"message": "Failed to create target period directory",
			})
			return
		}
		
		// Save file in new location
		newFilePath := filepath.Join(targetPeriodDir, newFileName)
		if err := utils.SaveFileAtomic(newFilePath, fileData); err != nil {
			log.Printf("ChangeDealPeriod: Failed to save file in new period: %v", err)
			
			// Rollback: restore original deal status
			models.ConnectToPeriod(req.FromPeriod)
			originalDeal.RecStatus = "NEW"
			models.UpdateDeal(dealID, originalDeal)
			
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "file_save_error",
				"message": "Failed to save file in target period",
			})
			return
		}
		
		// Preserve original file timestamps if we have the info
		if originalFileInfo != nil {
			modTime := originalFileInfo.ModTime()
			// For Windows, we need to preserve both access time and modified time
			// Using the modified time for both since Go's os.Chtimes doesn't expose creation time
			if err := os.Chtimes(newFilePath, modTime, modTime); err != nil {
				log.Printf("ChangeDealPeriod: Warning - could not preserve file timestamps: %v", err)
				// Not a critical error, continue
			}
		}
		
		newDeal.FilePath = newFileName
	}
	
	// Step 3: Create the new deal in target period
	if err := models.CreateDeal(&newDeal); err != nil {
		log.Printf("ChangeDealPeriod: Failed to create deal in target period: %v", err)
		
		// Rollback: delete copied file if it was created
		if newDeal.FilePath != "" {
			os.Remove(filepath.Join(models.GetBasePath(), req.ToPeriod, newDeal.FilePath))
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "database_error",
			"message": "Failed to create deal in target period",
		})
		return
	}
	
	// Step 4: Now mark the original deal as DELETE (only after successful creation in new period)
	if err := models.ConnectToPeriod(req.FromPeriod); err != nil {
		log.Printf("ChangeDealPeriod: Warning - could not reconnect to source period to mark as DELETE: %v", err)
		// New deal is already created, so we continue but log the warning
	} else {
		originalDeal.RecStatus = "DELETE"
		originalDeal.RecUpdate = time.Now().Format("2006-01-02T15:04:05Z")
		
		if err := models.UpdateDeal(dealID, originalDeal); err != nil {
			log.Printf("ChangeDealPeriod: Warning - could not mark original deal as DELETE: %v", err)
			// New deal is already created, so this is not critical
		}
	}
	
	log.Printf("ChangeDealPeriod: Successfully moved deal %s to period %s with new ID %s", dealID, req.ToPeriod, newDeal.NO)
	
	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"message":      "Deal period changed successfully",
		"originalNo":   dealID,
		"newNo":        newDeal.NO,
		"fromPeriod":   req.FromPeriod,
		"toPeriod":     req.ToPeriod,
		"fileMoved":    newDeal.FilePath != "",
	})
}