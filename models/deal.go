package models

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Deal struct {
	NO          string  `json:"NO"`
	NextNO      *string `json:"nextNO,omitempty"`
	PrevNO      *string `json:"prevNO,omitempty"`
	DealType    string  `json:"DealType"`
	DealDate    string  `json:"DealDate"`
	DealName    string  `json:"DealName"`
	DealPartner string  `json:"DealPartner"`
	DealPrice   int     `json:"DealPrice"`
	DealRemark  string  `json:"DealRemark"`
	RecUpdate   string  `json:"RecUpdate"`
	RegDate     string  `json:"RegDate"`
	RecStatus   string  `json:"RecStatus"`
	FilePath    string  `json:"FilePath"`
	Hash        string  `json:"Hash"`
}

type DealFilter struct {
	Period    string `form:"period" binding:"required"`
	FromDate  string `form:"from_date"`
	ToDate    string `form:"to_date"`
	Partner   string `form:"partner"`
	Type      string `form:"type"`
	Keyword   string `form:"keyword"`
	View      string `form:"view"`  // "flat" or "history", default is "flat"
	Limit     int    `form:"limit"`
	Offset    int    `form:"offset"`
}

// DealWithHistory represents a deal with its update history
type DealWithHistory struct {
	Deal
	BaseNO      string  `json:"baseNO"`
	HasChildren bool    `json:"hasChildren"`
	ChildCount  int     `json:"childCount"`
	Children    []Deal  `json:"children,omitempty"`
}

