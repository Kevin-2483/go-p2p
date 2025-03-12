package handlers

import (
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/gorilla/websocket"
	"server/models"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 在生产环境中应该限制来源
	},
}

// HandleWebSocket 处理WebSocket连接
func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("WebSocket连接升级失败", "error", err)
		return
	}
	defer conn.Close()

	// 创建新的客户端连接
	client := &models.Client{
		Conn: conn,
	}

	for {
		// 读取消息
		var msg models.Message
		if err := conn.ReadJSON(&msg); err != nil {
			log.Error("消息读取失败", "error", err)
			break
		}

		// 处理消息
		handleMessage(client, &msg)
	}
}

// handleMessage 处理接收到的WebSocket消息
func handleMessage(client *models.Client, msg *models.Message) {
	// TODO: 根据消息类型处理不同的信令逻辑
	switch msg.Type {
	case "offer":
		// 处理offer
	case "answer":
		// 处理answer
	case "candidate":
		// 处理ICE candidate
	default:
		log.Warn("未知的消息类型", "type", msg.Type)
	}
}