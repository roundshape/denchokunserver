package models

import (
	"database/sql"
	"fmt"
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
	Limit     int    `form:"limit"`
	Offset    int    `form:"offset"`
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

	if deal.DealPartner != "" {
		_, err = tx.Exec("INSERT OR IGNORE INTO DealPartners (name) VALUES (?)", deal.DealPartner)
		if err != nil {
			return fmt.Errorf("failed to update partner master: %v", err)
		}
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

	query := "SELECT NO, nextNO, prevNO, DealType, DealDate, DealName, DealPartner, DealPrice, DealRemark, RecUpdate, RegDate, RecStatus, FilePath, Hash FROM Deals WHERE 1=1"
	countQuery := "SELECT COUNT(*) FROM Deals WHERE 1=1"
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
		query += " AND (DealName LIKE ? OR DealRemark LIKE ?)"
		countQuery += " AND (DealName LIKE ? OR DealRemark LIKE ?)"
		args = append(args, "%"+filter.Keyword+"%", "%"+filter.Keyword+"%")
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

	if deal.DealPartner != "" {
		_, err = tx.Exec("INSERT OR IGNORE INTO DealPartners (name) VALUES (?)", deal.DealPartner)
		if err != nil {
			return fmt.Errorf("failed to update partner master: %v", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	return nil
}

func DeleteDeal(dealID string) error {
	db, err := GetDB()
	if err != nil {
		return err
	}

	result, err := db.Exec("DELETE FROM Deals WHERE NO = ?", dealID)
	if err != nil {
		return fmt.Errorf("failed to delete deal: %v", err)
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