func CreateDeal(deal *Deal) error {
	db, err := GetDB()
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %v", err)
	}
	defer tx.Rollback()

	var exists int
	err = tx.QueryRow("SELECT COUNT(*) FROM Deals WHERE NO = ?", deal.NO).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check existence: %v", err)
	}

	if exists > 0 {
		return fmt.Errorf("deal number already exists: %s", deal.NO)
	}

	now := time.Now().Format("2006-01-02T15:04:05Z")
	deal.RecUpdate = now
	deal.RegDate = now

	query := `INSERT INTO Deals (NO, nextNO, prevNO, DealType, DealDate, DealName, 
			  DealPartner, DealPrice, DealRemark, RecUpdate, RegDate, RecStatus, FilePath, Hash)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = tx.Exec(query, deal.NO, deal.NextNO, deal.PrevNO, deal.DealType, deal.DealDate,
		deal.DealName, deal.DealPartner, deal.DealPrice, deal.DealRemark,
		deal.RecUpdate, deal.RegDate, deal.RecStatus, deal.FilePath, deal.Hash)
	if err != nil {
		return fmt.Errorf("failed to insert deal: %v", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	return nil
}

func GetDealByID(dealID string) (*Deal, error) {
	db, err := GetDB()
	if err != nil {
		return nil, err
	}

	deal := &Deal{}
	query := `SELECT NO, nextNO, prevNO, DealType, DealDate, DealName, DealPartner, 
			  DealPrice, DealRemark, RecUpdate, RegDate, RecStatus, FilePath, Hash 
			  FROM Deals WHERE NO = ?`

	err = db.QueryRow(query, dealID).Scan(
		&deal.NO, &deal.NextNO, &deal.PrevNO, &deal.DealType, &deal.DealDate,
		&deal.DealName, &deal.DealPartner, &deal.DealPrice, &deal.DealRemark,
		&deal.RecUpdate, &deal.RegDate, &deal.RecStatus, &deal.FilePath, &deal.Hash)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("deal not found: %s", dealID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get deal: %v", err)
	}

	return deal, nil
}

func GetDeals(filter *DealFilter) ([]Deal, int, error) {
	db, err := GetDB()
	if err != nil {
		return nil, 0, err
	}

	// Default to "flat" view if not specified
	if filter.View == "" {
		filter.View = "flat"
	}

	query := "SELECT NO, nextNO, prevNO, DealType, DealDate, DealName, DealPartner, DealPrice, DealRemark, RecUpdate, RegDate, RecStatus, FilePath, Hash FROM Deals WHERE 1=1"
	countQuery := "SELECT COUNT(*) FROM Deals WHERE 1=1"
	args := []interface{}{}

	// Add view-specific conditions
	if filter.View == "flat" {
		// Include NEW and DELETE records, but only the latest version (nextNO IS NULL)
		query += " AND (RecStatus = 'NEW' OR RecStatus = 'DELETE') AND nextNO IS NULL"
		countQuery += " AND (RecStatus = 'NEW' OR RecStatus = 'DELETE') AND nextNO IS NULL"
	}

	if filter.FromDate != "" {
		query += " AND DealDate >= ?"
		countQuery += " AND DealDate >= ?"
		args = append(args, filter.FromDate)
	}

	if filter.ToDate != "" {
		query += " AND DealDate <= ?"
		countQuery += " AND DealDate <= ?"
		args = append(args, filter.ToDate)
	}

	if filter.Partner != "" {
		query += " AND DealPartner LIKE ?"
		countQuery += " AND DealPartner LIKE ?"
		args = append(args, "%"+filter.Partner+"%")
	}

	if filter.Type != "" {
		query += " AND DealType = ?"
		countQuery += " AND DealType = ?"
		args = append(args, filter.Type)
	}

	if filter.Keyword != "" {
		query += " AND (DealName LIKE ? OR DealRemark LIKE ? OR DealPartner LIKE ?)"
		countQuery += " AND (DealName LIKE ? OR DealRemark LIKE ? OR DealPartner LIKE ?)"
		args = append(args, "%"+filter.Keyword+"%", "%"+filter.Keyword+"%", "%"+filter.Keyword+"%")
	}

	var totalCount int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	err = db.QueryRow(countQuery, countArgs...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count deals: %v", err)
	}

	query += " ORDER BY DealDate DESC, NO DESC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	} else {
		query += " LIMIT 1000"
		args = append(args, 1000)
	}

	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query deals: %v", err)
	}
	defer rows.Close()

	deals := []Deal{}
	for rows.Next() {
		deal := Deal{}
		err := rows.Scan(
			&deal.NO, &deal.NextNO, &deal.PrevNO, &deal.DealType, &deal.DealDate,
			&deal.DealName, &deal.DealPartner, &deal.DealPrice, &deal.DealRemark,
			&deal.RecUpdate, &deal.RegDate, &deal.RecStatus, &deal.FilePath, &deal.Hash)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan deal: %v", err)
		}
		deals = append(deals, deal)
	}

	return deals, totalCount, nil
}

func UpdateDeal(dealID string, deal *Deal) error {
	db, err := GetDB()
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %v", err)
	}
	defer tx.Rollback()

	var exists int
	err = tx.QueryRow("SELECT COUNT(*) FROM Deals WHERE NO = ?", dealID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check existence: %v", err)
	}

	if exists == 0 {
		return fmt.Errorf("deal not found: %s", dealID)
	}

	deal.RecUpdate = time.Now().Format("2006-01-02T15:04:05Z")
	deal.RecStatus = "UPDATE"

	query := `UPDATE Deals SET nextNO=?, prevNO=?, DealType=?, DealDate=?, DealName=?, 
			  DealPartner=?, DealPrice=?, DealRemark=?, RecUpdate=?, RecStatus=?, 
			  FilePath=?, Hash=? WHERE NO=?`

	_, err = tx.Exec(query, deal.NextNO, deal.PrevNO, deal.DealType, deal.DealDate,
		deal.DealName, deal.DealPartner, deal.DealPrice, deal.DealRemark,
		deal.RecUpdate, deal.RecStatus, deal.FilePath, deal.Hash, dealID)
	if err != nil {
		return fmt.Errorf("failed to update deal: %v", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	return nil
}

func UpdateDealLinks(dealID string, recStatus string, nextNO string) error {
	db, err := GetDB()
	if err != nil {
		return err
	}

	query := `UPDATE Deals SET RecStatus=?, nextNO=?, RecUpdate=? WHERE NO=?`
	now := time.Now().Format("2006-01-02T15:04:05Z")
	
	var nextNOPtr *string
	if nextNO != "" {
		nextNOPtr = &nextNO
	}
	
	result, err := db.Exec(query, recStatus, nextNOPtr, now, dealID)
	if err != nil {
		return fmt.Errorf("failed to update deal links: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check affected rows: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("deal not found: %s", dealID)
	}

	return nil
}

// GetDealsByHash retrieves all deals with the specified hash value in the current period
// Only checks records with RecStatus = 'NEW' to avoid duplicate checking on UPDATE/DELETE records
func GetDealsByHash(hash string) ([]Deal, error) {
	if hash == "" {
		return []Deal{}, nil
	}

	db, err := GetDB()
	if err != nil {
		return nil, err
	}

	query := `SELECT NO, DealType, DealDate, DealName, DealPartner, DealPrice,
	          DealRemark, RecUpdate, RegDate, RecStatus, FilePath, Hash,
	          nextNO, prevNO
	          FROM Deals WHERE Hash = ? AND RecStatus = 'NEW'`

	rows, err := db.Query(query, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to query deals by hash: %v", err)
	}
	defer rows.Close()

	var deals []Deal
	for rows.Next() {
		var deal Deal
		err := rows.Scan(
			&deal.NO, &deal.DealType, &deal.DealDate, &deal.DealName,
			&deal.DealPartner, &deal.DealPrice, &deal.DealRemark,
			&deal.RecUpdate, &deal.RegDate, &deal.RecStatus,
			&deal.FilePath, &deal.Hash, &deal.NextNO, &deal.PrevNO,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan deal: %v", err)
		}
		deals = append(deals, deal)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %v", err)
	}

	return deals, nil
}

// DealWithPeriod represents a deal with its period information
type DealWithPeriod struct {
	Deal
	Period string `json:"period"`
}

// GetDealsByHashAllPeriods checks all periods for deals with the specified hash
// Returns a map with period as key and deals as value
func GetDealsByHashAllPeriods(hash string) ([]DealWithPeriod, error) {
	if hash == "" {
		return []DealWithPeriod{}, nil
	}

	// Get all available periods
	periods, err := GetAvailablePeriods()
	if err != nil {
		return nil, fmt.Errorf("failed to get periods: %v", err)
	}

	var allDeals []DealWithPeriod

	// Save current connection state
	originalDB := currentDB
	originalPeriod := currentPeriod

	// Check each period
	for _, period := range periods {
		// Connect to period database
		periodPath := filepath.Join(basePath, period)
		dbPath := filepath.Join(periodPath, "Denchokun.db")

		// Check if database file exists
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			continue // Skip if database doesn't exist
		}

		// Connect to this period
		if err := ConnectToPeriod(period); err != nil {
			// Log error but continue checking other periods
			fmt.Printf("Failed to connect to period %s: %v\n", period, err)
			continue
		}

		// Get deals with this hash in current period
		deals, err := GetDealsByHash(hash)
		if err != nil {
			// Log error but continue
			fmt.Printf("Failed to get deals for period %s: %v\n", period, err)
			continue
		}

		// Add period information to each deal
		for _, deal := range deals {
			allDeals = append(allDeals, DealWithPeriod{
				Deal:   deal,
				Period: period,
			})
		}
	}

	// Restore original connection
	if originalPeriod != "" {
		currentDB = originalDB
		currentPeriod = originalPeriod
	}

	return allDeals, nil
}

// CreateDealWithHistory creates a new deal record with history tracking in a single transaction
func CreateDealWithHistory(oldDealID string, newDeal *Deal) error {
	db, err := GetDB()
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %v", err)
	}
	defer tx.Rollback()

	// Step 1: Update old deal record (set RecStatus to UPDATE and nextNO to new deal number)
	// Use WHERE clause with RecStatus check to prevent updating already updated records
	updateQuery := `UPDATE Deals SET RecStatus=?, nextNO=?, RecUpdate=? 
	                WHERE NO=? AND RecStatus='NEW'`
	now := time.Now().Format("2006-01-02T15:04:05Z")
	
	result, err := tx.Exec(updateQuery, "UPDATE", newDeal.NO, now, oldDealID)
	if err != nil {
		return fmt.Errorf("failed to update old deal: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check affected rows: %v", err)
	}

	if rowsAffected == 0 {
		// Check if the deal exists but was already updated
		var recStatus string
		err = tx.QueryRow("SELECT RecStatus FROM Deals WHERE NO=?", oldDealID).Scan(&recStatus)
		if err == sql.ErrNoRows {
			return fmt.Errorf("old deal not found: %s", oldDealID)
		}
		if recStatus == "UPDATE" {
			return fmt.Errorf("deal %s has already been updated", oldDealID)
		}
		return fmt.Errorf("unable to update deal: %s", oldDealID)
	}

	// Step 2: Insert new deal record
	insertQuery := `INSERT INTO Deals (NO, nextNO, prevNO, DealType, DealDate, DealName, 
			  DealPartner, DealPrice, DealRemark, RecUpdate, RegDate, RecStatus, FilePath, Hash)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = tx.Exec(insertQuery, newDeal.NO, newDeal.NextNO, newDeal.PrevNO, newDeal.DealType, newDeal.DealDate,
		newDeal.DealName, newDeal.DealPartner, newDeal.DealPrice, newDeal.DealRemark,
		newDeal.RecUpdate, newDeal.RegDate, newDeal.RecStatus, newDeal.FilePath, newDeal.Hash)
	if err != nil {
		return fmt.Errorf("failed to insert new deal: %v", err)
	}

	// Step 3: Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	return nil
}

