package handlers

import (
	"net/http"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gorilla/websocket"
	"github.com/google/uuid"
	"server/models"
)

var (
	// 全局客户端连接管理器
	clients    = make(map[string]*models.Client)
	monitors   = make(map[string]*websocket.Conn)
	clientsLock sync.RWMutex
)

// HandleInfoWebSocket 处理WebSocket信息监控连接
func HandleInfoWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("WebSocket信息监控连接升级失败", "error", err)
		return
	}
	defer conn.Close()

	// 为监控连接生成唯一ID
	monitorID := uuid.New().String()
	
	// 注册监控连接
	clientsLock.Lock()
	monitors[monitorID] = conn
	clientsLock.Unlock()

	// 清理函数
	defer func() {
		clientsLock.Lock()
		delete(monitors, monitorID)
		clientsLock.Unlock()
	}()

	// 立即发送当前客户端状态
	sendClientsInfo(conn)

	// 保持连接并处理可能的错误
	for {
		// 读取消息（用于检测连接状态）
		if _, _, err := conn.ReadMessage(); err != nil {
			log.Error("监控连接断开", "error", err)
			break
		}
	}
}

// RegisterClient 注册新的客户端连接
func RegisterClient(client *models.Client) {
	// 设置客户端ID和连接时间
	client.ConnectedAt = time.Now()
	client.LastPingTime = time.Now()

	// 注册客户端
	clientsLock.Lock()
	clients[client.ID] = client
	clientsLock.Unlock()

	// 广播客户端状态更新
	broadcastClientsInfo()
}

// UnregisterClient 注销客户端连接
func UnregisterClient(clientID string) {
	clientsLock.Lock()
	delete(clients, clientID)
	clientsLock.Unlock()

	// 广播客户端状态更新
	broadcastClientsInfo()
}

// UpdateClientPing 更新客户端的ping信息
func UpdateClientPing(clientID string, pingDelay int64) {
	clientsLock.Lock()
	if client, ok := clients[clientID]; ok {
		client.LastPingTime = time.Now()
		client.LastPingDelay = pingDelay
	}
	clientsLock.Unlock()

	// 广播客户端状态更新
	broadcastClientsInfo()
}

// sendClientsInfo 发送客户端信息到指定的监控连接
func sendClientsInfo(conn *websocket.Conn) {
	clientsLock.RLock()
	clientsList := make([]*models.Client, 0, len(clients))
	for _, client := range clients {
		clientsList = append(clientsList, client)
	}
	clientsLock.RUnlock()

	// 发送客户端列表
	if err := conn.WriteJSON(map[string]interface{}{
		"type": "clients_info",
		"data": clientsList,
	}); err != nil {
		log.Error("发送客户端信息失败", "error", err)
	}
}

// broadcastClientsInfo 广播客户端信息到所有监控连接
func broadcastClientsInfo() {
	clientsLock.RLock()
	clientsList := make([]*models.Client, 0, len(clients))
	for _, client := range clients {
		clientsList = append(clientsList, client)
	}
	clientsLock.RUnlock()

	// 准备广播消息
	message := map[string]interface{}{
		"type": "clients_info",
		"data": clientsList,
	}

	// 广播到所有监控连接
	clientsLock.RLock()
	for _, conn := range monitors {
		if err := conn.WriteJSON(message); err != nil {
			log.Error("广播客户端信息失败", "error", err)
		}
	}
	clientsLock.RUnlock()
}