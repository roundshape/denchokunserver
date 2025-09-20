package models

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

var (
	dbConnections map[string]*sql.DB
	currentDB     *sql.DB
	currentPeriod string
	systemDB      *sql.DB  // System.db connection
	dbMutex       sync.RWMutex
	basePath      string
)

// GetBasePath returns the base path for data storage
func GetBasePath() string {
	return basePath
}

// ClosePeriodDB closes database connection for a specific period
func ClosePeriodDB(period string) error {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	if db, exists := dbConnections[period]; exists {
		err := db.Close()
		delete(dbConnections, period)
		
		// If this was the current period, reset current connection
		if currentPeriod == period {
			currentDB = nil
			currentPeriod = ""
		}
		
		return err
	}
	
	return nil // No connection to close
}

// CloseAllPeriodDBs closes all database connections
func CloseAllPeriodDBs() error {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	var lastErr error
	
	// Close all period database connections
	for period, db := range dbConnections {
		if err := db.Close(); err != nil {
			lastErr = err
		}
		delete(dbConnections, period)
	}
	
	// Reset current connection
	currentDB = nil
	currentPeriod = ""
	
	return lastErr
}

func InitDB(path string) error {
	basePath = path
	dbConnections = make(map[string]*sql.DB)

	if err := os.MkdirAll(basePath, 0755); err != nil {
		return fmt.Errorf("failed to create base directory: %v", err)
	}

	// Initialize System.db
	if err := initSystemDB(); err != nil {
		return fmt.Errorf("failed to initialize system database: %v", err)
	}

	return nil
}

func initSystemDB() error {
	systemDBPath := filepath.Join(basePath, "System.db")
	
	db, err := sql.Open("sqlite", systemDBPath+"?_journal_mode=WAL&_busy_timeout=30000")
	if err != nil {
		return fmt.Errorf("failed to open system database: %v", err)
	}

	// Create Periods table in System.db
	query := `CREATE TABLE IF NOT EXISTS "Periods" (
		"name" TEXT PRIMARY KEY,
		"fromDate" TEXT,
		"toDate" TEXT,
		"created" TEXT,
		"updated" TEXT
	)`
	
	if _, err := db.Exec(query); err != nil {
		db.Close()
		return fmt.Errorf("failed to create Periods table: %v", err)
	}

	// Create DealPartners table in System.db
	partnerQuery := `CREATE TABLE IF NOT EXISTS "DealPartners" (
		"name" TEXT PRIMARY KEY
	)`
	
	if _, err := db.Exec(partnerQuery); err != nil {
		db.Close()
		return fmt.Errorf("failed to create DealPartners table: %v", err)
	}

	// Create System table in System.db (for app version info)
	systemQuery := `CREATE TABLE IF NOT EXISTS "System" (
		"AppVersion" TEXT,
		"SQLiteLibraryVersion" TEXT
	)`
	
	if _, err := db.Exec(systemQuery); err != nil {
		db.Close()
		return fmt.Errorf("failed to create System table: %v", err)
	}

	// Initialize System table if empty
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM System").Scan(&count)
	if err != nil {
		db.Close()
		return fmt.Errorf("failed to check System table: %v", err)
	}

	if count == 0 {
		_, err = db.Exec("INSERT INTO System (AppVersion, SQLiteLibraryVersion) VALUES (?, ?)",
			"1.0.0", "3.40.0")
		if err != nil {
			db.Close()
			return fmt.Errorf("failed to initialize System table: %v", err)
		}
	}

	systemDB = db
	return nil
}

func GetSystemDB() (*sql.DB, error) {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	if systemDB == nil {
		return nil, fmt.Errorf("system database not initialized")
	}

	return systemDB, nil
}

