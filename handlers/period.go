package handlers

import (
	"denchokun-api/models"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func GetPeriods(c *gin.Context) {
	periods, err := models.GetAllPeriodsWithDetails()
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
		"periods": periods,
	})
}

func GetPeriod(c *gin.Context) {
	period := c.Param("period")
	if period == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"message": "Period parameter is required",
		})
		return
	}

	periodData, err := models.GetPeriodByName(period)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "failed to connect") {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "not_found",
				"message": "Period not found: " + period,
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
		"period":  periodData,
	})
}

func ConnectPeriod(c *gin.Context) {
	period := c.Param("period")
	if period == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"message": "Period parameter is required",
		})
		return
	}

	err := models.ConnectToPeriod(period)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "connection_error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"message":      "Connected to period " + period,
		"databasePath": "./data/" + period + "/Denchokun.db",
	})
}


// CreatePeriod creates a new period
func CreatePeriod(c *gin.Context) {
	var req models.PeriodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	period, err := models.CreatePeriod(&req)
	if err != nil {
		if strings.Contains(err.Error(), "format") || strings.Contains(err.Error(), "required") {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "validation_error",
				"message": err.Error(),
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "database_error",
				"message": err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Period created successfully",
		"period":  period,
	})
}

// UpdatePeriodDates updates a specific period's dates
func UpdatePeriodDates(c *gin.Context) {
	periodName := c.Param("period")
	if periodName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"message": "Period parameter is required",
		})
		return
	}

	var req models.PeriodUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	period, err := models.UpdatePeriod(periodName, &req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "period_not_found",
				"message": err.Error(),
			})
		} else if strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusConflict, gin.H{
				"success": false,
				"error":   "period_already_exists",
				"message": err.Error(),
			})
		} else if strings.Contains(err.Error(), "format") || strings.Contains(err.Error(), "range") {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "validation_error",
				"message": err.Error(),
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "update_error",
				"message": err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Period updated successfully",
		"period":  period,
	})
}


// UpdatePeriodName updates a period's name
func UpdatePeriodName(c *gin.Context) {
	periodName := c.Param("period")
	if periodName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"message": "Period parameter is required",
		})
		return
	}

	var req models.PeriodRenameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	period, err := models.RenamePeriod(periodName, req.NewName)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "period_not_found",
				"message": err.Error(),
			})
		} else if strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusConflict, gin.H{
				"success": false,
				"error":   "period_already_exists",
				"message": err.Error(),
			})
		} else if strings.Contains(err.Error(), "format") {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "validation_error",
				"message": err.Error(),
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "rename_error",
				"message": err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Period renamed successfully",
		"period":  period,
	})
}

