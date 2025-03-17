package handlers

import (
	"time"
	"server/models"

	"github.com/charmbracelet/log"
)

// handleOffer 处理来自客户端的offer
func handleOffer(client *models.Client, msg *models.Message) {
	if msg.TargetID == "" {
		log.Error("目标客户端ID为空")
		return
	}

	clientsLock.RLock()
	targetClient, exists := clients[msg.TargetID]
	clientsLock.RUnlock()

	if !exists || targetClient.Conn == nil {
		log.Error("目标客户端不存在或未连接")
		return
	}

	// 设置源客户端ID
	msg.SourceID = client.ID
	msg.FromClientID = client.ID

	// 转发offer给目标客户端
	if err := targetClient.Conn.WriteJSON(msg); err != nil {
		log.Error("转发offer失败", "error", err)
	}
}

// handleICECandidate 处理ICE候选者
func HandleICECandidates(client *models.Client, msg *models.Message) {
	if msg.TargetID == "" {
		log.Error("目标客户端ID为空")
		return
	}

	// 根据FromClientID确定消息流向
	targetID := ""
	if msg.FromClientID == msg.TargetID {
		// 如果FromClientID与TargetID相同，说明这是目标发来的ICE候选，应该发送给来源
		targetID = msg.SourceID
	} else if msg.FromClientID == msg.SourceID {
		// 如果FromClientID与SourceID相同，说明这是来源发来的ICE候选，应该发送给目标
		targetID = msg.TargetID
	} else {
		log.Error("无法确定ICE候选消息的流向")
		return
	}

	clientsLock.RLock()
	targetClient, exists := clients[targetID]
	clientsLock.RUnlock()

	if exists && targetClient.Conn != nil {
		// 更新FromClientID
		msg.FromClientID = client.ID

		// 转发ICE候选
		if err := targetClient.Conn.WriteJSON(msg); err != nil {
			log.Error("转发ICE候选失败", "error", err)
		}
	}
}

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

// handleAnswer 处理来自客户端的answer
func handleAnswer(client *models.Client, msg *models.Message) {
	if msg.SourceID == "" {
		log.Error("源客户端ID为空")
		return
	}

	clientsLock.RLock()
	sourceClient, exists := clients[msg.SourceID]
	clientsLock.RUnlock()

	if exists && sourceClient.Conn != nil {
		// 设置FromClientID
		msg.FromClientID = client.ID

		// 转发answer给源客户端
		if err := sourceClient.Conn.WriteJSON(msg); err != nil {
			log.Error("转发answer失败", "error", err)
		}
	}
}