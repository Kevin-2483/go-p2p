package db

import (
	"database/sql"
	"fmt"
	"server/models"
)

// SaveClient 保存客户端信息到数据库
func SaveClient(client *models.Client) error {
	_, err := db.Exec(`
		INSERT INTO clients (id, owner_id, space_id, public_key)
		VALUES (?, ?, ?, ?)
	`, client.ID, client.OwnerID, client.SpaceID, client.PublicKey)
	return err
}

// GetClientByID 根据ID获取客户端信息
func GetClientByID(id string) (*models.Client, error) {
	var client models.Client
	err := db.QueryRow(`
		SELECT id, owner_id, space_id, public_key
		FROM clients
		WHERE id = ?
	`, id).Scan(&client.ID, &client.OwnerID, &client.SpaceID, &client.PublicKey)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &client, err
}

// GetClientsByOwnerID 获取用户的所有客户端
func GetClientsByOwnerID(ownerID string) ([]*models.Client, error) {
	rows, err := db.Query(`
		SELECT id, owner_id, space_id, public_key
		FROM clients
		WHERE owner_id = ?
	`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clients []*models.Client
	for rows.Next() {
		var client models.Client
		if err := rows.Scan(&client.ID, &client.OwnerID, &client.SpaceID, &client.PublicKey); err != nil {
			return nil, err
		}
		clients = append(clients, &client)
	}
	return clients, nil
}

// UpdateClient 更新客户端信息
func UpdateClient(client *models.Client) error {
	result, err := db.Exec(`
		UPDATE clients
		SET space_id = ?, public_key = ?
		WHERE id = ? AND owner_id = ?
	`, client.SpaceID, client.PublicKey, client.ID, client.OwnerID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("client not found or not owned by user")
	}
	return nil
}

// DeleteClient 删除客户端
func DeleteClient(id string, ownerID string) error {
	result, err := db.Exec(`
		DELETE FROM clients
		WHERE id = ? AND owner_id = ?
	`, id, ownerID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("client not found or not owned by user")
	}
	return nil
}