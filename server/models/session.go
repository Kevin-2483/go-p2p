package models

import (
	"time"
	"github.com/google/uuid"
)

// Session 表示用户会话
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewSession 创建新的会话
func NewSession(userID string) *Session {
	now := time.Now()
	return &Session{
		ID:        uuid.New().String(),
		UserID:    userID,
		Token:     uuid.New().String(),
		ExpiresAt: now.Add(24 * time.Hour), // 会话有效期24小时
		CreatedAt: now,
		UpdatedAt: now,
	}
}