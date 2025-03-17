package handlers

import (
	"net/http"
	"server/crypto"
	"server/db"
	"server/models"

	"github.com/charmbracelet/log"
	"github.com/gorilla/websocket"
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

	// 等待客户端发送ID
	var msg models.Message
	if err := conn.ReadJSON(&msg); err != nil {
		log.Error("读取客户端ID失败", "error", err)
		return
	}

	if msg.Type != "auth" || msg.Data == nil {
		log.Error("无效的身份验证消息")
		return
	}

	// 解析客户端ID
	clientID, ok := msg.Data.(string)
	if !ok {
		log.Error("客户端ID格式错误")
		return
	}

	// 查询客户端信息
	dbClient, err := db.GetClientByID(clientID)
	if err != nil {
		log.Error("查询客户端信息失败", "error", err)
		return
	}
	if dbClient == nil {
		log.Error("客户端不存在")
		return
	}

	// 生成随机数并使用客户端公钥加密
	challenge, err := crypto.GenerateRandomChallenge()
	if err != nil {
		log.Error("生成随机挑战失败", "error", err)
		return
	}

	encryptedChallenge, err := crypto.EncryptWithPublicKey(challenge, dbClient.PublicKey)
	if err != nil {
		log.Error("加密随机挑战失败", "error", err)
		return
	}

	// 发送加密的随机挑战
	response := models.Message{
		Type: "challenge",
		Data: encryptedChallenge,
	}
	if err := conn.WriteJSON(response); err != nil {
		log.Error("发送随机挑战失败", "error", err)
		return
	}

	// 等待客户端响应
	if err := conn.ReadJSON(&msg); err != nil {
		log.Error("读取客户端响应失败", "error", err)
		return
	}

	if msg.Type != "challenge_response" || msg.Data == nil {
		log.Error("无效的挑战响应")
		return
	}

	// 验证客户端响应
	responseData, ok := msg.Data.(string)
	if !ok || responseData != challenge {
		log.Error("验证失败")
		return
	}

	// 创建新的客户端连接
	client := &models.Client{
		ID:        clientID,
		PublicKey: dbClient.PublicKey,
		Conn:      conn,
	}

	// 注册客户端
	RegisterClient(client)
	defer UnregisterClient(client.ID)

	for {
		// 读取消息
		var msg models.Message
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				log.Info("客户端正常关闭连接")
			} else {
				log.Error("消息读取失败", "error", err)
				UnregisterClient(client.ID)
				return
			}
			break
		}

		// 处理所有消息
		handleMessage(client, &msg)
	}
}

// handleMessage 处理接收到的WebSocket消息
func handleMessage(client *models.Client, msg *models.Message) {
	// 根据消息类型处理不同的信令逻辑
	switch msg.Type {
	case "ping":
		handlePing(client, msg)
	case "offer":
		// 处理offer
		handleOffer(client, msg)
	case "answer":
		// 处理answer
		handleAnswer(client, msg)
	case "ice_candidates":
		// 处理ICE候选列表
		HandleICECandidates(client, msg)
	default:
		log.Warn("未知的消息类型", "type", msg.Type)
	}
}