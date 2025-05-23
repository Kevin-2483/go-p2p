package models

import "time"

// TurnServer 表示TURN服务器配置
type TurnServer struct {
	ID        string    `json:"id"`
	OwnerID   string    `json:"owner_id"`
	SpaceID   string    `json:"space_id"`
	URL       string    `json:"url"`
	Username  string    `json:"username"`
	Password  string    `json:"password"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}