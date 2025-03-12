package db

import (
	"database/sql"
	"fmt"
	"server/models"
	"time"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

var db *sql.DB

// Init 初始化SQLite数据库连接并执行自动迁移
func Init() error {
	var err error
	db, err = sql.Open("sqlite3", "./data.db")
	if err != nil {
		return err
	}

	// 创建clients表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS clients (
			id TEXT PRIMARY KEY,
			owner_id TEXT NOT NULL,
			space_id TEXT NOT NULL,
			public_key TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(owner_id) REFERENCES users(id),
			FOREIGN KEY(space_id) REFERENCES spaces(id)
		)
	`)
	if err != nil {
		log.Error("创建clients表失败", "error", err)
		return err
	}

	// 创建admins表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS admins (
			id TEXT PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			password TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		log.Error("创建admins表失败", "error", err)
		return err
	}

	// 创建users表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			username TEXT NOT NULL,
			password TEXT NOT NULL,
			email TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(username),
			UNIQUE(email)
		)
	`)
	if err != nil {
		log.Error("创建users表失败", "error", err)
		return err
	}

	// 创建spaces表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS spaces (
			id TEXT PRIMARY KEY,
			owner_id TEXT NOT NULL,
			name TEXT NOT NULL,
			description TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(owner_id) REFERENCES users(id)
		)
	`)
	if err != nil {
		log.Error("创建spaces表失败", "error", err)
		return err
	}

	// 创建turn_servers表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS turn_servers (
			id TEXT PRIMARY KEY,
			owner_id TEXT NOT NULL,
			space_id TEXT NOT NULL,
			url TEXT NOT NULL,
			username TEXT NOT NULL,
			password TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(owner_id) REFERENCES users(id),
			FOREIGN KEY(space_id) REFERENCES spaces(id)
		)
	`)
	if err != nil {
		log.Error("创建turn_servers表失败", "error", err)
		return err
	}

	// 创建sessions表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			token TEXT NOT NULL UNIQUE,
			expires_at DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(user_id) REFERENCES users(id)
		)
	`)
	if err != nil {
		log.Error("创建sessions表失败", "error", err)
		return err
	}

	// 检查是否需要创建初始管理员账户
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM admins").Scan(&count)
	if err != nil {
		log.Error("检查管理员账户失败", "error", err)
		return err
	}

	// 如果没有管理员账户，创建初始管理员
	if count == 0 {
		adminID := uuid.New().String()
		// 对初始密码进行加密
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		if err != nil {
			log.Error("加密管理员密码失败", "error", err)
			return err
		}

		_, err = db.Exec(
			"INSERT INTO admins (id, username, password, created_at) VALUES (?, ?, ?, ?)",
			adminID,
			"admin",
			string(hashedPassword),
			time.Now(),
		)
		if err != nil {
			log.Error("创建初始管理员账户失败", "error", err)
			return err
		}
		log.Info("初始管理员账户创建成功", "username", "admin", "password", "admin123")
	}

	log.Info("数据库初始化完成")
	return nil
}

// GetDB 返回数据库连接实例
func GetDB() *sql.DB {
	return db
}

// GetUserByUsername 通过用户名查询用户信息
func GetUserByUsername(username string) (*models.User, error) {
	user := &models.User{}
	err := db.QueryRow(
		"SELECT id, username, password, email, created_at, updated_at FROM users WHERE username = ?",
		username,
	).Scan(&user.ID, &user.Username, &user.Password, &user.Email, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		log.Error("查询用户失败", "error", err)
		return nil, err
	}

	return user, nil
}

// GetUserByEmail 通过邮箱查询用户信息
func GetUserByEmail(email string) (*models.User, error) {
	user := &models.User{}
	err := db.QueryRow(
		"SELECT id, username, password, email, created_at, updated_at FROM users WHERE email = ?",
		email,
	).Scan(&user.ID, &user.Username, &user.Password, &user.Email, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		log.Error("查询用户失败", "error", err)
		return nil, err
	}

	return user, nil
}

// SaveUser 保存用户信息到数据库
func SaveUser(user *models.User) error {
	// 检查用户名是否已存在
	existingUser, err := GetUserByUsername(user.Username)
	if err != nil {
		return err
	}
	if existingUser != nil {
		return fmt.Errorf("用户名已存在")
	}

	// 检查邮箱是否已存在
	existingUser, err = GetUserByEmail(user.Email)
	if err != nil {
		return err
	}
	if existingUser != nil {
		return fmt.Errorf("邮箱已被注册")
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// 保存用户信息
	_, err = db.Exec(
		"INSERT INTO users (id, username, password, email, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		user.ID,
		user.Username,
		string(hashedPassword),
		user.Email,
		time.Now(),
		time.Now(),
	)

	return err
}

// GetAdminByUsername 通过用户名查询管理员信息
func GetAdminByUsername(username string) (*models.Admin, error) {
	admin := &models.Admin{}
	err := db.QueryRow(
		"SELECT id, username, password, created_at FROM admins WHERE username = ?",
		username,
	).Scan(&admin.ID, &admin.Username, &admin.Password, &admin.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		log.Error("查询管理员失败", "error", err)
		return nil, err
	}

	return admin, nil
}