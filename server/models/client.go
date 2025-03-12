package models

import (
	"github.com/gorilla/websocket"
)

// Client 表示一个连接的客户端
type Client struct {
	ID        string          `json:"id"`
	OwnerID   string          `json:"owner_id"`
	SpaceID   string          `json:"space_id"`
	PublicKey string          `json:"public_key"`
	Conn      *websocket.Conn `json:"-"`
}