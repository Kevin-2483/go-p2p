package handlers

import (
	"time"
	"server/models"

	"github.com/charmbracelet/log"
)

// handlePing 处理来自客户端的ping消息
func handlePing(client *models.Client, msg *models.Message) {
	// 计算网络延迟
	if timestamp, ok := msg.Data.(float64); ok {
		// 使用客户端发送的时间戳计算延迟
		delay := time.Now().UnixMilli() - int64(timestamp)
		// 更新客户端最后ping时间和延迟
		UpdateClientPing(client.ID, delay)
	}

	// 回复pong消息
	response := models.Message{
		Type: "pong",
		Data: time.Now().UnixMilli(),
	}
	if err := client.Conn.WriteJSON(response); err != nil {
		log.Error("发送pong消息失败", "error", err)
	}
}
