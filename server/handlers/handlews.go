package handlers

import (
	"server/models"
	"time"

	"github.com/charmbracelet/log"
)

// handlePing 处理来自客户端的ping消息
func handlePing(client *models.Client, msg *models.Message) {
	// 解析ping消息数据
	if data, ok := msg.Data.(map[string]interface{}); ok {
		// 获取时间戳
		if timestamp, ok := data["timestamp"].(float64); ok {
			// 使用客户端发送的时间戳计算延迟
			delay := time.Now().UnixMilli() - int64(timestamp)
			// 更新客户端最后ping时间和延迟
			UpdateClientPing(client.ID, delay)
		}

		// 获取WebRTC状态
		if webrtcStatus, ok := data["webrtc_status"].(map[string]interface{}); ok {
			// 转换状态为map[string]string
			status := make(map[string]string)
			for id, state := range webrtcStatus {
				if stateStr, ok := state.(string); ok {
					status[id] = stateStr
				}
			}
			// 更新客户端的WebRTC状态
			client.WebRTCStatus = status
		}
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
