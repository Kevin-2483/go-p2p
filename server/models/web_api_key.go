package models

import (
	"time"

	"github.com/google/uuid"
)

// WebAPIKey 表示客户端初始化使用的一次性密钥
type WebAPIKey struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Key         string    `json:"key"`
	SpaceID     string    `json:"space_id"`
	Name        string    `json:"name"`        // 客户端名称
	Description string    `json:"description"` // 客户端描述
	Used        bool      `json:"used"`
	ExpiresAt   time.Time `json:"expires_at"`
	CreatedAt   time.Time `json:"created_at"`

	// 服务器连接信息
	Server struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	} `json:"server"`

	// WebSocket配置
	WebSocket struct {
		Path           string `json:"path"`
		PingInterval   int    `json:"ping_interval"`
		ReconnectDelay int    `json:"reconnect_delay"`
	} `json:"websocket"`
}

// NewWebAPIKey 创建新的WebAPIKey
func NewWebAPIKey(userID string, name string, description string, spaceID string) *WebAPIKey {
	now := time.Now()
	return &WebAPIKey{
		ID:          uuid.New().String(),
		UserID:      userID,
		Key:         uuid.New().String(),
		SpaceID:     spaceID,
		Name:        name,
		Description: description,
		Used:        false,
		ExpiresAt:   now.Add(24 * time.Hour), // 密钥有效期24小时
		CreatedAt:   now,
	}
}
