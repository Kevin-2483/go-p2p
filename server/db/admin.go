package db

import (
	"database/sql"
	"fmt"
	"server/models"
	"time"

	"github.com/charmbracelet/log"
	"golang.org/x/crypto/bcrypt"
)

// SaveAdmin 保存管理员信息到数据库
func SaveAdmin(admin *models.Admin) error {
	// 检查用户名是否已存在
	existingAdmin, err := GetAdminByUsername(admin.Username)
	if err != nil {
		return err
	}
	if existingAdmin != nil {
		return fmt.Errorf("用户名已存在")
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(admin.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// 保存管理员信息
	_, err = db.Exec(
		"INSERT INTO admins (id, username, password, created_at) VALUES (?, ?, ?, ?)",
		admin.ID,
		admin.Username,
		string(hashedPassword),
		time.Now(),
	)

	return err
}

// GetAdminList 获取管理员列表
func GetAdminList() ([]*models.Admin, error) {
	rows, err := db.Query("SELECT id, username, created_at FROM admins")
	if err != nil {
		log.Error("查询管理员列表失败", "error", err)
		return nil, err
	}
	defer rows.Close()

	var admins []*models.Admin
	for rows.Next() {
		admin := &models.Admin{}
		err := rows.Scan(&admin.ID, &admin.Username, &admin.CreatedAt)
		if err != nil {
			log.Error("扫描管理员数据失败", "error", err)
			return nil, err
		}
		admins = append(admins, admin)
	}

	return admins, nil
}

// GetAdminByID 通过ID获取管理员信息
func GetAdminByID(id string) (*models.Admin, error) {
	admin := &models.Admin{}
	err := db.QueryRow(
		"SELECT id, username, created_at FROM admins WHERE id = ?",
		id,
	).Scan(&admin.ID, &admin.Username, &admin.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		log.Error("查询管理员失败", "error", err)
		return nil, err
	}

	return admin, nil
}

// UpdateAdmin 更新管理员信息
func UpdateAdmin(admin *models.Admin) error {
	// 检查管理员是否存在
	existingAdmin, err := GetAdminByID(admin.ID)
	if err != nil {
		return err
	}
	if existingAdmin == nil {
		return fmt.Errorf("管理员不存在")
	}

	// 如果要更新用户名，检查新用户名是否已存在
	if admin.Username != existingAdmin.Username {
		tempAdmin, err := GetAdminByUsername(admin.Username)
		if err != nil {
			return err
		}
		if tempAdmin != nil {
			return fmt.Errorf("用户名已存在")
		}
	}

	// 如果提供了新密码，则加密新密码
	if admin.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(admin.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		// 更新用户名和密码
		_, err = db.Exec(
			"UPDATE admins SET username = ?, password = ? WHERE id = ?",
			admin.Username,
			string(hashedPassword),
			admin.ID,
		)
	} else {
		// 只更新用户名
		_, err = db.Exec(
			"UPDATE admins SET username = ? WHERE id = ?",
			admin.Username,
			admin.ID,
		)
	}

	return err
}

// DeleteAdmin 删除管理员
func DeleteAdmin(id string) error {
	// 检查是否是最后一个管理员
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM admins").Scan(&count)
	if err != nil {
		return err
	}
	if count <= 1 {
		return fmt.Errorf("不能删除最后一个管理员")
	}

	// 删除管理员
	result, err := db.Exec("DELETE FROM admins WHERE id = ?", id)
	if err != nil {
		return err
	}

	// 检查是否找到并删除了管理员
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("管理员不存在")
	}

	return nil
}