package db

import (
	"database/sql"
	"fmt"
	"server/models"
	"time"

	"github.com/charmbracelet/log"
	"golang.org/x/crypto/bcrypt"
)

// GetUserList 获取用户列表
func GetUserList() ([]*models.User, error) {
	rows, err := db.Query("SELECT id, username, email, created_at, updated_at FROM users")
	if err != nil {
		log.Error("查询用户列表失败", "error", err)
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.CreatedAt, &user.UpdatedAt)
		if err != nil {
			log.Error("扫描用户数据失败", "error", err)
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

// GetUserByID 通过ID获取用户信息
func GetUserByID(id string) (*models.User, error) {
	user := &models.User{}
	err := db.QueryRow(
		"SELECT id, username, email, created_at, updated_at FROM users WHERE id = ?",
		id,
	).Scan(&user.ID, &user.Username, &user.Email, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		log.Error("查询用户失败", "error", err)
		return nil, err
	}

	return user, nil
}

// UpdateUser 更新用户信息
func UpdateUser(user *models.User) error {
	// 检查用户是否存在
	existingUser, err := GetUserByID(user.ID)
	if err != nil {
		return err
	}
	if existingUser == nil {
		return fmt.Errorf("用户不存在")
	}

	// 如果要更新用户名，检查新用户名是否已存在
	if user.Username != existingUser.Username {
		tempUser, err := GetUserByUsername(user.Username)
		if err != nil {
			return err
		}
		if tempUser != nil {
			return fmt.Errorf("用户名已存在")
		}
	}

	// 如果要更新邮箱，检查新邮箱是否已存在
	if user.Email != existingUser.Email {
		tempUser, err := GetUserByEmail(user.Email)
		if err != nil {
			return err
		}
		if tempUser != nil {
			return fmt.Errorf("邮箱已被注册")
		}
	}

	// 如果提供了新密码，则加密新密码
	if user.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		// 更新所有字段
		_, err = db.Exec(
			"UPDATE users SET username = ?, password = ?, email = ?, updated_at = ? WHERE id = ?",
			user.Username,
			string(hashedPassword),
			user.Email,
			time.Now(),
			user.ID,
		)
	} else {
		// 更新除密码外的其他字段
		_, err = db.Exec(
			"UPDATE users SET username = ?, email = ?, updated_at = ? WHERE id = ?",
			user.Username,
			user.Email,
			time.Now(),
			user.ID,
		)
	}

	return err
}

// DeleteUser 删除用户
func DeleteUser(id string) error {
	// 删除用户
	result, err := db.Exec("DELETE FROM users WHERE id = ?", id)
	if err != nil {
		return err
	}

	// 检查是否找到并删除了用户
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("用户不存在")
	}

	return nil
}