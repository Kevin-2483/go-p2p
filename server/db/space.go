package db

import (
	"database/sql"
	"errors"
	"server/models"
	"time"

	"github.com/charmbracelet/log"
)

// SaveSpace 保存空间信息到数据库
func SaveSpace(space *models.Space) error {
	// 设置创建和更新时间
	now := time.Now()
	space.CreatedAt = now
	space.UpdatedAt = now

	// 保存到数据库
	_, err := db.Exec(
		"INSERT INTO spaces (id, owner_id, name, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		space.ID,
		space.OwnerID,
		space.Name,
		space.Description,
		space.CreatedAt,
		space.UpdatedAt,
	)
	if err != nil {
		log.Error("保存空间失败", "error", err)
		return err
	}

	return nil
}

// GetSpaceByID 根据ID获取空间信息
func GetSpaceByID(id string) (*models.Space, error) {
	space := &models.Space{}
	err := db.QueryRow(
		"SELECT id, owner_id, name, description, created_at, updated_at FROM spaces WHERE id = ?",
		id,
	).Scan(&space.ID, &space.OwnerID, &space.Name, &space.Description, &space.CreatedAt, &space.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		log.Error("获取空间信息失败", "error", err)
		return nil, err
	}

	return space, nil
}

// GetSpacesByOwnerID 获取用户的所有空间
func GetSpacesByOwnerID(ownerID string) ([]*models.Space, error) {
	rows, err := db.Query(
		"SELECT id, owner_id, name, description, created_at, updated_at FROM spaces WHERE owner_id = ?",
		ownerID,
	)
	if err != nil {
		log.Error("获取用户空间列表失败", "error", err)
		return nil, err
	}
	defer rows.Close()

	var spaces []*models.Space
	for rows.Next() {
		space := &models.Space{}
		err := rows.Scan(&space.ID, &space.OwnerID, &space.Name, &space.Description, &space.CreatedAt, &space.UpdatedAt)
		if err != nil {
			log.Error("扫描空间数据失败", "error", err)
			return nil, err
		}
		spaces = append(spaces, space)
	}

	return spaces, nil
}

// UpdateSpace 更新空间信息
func UpdateSpace(space *models.Space) error {
	// 检查空间是否存在
	existingSpace, err := GetSpaceByID(space.ID)
	if err != nil {
		return err
	}
	if existingSpace == nil {
		return errors.New("空间不存在")
	}

	// 确保只能更新自己的空间
	if existingSpace.OwnerID != space.OwnerID {
		return errors.New("无权更新此空间")
	}

	// 更新时间
	space.UpdatedAt = time.Now()

	// 更新数据库
	_, err = db.Exec(
		"UPDATE spaces SET name = ?, description = ?, updated_at = ? WHERE id = ? AND owner_id = ?",
		space.Name,
		space.Description,
		space.UpdatedAt,
		space.ID,
		space.OwnerID,
	)
	if err != nil {
		log.Error("更新空间信息失败", "error", err)
		return err
	}

	return nil
}

// DeleteSpace 删除空间
func DeleteSpace(id string, ownerID string) error {
	// 检查空间是否存在
	space, err := GetSpaceByID(id)
	if err != nil {
		return err
	}
	if space == nil {
		return errors.New("空间不存在")
	}

	// 确保只能删除自己的空间
	if space.OwnerID != ownerID {
		return errors.New("无权删除此空间")
	}

	// 从数据库中删除
	result, err := db.Exec("DELETE FROM spaces WHERE id = ? AND owner_id = ?", id, ownerID)
	if err != nil {
		log.Error("删除空间失败", "error", err)
		return err
	}

	// 检查是否找到并删除了空间
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errors.New("空间不存在或无权删除")
	}

	return nil
}