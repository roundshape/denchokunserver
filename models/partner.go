package models

import (
	"database/sql"
	"fmt"
)

type DealPartner struct {
	Name string `json:"name" binding:"required"`
}

func GetDealPartners() ([]string, error) {
	db, err := GetSystemDB()
	if err != nil {
		return nil, err
	}

	query := "SELECT name FROM DealPartners ORDER BY name"
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query partners: %v", err)
	}
	defer rows.Close()

	partners := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to scan partner: %v", err)
		}
		partners = append(partners, name)
	}

	return partners, nil
}

func CreateDealPartner(partner *DealPartner) error {
	db, err := GetSystemDB()
	if err != nil {
		return err
	}

	_, err = db.Exec("INSERT INTO DealPartners (name) VALUES (?)", partner.Name)
	if err != nil {
		if err.Error() == "UNIQUE constraint failed: DealPartners.name" {
			return fmt.Errorf("partner already exists: %s", partner.Name)
		}
		return fmt.Errorf("failed to create partner: %v", err)
	}

	return nil
}

func UpdateDealPartner(oldName string, newName string) error {
	systemDB, err := GetSystemDB()
	if err != nil {
		return err
	}

	// Start transaction on System.db
	tx, err := systemDB.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %v", err)
	}
	defer tx.Rollback()

	var exists int
	err = tx.QueryRow("SELECT COUNT(*) FROM DealPartners WHERE name = ?", oldName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check existence: %v", err)
	}

	if exists == 0 {
		return fmt.Errorf("partner not found: %s", oldName)
	}

	_, err = tx.Exec("UPDATE DealPartners SET name = ? WHERE name = ?", newName, oldName)
	if err != nil {
		return fmt.Errorf("failed to update partner: %v", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	// Update all period databases that contain deals with this partner
	if err := updateDealPartnerInAllPeriods(oldName, newName); err != nil {
		return fmt.Errorf("failed to update deals in period databases: %v", err)
	}

	return nil
}

func updateDealPartnerInAllPeriods(oldName, newName string) error {
	periods, err := GetAvailablePeriods()
	if err != nil {
		return err
	}

	for _, period := range periods {
		periodDB, err := ConnectPeriodDB(period)
		if err != nil {
			continue
		}

		// Update deals in this period database
		_, err = periodDB.Exec("UPDATE Deals SET DealPartner = ? WHERE DealPartner = ?", newName, oldName)
		if err != nil {
			return fmt.Errorf("failed to update deals in period %s: %v", period, err)
		}
	}

	return nil
}

func DeleteDealPartner(name string) error {
	// First check if partner is used in any period database
	if err := checkPartnerUsageInAllPeriods(name); err != nil {
		return err
	}

	systemDB, err := GetSystemDB()
	if err != nil {
		return err
	}

	result, err := systemDB.Exec("DELETE FROM DealPartners WHERE name = ?", name)
	if err != nil {
		return fmt.Errorf("failed to delete partner: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check affected rows: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("partner not found: %s", name)
	}

	return nil
}

func checkPartnerUsageInAllPeriods(partnerName string) error {
	periods, err := GetAvailablePeriods()
	if err != nil {
		return err
	}

	for _, period := range periods {
		periodDB, err := ConnectPeriodDB(period)
		if err != nil {
			continue
		}

		var dealCount int
		err = periodDB.QueryRow("SELECT COUNT(*) FROM Deals WHERE DealPartner = ?", partnerName).Scan(&dealCount)
		if err != nil && err != sql.ErrNoRows {
			return fmt.Errorf("failed to check deals in period %s: %v", period, err)
		}

		if dealCount > 0 {
			return fmt.Errorf("cannot delete partner with existing deals: %s (has %d deals in period %s)", partnerName, dealCount, period)
		}
	}

	return nil
}