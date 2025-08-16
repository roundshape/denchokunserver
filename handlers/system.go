package handlers

import (
	"denchokun-api/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

type SystemInfo struct {
	AppVersion           string `json:"appVersion"`
	SQLiteLibraryVersion string `json:"sqliteLibraryVersion"`
}

// GetSystemInfo handles GET /system
func GetSystemInfo(c *gin.Context) {
	appVersion, sqliteVersion, err := models.GetSystemInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"system": SystemInfo{
			AppVersion:           appVersion,
			SQLiteLibraryVersion: sqliteVersion,
		},
	})
}

// UpdateSystemInfo handles PUT /system
func UpdateSystemInfo(c *gin.Context) {
	var systemInfo SystemInfo
	if err := c.ShouldBindJSON(&systemInfo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
		})
		return
	}

	if err := models.UpdateSystemInfo(systemInfo.AppVersion, systemInfo.SQLiteLibraryVersion); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "System info updated successfully",
	})
}