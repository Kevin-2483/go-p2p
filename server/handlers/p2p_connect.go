package handlers

import (
	"server/db"
	"server/models"

	"github.com/charmbracelet/log"
	"github.com/pion/webrtc/v3"
)

// 全局ICE候选缓存，用于存储客户端的ICE候选
var iceCandidatesCache = make(map[string][]models.ICECandidate)

// HandleP2PConnect 处理客户端之间的P2P连接请求
func HandleP2PConnect(sourceClient *models.Client, msg *models.Message) {
	if msg.TargetID == "" || msg.SpaceID == "" {
		log.Error("P2P连接请求缺少必要参数")
		return
	}

	// 获取目标客户端信息
	targetClient, err := db.GetClientByID(msg.TargetID)
	if err != nil || targetClient == nil {
		log.Error("目标客户端不存在", "error", err)
		return
	}

	// 确保两个客户端在同一个空间
	if sourceClient.SpaceID != targetClient.SpaceID || sourceClient.SpaceID != msg.SpaceID {
		log.Error("客户端不在同一个空间")
		return
	}

	// 查找目标客户端的WebSocket连接
	clientsLock.RLock()
	targetConn, exists := clients[msg.TargetID]
	clientsLock.RUnlock()

	if !exists || targetConn.Conn == nil {
		log.Error("目标客户端未连接")
		return
	}

	// 获取空间内的TURN服务器配置
	turnServers, err := db.GetTurnsBySpaceID(msg.SpaceID)
	if err != nil {
		log.Error("获取TURN服务器配置失败", "error", err)
		// 继续执行，因为TURN服务器不是必须的
	}

	// 构建连接请求消息
	connectMsg := models.Message{
		Type:     "connect",
		SourceID: sourceClient.ID,
		TargetID: targetClient.ID,
		SpaceID:  msg.SpaceID,
		Data: map[string]interface{}{
			"turn_servers": turnServers,
		},
	}

	// 发送连接请求给目标客户端
	if err := targetConn.Conn.WriteJSON(connectMsg); err != nil {
		log.Error("发送连接请求失败", "error", err)
		return
	}

	log.Info("已发送P2P连接请求", "source_id", sourceClient.ID, "target_id", targetClient.ID)
}

// HandleICECandidates 处理客户端发送的ICE候选列表
func HandleICECandidates(sourceClient *models.Client, msg *models.Message) {
	if msg.TargetID == "" || len(msg.ICECandidates) == 0 {
		log.Error("ICE候选请求缺少必要参数")
		return
	}

	// 获取目标客户端信息
	targetClient, err := db.GetClientByID(msg.TargetID)
	if err != nil || targetClient == nil {
		log.Error("目标客户端不存在", "error", err)
		return
	}

	// 确保两个客户端在同一个空间
	if sourceClient.SpaceID != targetClient.SpaceID {
		log.Error("客户端不在同一个空间")
		return
	}

	// 获取空间内的TURN服务器配置
	turnServers, err := db.GetTurnsBySpaceID(sourceClient.SpaceID)
	if err != nil {
		log.Error("获取TURN服务器配置失败", "error", err)
		// 继续执行，因为TURN服务器不是必须的
	}

	// 添加STUN和TURN服务器信息
	iceServers := []webrtc.ICEServer{
		{
			URLs: []string{"stun:stun.l.google.com:19302"},
		},
	}

	// 添加TURN服务器
	for _, turn := range turnServers {
		iceServers = append(iceServers, webrtc.ICEServer{
			URLs:       []string{turn.URL},
			Username:   turn.Username,
			Credential: turn.Password,
		})
	}

	// 查找目标客户端的WebSocket连接
	clientsLock.RLock()
	targetConn, exists := clients[msg.TargetID]
	clientsLock.RUnlock()

	if !exists || targetConn.Conn == nil {
		log.Error("目标客户端未连接")
		return
	}

	// 构建ICE候选消息
	iceCandidatesMsg := models.Message{
		Type:          "ice_candidates",
		SourceID:      sourceClient.ID,
		TargetID:      targetClient.ID,
		SpaceID:       sourceClient.SpaceID,
		ICECandidates: msg.ICECandidates,
		Data: map[string]interface{}{
			"ice_servers": iceServers,
		},
	}

	// 发送ICE候选给目标客户端
	if err := targetConn.Conn.WriteJSON(iceCandidatesMsg); err != nil {
		log.Error("发送ICE候选失败", "error", err)
		return
	}

	log.Info("已交换ICE候选", "source_id", sourceClient.ID, "target_id", targetClient.ID)
}