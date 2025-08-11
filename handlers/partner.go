package handlers

import (
	"denchokun-api/models"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func GetDealPartners(c *gin.Context) {
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

	partners, err := models.GetDealPartners()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "database_error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"partners": partners,
	})
}

func CreateDealPartner(c *gin.Context) {
	var partner models.DealPartner
	if err := c.ShouldBindJSON(&partner); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"message": err.Error(),
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

	if err := models.CreateDealPartner(&partner); err != nil {
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
		"success": true,
		"message": "Partner created successfully",
		"name":    partner.Name,
	})
}

func UpdateDealPartner(c *gin.Context) {
	oldName := c.Param("name")
	if oldName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"message": "Partner name is required",
		})
		return
	}

	var req struct {
		NewName string `json:"newName" binding:"required"`
		Period  string `json:"period"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	if req.Period != "" {
		if err := models.ConnectToPeriod(req.Period); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "connection_error",
				"message": err.Error(),
			})
			return
		}
	}

	if err := models.UpdateDealPartner(oldName, req.NewName); err != nil {
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
		"message": "Partner updated successfully",
	})
}

func DeleteDealPartner(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"message": "Partner name is required",
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

	if err := models.DeleteDealPartner(name); err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "not_found",
				"message": err.Error(),
			})
			return
		}

		if strings.Contains(err.Error(), "cannot delete") {
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
		"success": true,
		"message": "Partner deleted successfully",
	})
}