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
	
	dbMutex.Lock()
	defer dbMutex.Unlock()

	if db, exists := dbConnections[period]; exists {
		currentDB = db
		currentPeriod = period
		return nil
	}

	periodPath := filepath.Join(basePath, period)
	if err := os.MkdirAll(periodPath, 0755); err != nil {
		return fmt.Errorf("failed to create period directory: %v", err)
	}

	dbPath := filepath.Join(periodPath, "Denchokun.db")
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=30000")
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

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
		`CREATE INDEX IF NOT EXISTS idx_Hash ON Deals (Hash)`,
		`CREATE INDEX IF NOT EXISTS idx_deal_date ON Deals(DealDate)`,
		`CREATE INDEX IF NOT EXISTS idx_deal_partner ON Deals(DealPartner)`,
		`CREATE INDEX IF NOT EXISTS idx_deal_type ON Deals(DealType)`,
		`CREATE TABLE IF NOT EXISTS "System" (
			"AppVersion" TEXT,
			"SQLiteLibraryVersion" TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS DealPartners (
			name TEXT PRIMARY KEY
		)`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}

	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM System").Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		_, err = db.Exec("INSERT INTO System (AppVersion, SQLiteLibraryVersion) VALUES (?, ?)",
			"1.0.0", "3.40.0")
		if err != nil {
			return err
		}
	}

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