func ConnectToPeriod(period string) error {
	fmt.Printf("ConnectToPeriod: Starting connection to period %s\n", period)
	
	dbMutex.Lock()
	defer dbMutex.Unlock()

	if db, exists := dbConnections[period]; exists {
		fmt.Printf("ConnectToPeriod: Using existing connection for period %s\n", period)
		currentDB = db
		currentPeriod = period
		return nil
	}

	periodPath := filepath.Join(basePath, period)
	fmt.Printf("ConnectToPeriod: Period path: %s\n", periodPath)
	
	// Create period directory if it doesn't exist
	if _, err := os.Stat(periodPath); os.IsNotExist(err) {
		fmt.Printf("ConnectToPeriod: Creating period directory: %s\n", periodPath)
		if err := os.MkdirAll(periodPath, 0755); err != nil {
			return fmt.Errorf("failed to create period directory: %v", err)
		}
	}

	dbPath := filepath.Join(periodPath, "Denchokun.db")
	fmt.Printf("ConnectToPeriod: Database path: %s\n", dbPath)
	
	// Database will be created if it doesn't exist (SQLite behavior)
	
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=30000")
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	fmt.Printf("ConnectToPeriod: Setting up database for period %s\n", period)
	if err := setupDatabase(db); err != nil {
		db.Close()
		return fmt.Errorf("failed to setup database: %v", err)
	}

	dbConnections[period] = db
	currentDB = db
	currentPeriod = period

	return nil
}

func setupDatabase(db *sql.DB) error {
	fmt.Println("setupDatabase: Starting database setup")
	queries := []string{
		`CREATE TABLE IF NOT EXISTS "Deals" (
			"NO" TEXT NOT NULL UNIQUE,
			"nextNO" TEXT,
			"prevNO" TEXT,
			"DealType" TEXT,
			"DealDate" TEXT,
			"DealName" TEXT,
			"DealPartner" TEXT,
			"DealPrice" INTEGER,
			"DealRemark" TEXT,
			"RecUpdate" TEXT,
			"RegDate" TEXT,
			"RecStatus" TEXT,
			"FilePath" TEXT,
			"Hash" TEXT,
			PRIMARY KEY("NO")
		)`,
		`CREATE TABLE IF NOT EXISTS "Period" (
			"fromDate" TEXT,
			"toDate" TEXT,
			"created" TEXT,
			"updated" TEXT
		)`,
		`INSERT INTO Period (fromDate, toDate, created, updated)
		 SELECT '未設定', '未設定', datetime('now'), datetime('now')
		 WHERE NOT EXISTS (SELECT 1 FROM Period)`,
		`CREATE INDEX IF NOT EXISTS idx_Hash ON Deals (Hash)`,
		`CREATE INDEX IF NOT EXISTS idx_deal_date ON Deals(DealDate)`,
		`CREATE INDEX IF NOT EXISTS idx_deal_partner ON Deals(DealPartner)`,
		`CREATE INDEX IF NOT EXISTS idx_deal_type ON Deals(DealType)`,
	}

	for i, query := range queries {
		fmt.Printf("setupDatabase: Executing query %d\n", i+1)
		if _, err := db.Exec(query); err != nil {
			fmt.Printf("setupDatabase: Query %d failed: %v\n", i+1, err)
			return err
		}
	}

	fmt.Println("setupDatabase: Database setup completed successfully")
	return nil
}

func GetDB() (*sql.DB, error) {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	if currentDB == nil {
		return nil, fmt.Errorf("no database connection")
	}

	return currentDB, nil
}

func GetCurrentPeriod() string {
	dbMutex.RLock()
	defer dbMutex.RUnlock()
	return currentPeriod
}

func ConnectPeriodDB(period string) (*sql.DB, error) {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	// Check if already connected
	if db, exists := dbConnections[period]; exists {
		return db, nil
	}

	// Connect to the period database
	periodPath := filepath.Join(basePath, period)
	dbPath := filepath.Join(periodPath, "Denchokun.db")
	
	// Check if database exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("database for period %s does not exist", period)
	}

	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=30000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	if err := setupDatabase(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to setup database: %v", err)
	}

	dbConnections[period] = db
	return db, nil
}

