package models

import (
	"fmt"
)

type DealPartner struct {
	Name string `json:"name" binding:"required"`
}

func GetDealPartners() ([]string, error) {
	db, err := GetDB()
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
	db, err := GetDB()
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

	_, err = tx.Exec("UPDATE Deals SET DealPartner = ? WHERE DealPartner = ?", newName, oldName)
	if err != nil {
		return fmt.Errorf("failed to update deals: %v", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	return nil
}

func DeleteDealPartner(name string) error {
	db, err := GetDB()
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %v", err)
	}
	defer tx.Rollback()

	var dealCount int
	err = tx.QueryRow("SELECT COUNT(*) FROM Deals WHERE DealPartner = ?", name).Scan(&dealCount)
	if err != nil {
		return fmt.Errorf("failed to check deals: %v", err)
	}

	if dealCount > 0 {
		return fmt.Errorf("cannot delete partner with existing deals: %s (has %d deals)", name, dealCount)
	}

	result, err := tx.Exec("DELETE FROM DealPartners WHERE name = ?", name)
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

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	return nil
}