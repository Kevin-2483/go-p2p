package handlers

import (
	"server/models"

	"github.com/charmbracelet/log"
)

// handleOffer 处理WebRTC offer信令
func handleOffer(client *models.Client, msg *models.Message) {
	// 验证消息格式
	if msg.TargetID == "" || msg.SDP == "" {
		log.Error("无效的offer消息格式", "client_id", client.ID)
		return
	}

	// 记录日志
	log.Info("收到offer信令", "source_id", client.ID, "target_id", msg.TargetID)

	// 查找目标客户端
	clientsLock.RLock()
	targetClient, exists := clients[msg.TargetID]
	clientsLock.RUnlock()

	if !exists {
		log.Error("目标客户端不存在", "target_id", msg.TargetID)
		return
	}

	// 构建转发消息
	forwardMsg := models.Message{
		Type:     "offer",
		SDP:      msg.SDP,
		SourceID: client.ID,
		TargetID: msg.TargetID,
		SpaceID:  msg.SpaceID,
	}

	// 转发offer到目标客户端
	if err := targetClient.Conn.WriteJSON(forwardMsg); err != nil {
		log.Error("转发offer失败", "error", err, "target_id", msg.TargetID)
	}
}

// handleAnswer 处理WebRTC answer信令
func handleAnswer(client *models.Client, msg *models.Message) {
	// 验证消息格式
	if msg.TargetID == "" || msg.SDP == "" {
		log.Error("无效的answer消息格式", "client_id", client.ID)
		return
	}

	// 记录日志
	log.Info("收到answer信令", "source_id", client.ID, "target_id", msg.TargetID)

	// 查找目标客户端
	clientsLock.RLock()
	targetClient, exists := clients[msg.TargetID]
	clientsLock.RUnlock()

	if !exists {
		log.Error("目标客户端不存在", "target_id", msg.TargetID)
		return
	}

	// 构建转发消息
	forwardMsg := models.Message{
		Type:     "answer",
		SDP:      msg.SDP,
		SourceID: client.ID,
		TargetID: msg.TargetID,
		SpaceID:  msg.SpaceID,
	}

	// 转发answer到目标客户端
	if err := targetClient.Conn.WriteJSON(forwardMsg); err != nil {
		log.Error("转发answer失败", "error", err, "target_id", msg.TargetID)
	}
}

// HandleICECandidates 处理WebRTC ICE候选信息
func HandleICECandidates(client *models.Client, msg *models.Message) {
	// 验证消息格式
	if msg.TargetID == "" || len(msg.ICECandidates) == 0 {
		log.Error("无效的ICE候选消息格式", "client_id", client.ID)
		return
	}

	// 记录日志
	log.Info("收到ICE候选信息", "source_id", client.ID, "target_id", msg.TargetID, "candidates_count", len(msg.ICECandidates))

	// 查找目标客户端
	clientsLock.RLock()
	targetClient, exists := clients[msg.TargetID]
	clientsLock.RUnlock()

	if !exists {
		log.Error("目标客户端不存在", "target_id", msg.TargetID)
		return
	}

	// 构建转发消息
	forwardMsg := models.Message{
		Type:          "ice_candidates",
		ICECandidates: msg.ICECandidates,
		SourceID:      msg.SourceID,
		TargetID:      msg.TargetID,
		FromClientID:  client.ID,
	}

	// 转发ICE候选到目标客户端
	if err := targetClient.Conn.WriteJSON(forwardMsg); err != nil {
		log.Error("转发ICE候选失败", "error", err, "target_id", msg.TargetID)
	}
}