// GetDealsWithHistory retrieves deals with their update history
func GetDealsWithHistory(filter *DealFilter) ([]DealWithHistory, int, error) {
	db, err := GetDB()
	if err != nil {
		return nil, 0, err
	}

	// Step 1: Get all NEW and DELETE records (latest versions) with filters
	query := `SELECT NO, nextNO, prevNO, DealType, DealDate, DealName, DealPartner, 
	          DealPrice, DealRemark, RecUpdate, RegDate, RecStatus, FilePath, Hash 
	          FROM Deals WHERE (RecStatus = 'NEW' OR RecStatus = 'DELETE') AND nextNO IS NULL`
	countQuery := `SELECT COUNT(*) FROM Deals WHERE (RecStatus = 'NEW' OR RecStatus = 'DELETE') AND nextNO IS NULL`
	args := []interface{}{}

	if filter.FromDate != "" {
		query += " AND DealDate >= ?"
		countQuery += " AND DealDate >= ?"
		args = append(args, filter.FromDate)
	}

	if filter.ToDate != "" {
		query += " AND DealDate <= ?"
		countQuery += " AND DealDate <= ?"
		args = append(args, filter.ToDate)
	}

	if filter.Partner != "" {
		query += " AND DealPartner LIKE ?"
		countQuery += " AND DealPartner LIKE ?"
		args = append(args, "%"+filter.Partner+"%")
	}

	if filter.Type != "" {
		query += " AND DealType = ?"
		countQuery += " AND DealType = ?"
		args = append(args, filter.Type)
	}

	if filter.Keyword != "" {
		query += " AND (DealName LIKE ? OR DealRemark LIKE ? OR DealPartner LIKE ?)"
		countQuery += " AND (DealName LIKE ? OR DealRemark LIKE ? OR DealPartner LIKE ?)"
		args = append(args, "%"+filter.Keyword+"%", "%"+filter.Keyword+"%", "%"+filter.Keyword+"%")
	}

	// Get count of NEW records
	var totalCount int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	err = db.QueryRow(countQuery, countArgs...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count deals: %v", err)
	}

	// Order by RecUpdate DESC for parent records
	query += " ORDER BY RecUpdate DESC, NO DESC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	} else {
		query += " LIMIT 1000"
		args = append(args, 1000)
	}

	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query deals: %v", err)
	}
	defer rows.Close()

	dealsWithHistory := []DealWithHistory{}
	
	for rows.Next() {
		deal := Deal{}
		err := rows.Scan(
			&deal.NO, &deal.NextNO, &deal.PrevNO, &deal.DealType, &deal.DealDate,
			&deal.DealName, &deal.DealPartner, &deal.DealPrice, &deal.DealRemark,
			&deal.RecUpdate, &deal.RegDate, &deal.RecStatus, &deal.FilePath, &deal.Hash)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan deal: %v", err)
		}

		// Extract base NO (without branch suffix)
		baseNO := extractBaseNO(deal.NO)
		
		// Get history for this deal
		history, err := getDealHistory(baseNO, deal.NO)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get deal history: %v", err)
		}

		dealWithHistory := DealWithHistory{
			Deal:        deal,
			BaseNO:      baseNO,
			HasChildren: len(history) > 0,
			ChildCount:  len(history),
			Children:    history,
		}
		
		dealsWithHistory = append(dealsWithHistory, dealWithHistory)
	}

	return dealsWithHistory, totalCount, nil
}

