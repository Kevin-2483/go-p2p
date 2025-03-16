package db

import (
	"errors"
	"database/sql"
	"server/models"
	"time"
)

// SaveTurn 保存TURN服务器配置
func SaveTurn(turn *models.TurnServer) error {
	if turn == nil {
		return errors.New("turn server config is nil")
	}

	// 设置创建和更新时间
	now := time.Now()
	turn.CreatedAt = now
	turn.UpdatedAt = now

	// 保存到数据库
	_, err := db.Exec(`
		INSERT INTO turn_servers (id, owner_id, space_id, url, username, password, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, turn.ID, turn.OwnerID, turn.SpaceID, turn.URL, turn.Username, turn.Password, turn.CreatedAt, turn.UpdatedAt)
	return err
}

// GetTurnsByOwnerID 获取指定用户的所有TURN服务器配置
func GetTurnsByOwnerID(ownerID string) ([]models.TurnServer, error) {
	rows, err := db.Query(`
		SELECT id, owner_id, space_id, url, username, password, created_at, updated_at
		FROM turn_servers
		WHERE owner_id = ?
	`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var turns []models.TurnServer
	for rows.Next() {
		var turn models.TurnServer
		if err := rows.Scan(&turn.ID, &turn.OwnerID, &turn.SpaceID, &turn.URL, &turn.Username, &turn.Password, &turn.CreatedAt, &turn.UpdatedAt); err != nil {
			return nil, err
		}
		turns = append(turns, turn)
	}
	return turns, nil
}

// UpdateTurn 更新TURN服务器配置
func UpdateTurn(turn *models.TurnServer) error {
	if turn == nil {
		return errors.New("turn server config is nil")
	}

	// 检查TURN服务器是否存在
	existingTurn, err := GetTurnByID(turn.ID)
	if err != nil {
		return err
	}
	if existingTurn == nil {
		return errors.New("TURN服务器不存在")
	}

	// 确保只能更新自己的TURN服务器
	if existingTurn.OwnerID != turn.OwnerID {
		return errors.New("无权更新此TURN服务器")
	}

	// 更新时间
	turn.UpdatedAt = time.Now()

	// 更新数据库中的记录
	_, err = db.Exec(`
		UPDATE turn_servers
		SET url = ?, username = ?, password = ?, updated_at = ?
		WHERE id = ? AND owner_id = ?
	`, turn.URL, turn.Username, turn.Password, turn.UpdatedAt, turn.ID, turn.OwnerID)
	return err
}

// DeleteTurn 删除TURN服务器配置
func DeleteTurn(turnID string, ownerID string) error {
	// 检查TURN服务器是否存在
	turn, err := GetTurnByID(turnID)
	if err != nil {
		return err
	}
	if turn == nil {
		return errors.New("TURN服务器不存在")
	}

	// 确保只能删除自己的TURN服务器
	if turn.OwnerID != ownerID {
		return errors.New("无权删除此TURN服务器")
	}

	// 从数据库中删除
	_, err = db.Exec(`
		DELETE FROM turn_servers
		WHERE id = ? AND owner_id = ?
	`, turnID, ownerID)
	return err
}

// GetTurnByID 根据ID获取TURN服务器配置
func GetTurnByID(id string) (*models.TurnServer, error) {
	var turn models.TurnServer
	err := db.QueryRow(`
		SELECT id, owner_id, space_id, url, username, password, created_at, updated_at
		FROM turn_servers
		WHERE id = ?
	`, id).Scan(&turn.ID, &turn.OwnerID, &turn.SpaceID, &turn.URL, &turn.Username, &turn.Password, &turn.CreatedAt, &turn.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &turn, nil
}

// GetTurnsBySpaceID 获取指定空间的所有TURN服务器配置
func GetTurnsBySpaceID(spaceID string) ([]models.TurnServer, error) {
	rows, err := db.Query(`
		SELECT id, owner_id, space_id, url, username, password, created_at, updated_at
		FROM turn_servers
		WHERE space_id = ?
	`, spaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var turns []models.TurnServer
	for rows.Next() {
		var turn models.TurnServer
		if err := rows.Scan(&turn.ID, &turn.OwnerID, &turn.SpaceID, &turn.URL, &turn.Username, &turn.Password, &turn.CreatedAt, &turn.UpdatedAt); err != nil {
			return nil, err
		}
		turns = append(turns, turn)
	}
	return turns, nil
}