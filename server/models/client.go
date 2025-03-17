package models

import (
	"github.com/gorilla/websocket"
	"time"
)

// Client 表示一个连接的客户端
type Client struct {
	ID            string          `json:"id"`
	OwnerID       string          `json:"owner_id"`
	SpaceID       string          `json:"space_id"`
	PublicKey     string          `json:"public_key"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	Conn          *websocket.Conn `json:"-"`
	ConnectedAt   time.Time      `json:"connected_at"`
	LastPingTime  time.Time      `json:"last_ping_time"`
	LastPingDelay int64         `json:"last_ping_delay"` // 毫秒
}