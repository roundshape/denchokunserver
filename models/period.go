package models

import (
	"database/sql"
	"fmt"
	"regexp"
	"time"
)

type Period struct {
	Name     string `json:"name" binding:"required"`
	FromDate string `json:"fromDate"`
	ToDate   string `json:"toDate"`
	Created  string `json:"created"`
	Updated  string `json:"updated"`
}

type PeriodRequest struct {
	Name     string `json:"name,omitempty"`
	FromDate string `json:"fromDate"`
	ToDate   string `json:"toDate"`
}

// GetAllPeriodsWithDetails returns all periods with their details
func GetAllPeriodsWithDetails() ([]Period, error) {
	// First get directories that have Denchokun.db
	availablePeriods, err := GetAvailablePeriods()
	if err != nil {
		return nil, err
	}

	var periods []Period
	for _, periodName := range availablePeriods {
		// Try to get period details from Periods table
		period, err := GetPeriodByName(periodName)
		if err != nil {
			// If not found in Periods table, create default entry
			period = &Period{
				Name:     periodName,
				FromDate: "未設定",
				ToDate:   "未設定",
				Created:  time.Now().Format(time.RFC3339),
				Updated:  time.Now().Format(time.RFC3339),
			}
			// Insert into database
			CreatePeriodRecord(period)
		}
		periods = append(periods, *period)
	}

	return periods, nil
}

// GetPeriodByName returns a specific period by name
func GetPeriodByName(name string) (*Period, error) {
	db, err := GetSystemDB()
	if err != nil {
		return nil, err
	}

	query := `SELECT name, fromDate, toDate, created, updated FROM Periods WHERE name = ?`
	var period Period
	
	err = db.QueryRow(query, name).Scan(&period.Name, &period.FromDate, &period.ToDate, &period.Created, &period.Updated)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("period not found: %s", name)
		}
		return nil, fmt.Errorf("failed to get period: %v", err)
	}

	return &period, nil
}

// CreatePeriod creates a new period
func CreatePeriod(req *PeriodRequest) (*Period, error) {
	// Validate request
	if err := ValidatePeriodRequest(req); err != nil {
		return nil, err
	}

	// Create period database and connect
	if err := ConnectToPeriod(req.Name); err != nil {
		return nil, fmt.Errorf("failed to create period database: %v", err)
	}

	now := time.Now().Format(time.RFC3339)
	period := &Period{
		Name:     req.Name,
		FromDate: req.FromDate,
		ToDate:   req.ToDate,
		Created:  now,
		Updated:  now,
	}

	// Insert into Periods table
	if err := CreatePeriodRecord(period); err != nil {
		return nil, err
	}

	return period, nil
}

// CreatePeriodRecord inserts a period record into the database
func CreatePeriodRecord(period *Period) error {
	db, err := GetSystemDB()
	if err != nil {
		return err
	}

	query := `INSERT OR REPLACE INTO Periods (name, fromDate, toDate, created, updated) 
			  VALUES (?, ?, ?, ?, ?)`
	
	_, err = db.Exec(query, period.Name, period.FromDate, period.ToDate, period.Created, period.Updated)
	if err != nil {
		return fmt.Errorf("failed to insert period: %v", err)
	}

	return nil
}

// UpdatePeriod updates an existing period
func UpdatePeriod(name string, req *PeriodRequest) (*Period, error) {
	// Get existing period
	existing, err := GetPeriodByName(name)
	if err != nil {
		return nil, err
	}

	// Validate dates
	if req.FromDate != "" && req.ToDate != "" && req.FromDate != "未設定" && req.ToDate != "未設定" {
		if err := ValidateDateRange(req.FromDate, req.ToDate); err != nil {
			return nil, err
		}
	}

	// Update fields
	if req.FromDate != "" {
		existing.FromDate = req.FromDate
	}
	if req.ToDate != "" {
		existing.ToDate = req.ToDate
	}
	existing.Updated = time.Now().Format(time.RFC3339)

	// Update in database
	db, dbErr := GetSystemDB()
	if dbErr != nil {
		return nil, dbErr
	}
	query := `UPDATE Periods SET fromDate = ?, toDate = ?, updated = ? WHERE name = ?`
	
	_, err = db.Exec(query, existing.FromDate, existing.ToDate, existing.Updated, name)
	if err != nil {
		return nil, fmt.Errorf("failed to update period: %v", err)
	}

	return existing, nil
}

// DeletePeriod deletes a period if it has no deals
func DeletePeriod(name string) error {
	// First check if period exists and has deals
	if err := ConnectToPeriod(name); err != nil {
		return fmt.Errorf("failed to connect to period: %v", err)
	}

	periodDB, err := GetDB()
	if err != nil {
		return err
	}

	// Check if period has deals
	var dealCount int
	err = periodDB.QueryRow("SELECT COUNT(*) FROM Deals").Scan(&dealCount)
	if err != nil {
		return fmt.Errorf("failed to check deals: %v", err)
	}

	if dealCount > 0 {
		return fmt.Errorf("period_has_deals")
	}

	// Delete from Periods table in System.db
	systemDB, err := GetSystemDB()
	if err != nil {
		return err
	}
	
	_, err = systemDB.Exec("DELETE FROM Periods WHERE name = ?", name)
	if err != nil {
		return fmt.Errorf("failed to delete period: %v", err)
	}

	return nil
}

// ValidatePeriodRequest validates a period request
func ValidatePeriodRequest(req *PeriodRequest) error {
	if req.Name == "" {
		return fmt.Errorf("period name is required")
	}

	// Validate name format (YYYY-MM recommended)
	nameRegex := regexp.MustCompile(`^\d{4}-\d{2}$`)
	if !nameRegex.MatchString(req.Name) {
		return fmt.Errorf("period name should follow YYYY-MM format (e.g., 2024-01)")
	}

	// Validate date formats
	if req.FromDate != "" && req.FromDate != "未設定" {
		if !IsValidDate(req.FromDate) {
			return fmt.Errorf("invalid fromDate format, use YYYY-MM-DD")
		}
	}

	if req.ToDate != "" && req.ToDate != "未設定" {
		if !IsValidDate(req.ToDate) {
			return fmt.Errorf("invalid toDate format, use YYYY-MM-DD")
		}
	}

	// Validate date range
	if req.FromDate != "" && req.ToDate != "" && req.FromDate != "未設定" && req.ToDate != "未設定" {
		if err := ValidateDateRange(req.FromDate, req.ToDate); err != nil {
			return err
		}
	}

	return nil
}

// IsValidDate checks if date string is in YYYY-MM-DD format
func IsValidDate(dateStr string) bool {
	dateRegex := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	if !dateRegex.MatchString(dateStr) {
		return false
	}

	// Try parsing to ensure it's a valid date
	_, err := time.Parse("2006-01-02", dateStr)
	return err == nil
}

// ValidateDateRange checks if fromDate <= toDate
func ValidateDateRange(fromDate, toDate string) error {
	from, err := time.Parse("2006-01-02", fromDate)
	if err != nil {
		return fmt.Errorf("invalid fromDate: %v", err)
	}

	to, err := time.Parse("2006-01-02", toDate)
	if err != nil {
		return fmt.Errorf("invalid toDate: %v", err)
	}

	if from.After(to) {
		return fmt.Errorf("fromDate must be before or equal to toDate")
	}

	return nil
}