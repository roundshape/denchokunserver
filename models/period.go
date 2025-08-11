package models

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
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

type PeriodUpdateRequest struct {
	NewName  string `json:"newName" binding:"required"`
	FromDate string `json:"fromDate,omitempty"`
	ToDate   string `json:"toDate,omitempty"`
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

// UpdatePeriods synchronizes the Periods table with the directories under data/
// This function:
// 1. Gets all directories under data/
// 2. For each directory, if it doesn't exist in Periods table, adds it with "未設定" dates
// 3. Returns error if a period in the table doesn't have a corresponding directory
func UpdatePeriods() ([]Period, error) {
	// Get all available periods (directories with Denchokun.db)
	availablePeriods, err := GetAvailablePeriods()
	if err != nil {
		return nil, err
	}

	// Get all periods from database
	db, err := GetSystemDB()
	if err != nil {
		return nil, err
	}

	query := `SELECT name FROM Periods`
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query periods: %v", err)
	}
	defer rows.Close()

	existingPeriods := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to scan period: %v", err)
		}
		existingPeriods[name] = false // false means directory not found yet
	}

	// Check each directory
	now := time.Now().Format(time.RFC3339)
	for _, dirName := range availablePeriods {
		if _, exists := existingPeriods[dirName]; exists {
			// Mark as found
			existingPeriods[dirName] = true
		} else {
			// Add new period to database
			period := &Period{
				Name:     dirName,
				FromDate: "未設定",
				ToDate:   "未設定",
				Created:  now,
				Updated:  now,
			}
			if err := CreatePeriodRecord(period); err != nil {
				return nil, fmt.Errorf("failed to add period %s: %v", dirName, err)
			}
		}
	}

	// Check for periods without directories
	var missingDirs []string
	for name, found := range existingPeriods {
		if !found {
			missingDirs = append(missingDirs, name)
		}
	}

	if len(missingDirs) > 0 {
		return nil, fmt.Errorf("periods without directories: %v", missingDirs)
	}

	// Return all periods with details
	return GetAllPeriodsWithDetails()

}

// UpdatePeriod updates a period name and optionally dates
func UpdatePeriod(oldName string, req *PeriodUpdateRequest) (*Period, error) {
	// Validate new name format
	if req.NewName != "" {
		nameRegex := regexp.MustCompile(`^\d{4}-\d{2}$`)
		if !nameRegex.MatchString(req.NewName) {
			return nil, fmt.Errorf("new period name should follow YYYY-MM format (e.g., 2024-01)")
		}
	}

	// Get existing period
	existing, err := GetPeriodByName(oldName)
	if err != nil {
		return nil, err
	}

	// Validate dates if provided
	if req.FromDate != "" && req.ToDate != "" && req.FromDate != "未設定" && req.ToDate != "未設定" {
		if err := ValidateDateRange(req.FromDate, req.ToDate); err != nil {
			return nil, err
		}
	}

	// Start transaction
	db, err := GetSystemDB()
	if err != nil {
		return nil, err
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %v", err)
	}
	defer tx.Rollback()

	// Update period name if changed
	needsRename := req.NewName != "" && req.NewName != oldName
	if needsRename {
		// Check if new name already exists
		var count int
		err = tx.QueryRow("SELECT COUNT(*) FROM Periods WHERE name = ?", req.NewName).Scan(&count)
		if err != nil {
			return nil, fmt.Errorf("failed to check new name: %v", err)
		}
		if count > 0 {
			return nil, fmt.Errorf("period %s already exists", req.NewName)
		}

		// Rename directory
		oldPath := filepath.Join(GetBasePath(), oldName)
		newPath := filepath.Join(GetBasePath(), req.NewName)
		
		if _, err := os.Stat(oldPath); err == nil {
			if err := os.Rename(oldPath, newPath); err != nil {
				return nil, fmt.Errorf("failed to rename directory: %v", err)
			}
		}
		
		existing.Name = req.NewName
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
	if needsRename {
		// Delete old record and insert new one (since name is primary key)
		_, err = tx.Exec("DELETE FROM Periods WHERE name = ?", oldName)
		if err != nil {
			return nil, fmt.Errorf("failed to delete old period: %v", err)
		}

		_, err = tx.Exec(`INSERT INTO Periods (name, fromDate, toDate, created, updated) 
						  VALUES (?, ?, ?, ?, ?)`,
			existing.Name, existing.FromDate, existing.ToDate, existing.Created, existing.Updated)
		if err != nil {
			return nil, fmt.Errorf("failed to insert new period: %v", err)
		}
	} else {
		// Just update dates
		_, err = tx.Exec(`UPDATE Periods SET fromDate = ?, toDate = ?, updated = ? WHERE name = ?`,
			existing.FromDate, existing.ToDate, existing.Updated, oldName)
		if err != nil {
			return nil, fmt.Errorf("failed to update period: %v", err)
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %v", err)
	}

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