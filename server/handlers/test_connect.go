package handlers

import (
	"encoding/json"
	"net/http"
	"server/db"
	"server/models"

	"github.com/charmbracelet/log"
)

// TestConnectRequest 测试连接请求结构
type TestConnectRequest struct {
	SourceID string `json:"source_id"`
	TargetID string `json:"target_id"`
	SpaceID  string `json:"space_id"`
}

// HandleTestConnect 处理测试连接API请求
func HandleTestConnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析请求体
	var req TestConnectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error("解析请求体失败", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 验证请求参数
	if req.SourceID == "" || req.TargetID == "" || req.SpaceID == "" {
		log.Error("请求参数不完整")
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	// 获取源客户端信息
	sourceClient, err := db.GetClientByID(req.SourceID)
	if err != nil || sourceClient == nil {
		log.Error("源客户端不存在", "error", err)
		http.Error(w, "Source client not found", http.StatusNotFound)
		return
	}

	// 获取目标客户端信息
	targetClient, err := db.GetClientByID(req.TargetID)
	if err != nil || targetClient == nil {
		log.Error("目标客户端不存在", "error", err)
		http.Error(w, "Target client not found", http.StatusNotFound)
		return
	}

	// 确保两个客户端在同一个空间
	if sourceClient.SpaceID != req.SpaceID || targetClient.SpaceID != req.SpaceID {
		log.Error("客户端不在指定空间")
		http.Error(w, "Clients are not in the specified space", http.StatusBadRequest)
		return
	}

	// 查找源客户端的WebSocket连接
	clientsLock.RLock()
	sourceConn, sourceExists := clients[req.SourceID]
	clientsLock.RUnlock()

	if !sourceExists || sourceConn.Conn == nil {
		log.Error("源客户端未连接")
		http.Error(w, "Source client not connected", http.StatusBadRequest)
		return
	}

	// 查找目标客户端的WebSocket连接
	clientsLock.RLock()
	targetConn, targetExists := clients[req.TargetID]
	clientsLock.RUnlock()

	if !targetExists || targetConn.Conn == nil {
		log.Error("目标客户端未连接")
		http.Error(w, "Target client not connected", http.StatusBadRequest)
		return
	}

	// 获取空间内的TURN服务器配置
	turnServers, err := db.GetTurnsBySpaceID(req.SpaceID)
	if err != nil {
		log.Error("获取TURN服务器配置失败", "error", err)
		// 继续执行，因为TURN服务器不是必须的
	}

	// 构建连接请求消息
	connectMsg := models.Message{
		Type:     "connect",
		SourceID: sourceClient.ID,
		TargetID: targetClient.ID,
		SpaceID:  req.SpaceID,
		Data: map[string]interface{}{
			"turn_servers": turnServers,
		},
	}

	// 发送连接请求给源客户端
	if err := sourceConn.Conn.WriteJSON(connectMsg); err != nil {
		log.Error("发送连接请求给源客户端失败", "error", err)
		http.Error(w, "Failed to send connect request to source client", http.StatusInternalServerError)
		return
	}

	// 发送连接请求给目标客户端
	if err := targetConn.Conn.WriteJSON(connectMsg); err != nil {
		log.Error("发送连接请求给目标客户端失败", "error", err)
		http.Error(w, "Failed to send connect request to target client", http.StatusInternalServerError)
		return
	}

	log.Info("已发送测试P2P连接请求", "source_id", sourceClient.ID, "target_id", targetClient.ID)

	// 返回成功响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"message": "Connection request sent to both clients",
	})
}