func GetAvailablePeriods() ([]string, error) {
	entries, err := os.ReadDir(basePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var periods []string
	for _, entry := range entries {
		if entry.IsDir() {
			dbPath := filepath.Join(basePath, entry.Name(), "Denchokun.db")
			if _, err := os.Stat(dbPath); err == nil {
				periods = append(periods, entry.Name())
			}
		}
	}

	return periods, nil
}

func CloseAllConnections() {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	for _, db := range dbConnections {
		db.Close()
	}
	dbConnections = make(map[string]*sql.DB)
	currentDB = nil
	currentPeriod = ""
}

// MigrateToSystemDB migrates DealPartners and System tables from period DBs to System.db
func MigrateToSystemDB() error {
	systemDB, err := GetSystemDB()
	if err != nil {
		return fmt.Errorf("failed to get system database: %v", err)
	}

	// Get all available periods
	periods, err := GetAvailablePeriods()
	if err != nil {
		return fmt.Errorf("failed to get periods: %v", err)
	}

	partnersSet := make(map[string]bool)

	// Collect all unique partners from all period databases
	for _, period := range periods {
		periodDB, err := ConnectPeriodDB(period)
		if err != nil {
			// Skip if period DB doesn't exist or can't be opened
			continue
		}

		// Check if DealPartners table exists in this period DB
		var tableExists int
		err = periodDB.QueryRow(`
			SELECT COUNT(*) FROM sqlite_master 
			WHERE type='table' AND name='DealPartners'
		`).Scan(&tableExists)
		if err != nil || tableExists == 0 {
			continue
		}

		// Get all partners from this period DB
		rows, err := periodDB.Query("SELECT name FROM DealPartners")
		if err != nil {
			continue
		}

		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err == nil {
				partnersSet[name] = true
			}
		}
		rows.Close()
	}

	// Insert unique partners into System.db
	tx, err := systemDB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT OR IGNORE INTO DealPartners (name) VALUES (?)")
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	for partner := range partnersSet {
		if _, err := stmt.Exec(partner); err != nil {
			return fmt.Errorf("failed to insert partner %s: %v", partner, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	// Now migrate System table data (if it exists in any period DB)
	// We'll take the latest version info from any period DB
	var latestAppVersion, latestSQLiteVersion string
	for _, period := range periods {
		periodDB, err := ConnectPeriodDB(period)
		if err != nil {
			continue
		}

		// Check if System table exists
		var tableExists int
		err = periodDB.QueryRow(`
			SELECT COUNT(*) FROM sqlite_master 
			WHERE type='table' AND name='System'
		`).Scan(&tableExists)
		if err != nil || tableExists == 0 {
			continue
		}

		// Get System info from this period DB
		var appVersion, sqliteVersion string
		err = periodDB.QueryRow("SELECT AppVersion, SQLiteLibraryVersion FROM System LIMIT 1").Scan(&appVersion, &sqliteVersion)
		if err == nil && appVersion != "" {
			latestAppVersion = appVersion
			latestSQLiteVersion = sqliteVersion
		}
	}

	// Update System table in System.db if we found newer version info
	if latestAppVersion != "" {
		_, err = systemDB.Exec("UPDATE System SET AppVersion = ?, SQLiteLibraryVersion = ? WHERE 1=1",
			latestAppVersion, latestSQLiteVersion)
		if err != nil {
			return fmt.Errorf("failed to update System table: %v", err)
		}
	}

	// Drop DealPartners and System tables from all period databases
	for _, period := range periods {
		periodDB, err := ConnectPeriodDB(period)
		if err != nil {
			continue
		}

		// Drop the tables if they exist
		_, _ = periodDB.Exec("DROP TABLE IF EXISTS DealPartners")
		_, _ = periodDB.Exec("DROP TABLE IF EXISTS System")
	}

	return nil
}

// GetSystemInfo returns the system information from System.db
func GetSystemInfo() (appVersion string, sqliteVersion string, err error) {
	systemDB, err := GetSystemDB()
	if err != nil {
		return "", "", fmt.Errorf("failed to get system database: %v", err)
	}

	err = systemDB.QueryRow("SELECT AppVersion, SQLiteLibraryVersion FROM System LIMIT 1").Scan(&appVersion, &sqliteVersion)
	if err != nil {
		return "", "", fmt.Errorf("failed to get system info: %v", err)
	}

	return appVersion, sqliteVersion, nil
}

// UpdateSystemInfo updates the system information in System.db
func UpdateSystemInfo(appVersion, sqliteVersion string) error {
	systemDB, err := GetSystemDB()
	if err != nil {
		return fmt.Errorf("failed to get system database: %v", err)
	}

	_, err = systemDB.Exec("UPDATE System SET AppVersion = ?, SQLiteLibraryVersion = ? WHERE 1=1",
		appVersion, sqliteVersion)
	if err != nil {
		return fmt.Errorf("failed to update system info: %v", err)
	}

	return nil
}