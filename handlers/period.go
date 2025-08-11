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

// GetPeriod gets details for a specific period
func GetPeriod(c *gin.Context) {
	periodName := c.Param("period")
	if periodName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"message": "Period parameter is required",
		})
		return
	}

	period, err := models.GetPeriodByName(periodName)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "period_not_found",
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

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"period":  period,
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

// UpdatePeriod updates an existing period
func UpdatePeriod(c *gin.Context) {
	periodName := c.Param("period")
	if periodName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"message": "Period parameter is required",
		})
		return
	}

	var req models.PeriodRequest
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
		} else if strings.Contains(err.Error(), "format") || strings.Contains(err.Error(), "range") {
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

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Period updated successfully",
		"period":  period,
	})
}

// DeletePeriod deletes a period
func DeletePeriod(c *gin.Context) {
	periodName := c.Param("period")
	if periodName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"message": "Period parameter is required",
		})
		return
	}

	err := models.DeletePeriod(periodName)
	if err != nil {
		if strings.Contains(err.Error(), "period_has_deals") {
			c.JSON(http.StatusConflict, gin.H{
				"success": false,
				"error":   "period_has_deals",
				"message": "Cannot delete period that contains deal records",
			})
		} else if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "period_not_found",
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

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Period deleted successfully",
	})
}