// extractBaseNO extracts the base deal number without branch suffix
func extractBaseNO(dealNO string) string {
	if idx := strings.LastIndex(dealNO, "-"); idx != -1 {
		// Check if the part after "-" is a number (branch suffix)
		if _, err := strconv.Atoi(dealNO[idx+1:]); err == nil {
			return dealNO[:idx]
		}
	}
	return dealNO
}

// getDealHistory retrieves all historical versions of a deal
func getDealHistory(baseNO string, currentNO string) ([]Deal, error) {
	db, err := GetDB()
	if err != nil {
		return nil, err
	}

	// Query to get all versions of this deal except the current one
	// Using LIKE to match base number and any branch versions
	query := `SELECT NO, nextNO, prevNO, DealType, DealDate, DealName, DealPartner, 
	          DealPrice, DealRemark, RecUpdate, RegDate, RecStatus, FilePath, Hash 
	          FROM Deals 
	          WHERE (NO = ? OR NO LIKE ?) AND NO != ?
	          ORDER BY RecUpdate DESC`

	rows, err := db.Query(query, baseNO, baseNO+"-%", currentNO)
	if err != nil {
		return nil, fmt.Errorf("failed to query deal history: %v", err)
	}
	defer rows.Close()

	history := []Deal{}
	for rows.Next() {
		deal := Deal{}
		err := rows.Scan(
			&deal.NO, &deal.NextNO, &deal.PrevNO, &deal.DealType, &deal.DealDate,
			&deal.DealName, &deal.DealPartner, &deal.DealPrice, &deal.DealRemark,
			&deal.RecUpdate, &deal.RegDate, &deal.RecStatus, &deal.FilePath, &deal.Hash)
		if err != nil {
			return nil, fmt.Errorf("failed to scan historical deal: %v", err)
		}
		history = append(history, deal)
	}

	return history, nil
}

func DeleteDeal(dealID string) error {
	db, err := GetDB()
	if err != nil {
		return err
	}

	// Logical delete: Update RecStatus to 'DELETE' and update RecUpdate timestamp
	query := `UPDATE Deals SET RecStatus='DELETE', RecUpdate=? WHERE NO=? AND (RecStatus='NEW' OR RecStatus='UPDATE')`
	now := time.Now().Format("2006-01-02T15:04:05Z")
	
	result, err := db.Exec(query, now, dealID)
	if err != nil {
		return fmt.Errorf("failed to delete deal: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check affected rows: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("deal not found or already deleted: %s", dealID)
	}

	return nil
}