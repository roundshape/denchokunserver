package models

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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

type PeriodUpdateRequest struct {
	FromDate string `json:"fromDate,omitempty"`
	ToDate   string `json:"toDate,omitempty"`
}

type PeriodRenameRequest struct {
	NewName string `json:"newName" binding:"required"`
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
	// Connect to the period's database
	if err := ConnectToPeriod(name); err != nil {
		return nil, fmt.Errorf("failed to connect to period %s: %v", name, err)
	}
	
	db, err := GetDB()
	if err != nil {
		return nil, err
	}

	query := `SELECT fromDate, toDate, created, updated FROM Period LIMIT 1`
	var period Period
	period.Name = name
	
	err = db.QueryRow(query).Scan(&period.FromDate, &period.ToDate, &period.Created, &period.Updated)
	if err != nil {
		if err == sql.ErrNoRows {
			// Period table exists but no record, return with defaults
			period.FromDate = "未設定"
			period.ToDate = "未設定"
			period.Created = time.Now().Format(time.RFC3339)
			period.Updated = time.Now().Format(time.RFC3339)
			return &period, nil
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
	// Connect to the period's database
	if err := ConnectToPeriod(period.Name); err != nil {
		return fmt.Errorf("failed to connect to period %s: %v", period.Name, err)
	}
	
	db, err := GetDB()
	if err != nil {
		return err
	}

	// First delete any existing record (should be only one)
	_, _ = db.Exec(`DELETE FROM Period`)
	
	// Insert the period record
	query := `INSERT INTO Period (fromDate, toDate, created, updated) 
			  VALUES (?, ?, ?, ?)`
	
	_, err = db.Exec(query, period.FromDate, period.ToDate, period.Created, period.Updated)
	if err != nil {
		return fmt.Errorf("failed to insert period: %v", err)
	}

	return nil
}

// UpdatePeriods synchronizes the Period tables in each Denchokun.db
// This function:
// 1. Gets all directories under data/
// 2. For each directory with Denchokun.db, ensures Period table has a record
func UpdatePeriods() ([]Period, error) {
	// Get all available periods (directories with Denchokun.db)
	availablePeriods, err := GetAvailablePeriods()
	if err != nil {
		return nil, err
	}

	// Check each directory and ensure Period record exists
	now := time.Now().Format(time.RFC3339)
	for _, dirName := range availablePeriods {
		// Try to get period details
		period, err := GetPeriodByName(dirName)
		if err != nil || period.Created == "" {
			// Create default period record if not exists
			period = &Period{
				Name:     dirName,
				FromDate: "未設定",
				ToDate:   "未設定",
				Created:  now,
				Updated:  now,
			}
			if err := CreatePeriodRecord(period); err != nil {
				// Log error but continue
				fmt.Printf("Warning: failed to create period record for %s: %v\n", dirName, err)
			}
		}
	}

	// Return all periods with details
	return GetAllPeriodsWithDetails()
}

// UpdatePeriod updates period dates only
func UpdatePeriod(periodName string, req *PeriodUpdateRequest) (*Period, error) {
	// Get existing period
	existing, err := GetPeriodByName(periodName)
	if err != nil {
		return nil, err
	}

	// Validate dates if provided
	if req.FromDate != "" && req.ToDate != "" && req.FromDate != "未設定" && req.ToDate != "未設定" {
		if err := ValidateDateRange(req.FromDate, req.ToDate); err != nil {
			return nil, err
		}
	}

	// Update dates if provided
	if req.FromDate != "" {
		existing.FromDate = req.FromDate
	}
	if req.ToDate != "" {
		existing.ToDate = req.ToDate
	}
	existing.Updated = time.Now().Format(time.RFC3339)

	// Update in database
	if err := ConnectToPeriod(periodName); err != nil {
		return nil, fmt.Errorf("failed to connect to period %s: %v", periodName, err)
	}
	
	db, err := GetDB()
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`UPDATE Period SET fromDate = ?, toDate = ?, updated = ?`,
		existing.FromDate, existing.ToDate, existing.Updated)
	if err != nil {
		return nil, fmt.Errorf("failed to update period: %v", err)
	}

	return existing, nil
}

// RenamePeriod renames a period (changes its directory name)
func RenamePeriod(oldName string, newName string) (*Period, error) {
	// Validate new name - allow any non-empty string
	if newName == "" {
		return nil, fmt.Errorf("new period name cannot be empty")
	}
	
	// Check for invalid file system characters
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	for _, char := range invalidChars {
		if strings.Contains(newName, char) {
			return nil, fmt.Errorf("period name cannot contain special characters: %s", char)
		}
	}

	// Check if new name already exists
	newPath := filepath.Join(GetBasePath(), newName)
	if _, err := os.Stat(newPath); err == nil {
		return nil, fmt.Errorf("period %s already exists", newName)
	}

	// Get existing period before closing DB
	existing, err := GetPeriodByName(oldName)
	if err != nil {
		return nil, err
	}

	// Close database connection before renaming directory
	if err := ClosePeriodDB(oldName); err != nil {
		return nil, fmt.Errorf("failed to close database connection: %v", err)
	}

	// Rename directory
	oldPath := filepath.Join(GetBasePath(), oldName)
	
	if _, err := os.Stat(oldPath); err == nil {
		if err := os.Rename(oldPath, newPath); err != nil {
			return nil, fmt.Errorf("failed to rename directory: %v", err)
		}
	}

	// Update period record with new name
	existing.Name = newName
	existing.Updated = time.Now().Format(time.RFC3339)

	// The Period table stays in the renamed directory
	// No need to update it since it doesn't store the period name

	return existing, nil
}

// DeletePeriod deletes a period if it has no deals
func DeletePeriod(name string) error {
	// Check if period exists and has deals
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

	// Period record is in the Denchokun.db itself, so deleting the database file removes it
	// No need to delete from System.db as Periods table no longer exists there

	return nil
}

// ValidatePeriodRequest validates a period request
func ValidatePeriodRequest(req *PeriodRequest) error {
	if req.Name == "" {
		return fmt.Errorf("period name is required")
	}

	// Check for invalid file system characters
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	for _, char := range invalidChars {
		if strings.Contains(req.Name, char) {
			return fmt.Errorf("period name cannot contain special characters: %s", char)
		}
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