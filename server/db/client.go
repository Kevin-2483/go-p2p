package db

import (
	"database/sql"
	"fmt"
	"server/models"
)

// SaveClient 保存客户端信息到数据库
func SaveClient(client *models.Client) error {
	_, err := db.Exec(`
		INSERT INTO clients (id, owner_id, space_id, public_key, name, description)
		VALUES (?, ?, ?, ?, ?, ?)
	`, client.ID, client.OwnerID, client.SpaceID, client.PublicKey, client.Name, client.Description)
	return err
}

// GetClientByID 根据ID获取客户端信息
func GetClientByID(id string) (*models.Client, error) {
	var client models.Client
	err := db.QueryRow(`
		SELECT id, owner_id, space_id, public_key, name, description
		FROM clients
		WHERE id = ?
	`, id).Scan(&client.ID, &client.OwnerID, &client.SpaceID, &client.PublicKey, &client.Name, &client.Description)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &client, err
}

// GetClientsByOwnerID 获取用户的所有客户端
func GetClientsByOwnerID(ownerID string) ([]*models.Client, error) {
	rows, err := db.Query(`
		SELECT id, owner_id, space_id, public_key, name, description
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
		if err := rows.Scan(&client.ID, &client.OwnerID, &client.SpaceID, &client.PublicKey, &client.Name, &client.Description); err != nil {
			return nil, err
		}
		clients = append(clients, &client)
	}
	return clients, nil
}

// UpdateClient 更新客户端信息
func UpdateClient(client *models.Client) error {
	// 检查客户端是否存在
	existingClient, err := GetClientByID(client.ID)
	if err != nil {
		return err
	}
	if existingClient == nil {
		return fmt.Errorf("客户端不存在")
	}

	// 确保只能更新自己的客户端
	if existingClient.OwnerID != client.OwnerID {
		return fmt.Errorf("无权更新此客户端")
	}

	// 更新数据库
	_, err = db.Exec(`
		UPDATE clients
		SET space_id = ?, public_key = ?, name =?, description =?
		WHERE id = ? AND owner_id = ?
	`, client.SpaceID, client.PublicKey, client.Name, client.Description, client.ID, client.OwnerID)
	return err
}



// DeleteClient 删除客户端
func DeleteClient(id string, ownerID string) error {
	// 检查客户端是否存在
	client, err := GetClientByID(id)
	if err != nil {
		return err
	}
	if client == nil {
		return fmt.Errorf("客户端不存在")
	}

	// 确保只能删除自己的客户端
	if client.OwnerID != ownerID {
		return fmt.Errorf("无权删除此客户端")
	}

	// 从数据库中删除
	_, err = db.Exec(`
		DELETE FROM clients
		WHERE id = ? AND owner_id = ?
	`, id, ownerID)
	return err
}

// GetClientsBySpaceID 获取同一空间内的所有客户端
func GetClientsBySpaceID(spaceID string) ([]*models.Client, error) {
	rows, err := db.Query(`
		SELECT id, owner_id, space_id, public_key, name, description
		FROM clients
		WHERE space_id = ?
	`, spaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clients []*models.Client
	for rows.Next() {
		var client models.Client
		if err := rows.Scan(&client.ID, &client.OwnerID, &client.SpaceID, &client.PublicKey, &client.Name, &client.Description); err != nil {
			return nil, err
		}
		clients = append(clients, &client)
	}
	return clients, nil
}