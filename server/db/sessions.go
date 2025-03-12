package db

import (
	"database/sql"
	"time"
	
	"github.com/charmbracelet/log"
	"github.com/google/uuid"
)

// Session 表示一个用户会话
type Session struct {
	ID        string
	UserID    string
	Token     string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// CreateSession 创建新的会话记录
func CreateSession(userID string) (*Session, error) {
	session := &Session{
		ID:        uuid.New().String(),
		UserID:    userID,
		Token:     uuid.New().String(),
		ExpiresAt: time.Now().Add(24 * time.Hour), // 会话有效期24小时
		CreatedAt: time.Now(),
	}

	_, err := db.Exec(
		"INSERT INTO sessions (id, user_id, token, expires_at, created_at) VALUES (?, ?, ?, ?, ?)",
		session.ID,
		session.UserID,
		session.Token,
		session.ExpiresAt,
		session.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return session, nil
}

// GetSessionByToken 通过令牌获取会话信息
func GetSessionByToken(token string) (*Session, error) {
	session := &Session{}
	err := db.QueryRow(
		"SELECT id, user_id, token, expires_at, created_at FROM sessions WHERE token = ?",
		token,
	).Scan(&session.ID, &session.UserID, &session.Token, &session.ExpiresAt, &session.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return session, nil
}

// DeleteSession 删除会话记录
func DeleteSession(token string) error {
	_, err := db.Exec("DELETE FROM sessions WHERE token = ?", token)
	return err
}

// CleanExpiredSessions 清理过期的会话记录
func CleanExpiredSessions() error {
	// 获取当前时间
	now := time.Now()

	// 删除过期的会话记录
	result, err := db.Exec("DELETE FROM sessions WHERE expires_at < ?", now)
	if err != nil {
		log.Error("清理过期会话失败", "error", err)
		return err
	}

	// 获取删除的记录数
	affected, _ := result.RowsAffected()
	log.Info("清理过期会话完成", "deleted_count", affected)

	return nil
}

// StartSessionCleaner 启动定时清理过期会话的任务
func StartSessionCleaner() {
	// 创建定时器，每天凌晨3点执行清理
	ticker := time.NewTicker(24 * time.Hour)
	go func() {
		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day()+1, 3, 0, 0, 0, now.Location())
			initialDelay := next.Sub(now)

			// 等待到第一次执行时间
			time.Sleep(initialDelay)

			// 执行清理
			if err := CleanExpiredSessions(); err != nil {
				log.Error("执行定时清理任务失败", "error", err)
			}

			// 等待下一次执行
			<-ticker.C
		}
	}()
}