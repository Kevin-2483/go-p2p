package handlers

import (
	"encoding/json"
	"net/http"
	"server/models"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
)

// HandleSpaces 处理空间相关的请求
func HandleSpaces(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		// 创建新空间
		var space models.Space
		if err := json.NewDecoder(r.Body).Decode(&space); err != nil {
			log.Error("创建空间失败：无效的请求体", "error", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// 生成空间ID
		space.ID = uuid.New().String()
		log.Info("创建新空间", "space_id", space.ID, "name", space.Name)

		// TODO: 保存空间信息到数据库
		// 这里暂时返回模拟数据
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"space":  space,
		})

	case http.MethodGet:
		// 获取空间列表
		// TODO: 从数据库获取空间列表
		// 这里暂时返回模拟数据
		spaces := []models.Space{
			{ID: "1", Name: "测试空间1", Description: "这是测试空间1"},
			{ID: "2", Name: "测试空间2", Description: "这是测试空间2"},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"spaces": spaces,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleSpaceDetail 处理单个空间的请求
func HandleSpaceDetail(w http.ResponseWriter, r *http.Request) {
	// 从URL中获取空间ID
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		log.Error("获取空间详情失败：无效的空间ID")
		http.Error(w, "Invalid space ID", http.StatusBadRequest)
		return
	}
	spaceID := parts[3]

	// TODO: 从数据库获取空间信息
	// 这里暂时返回模拟数据
	space := models.Space{
		ID:          spaceID,
		Name:        "测试空间",
		Description: "这是一个测试空间",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"space":  space,
	})
}

// HandleTurnServers 处理TURN服务器相关的请求
func HandleTurnServers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		// 添加新的TURN服务器
		var server models.TurnServer
		if err := json.NewDecoder(r.Body).Decode(&server); err != nil {
			log.Error("创建TURN服务器失败：无效的请求体", "error", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// 生成服务器ID
		server.ID = uuid.New().String()
		log.Info("创建新TURN服务器", "server_id", server.ID, "url", server.URL, "space_id", server.SpaceID)

		// TODO: 保存TURN服务器信息到数据库
		// 这里暂时返回模拟数据
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"server": server,
		})

	case http.MethodGet:
		// 获取TURN服务器列表
		// TODO: 从数据库获取服务器列表
		// 这里暂时返回模拟数据
		servers := []models.TurnServer{
			{ID: "1", URL: "turn:example1.com:3478", Username: "test1", SpaceID: "space1"},
			{ID: "2", URL: "turn:example2.com:3478", Username: "test2", SpaceID: "space2"},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "success",
			"servers": servers,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleTurnServerDetail 处理单个TURN服务器的请求
func HandleTurnServerDetail(w http.ResponseWriter, r *http.Request) {
	// 从URL中获取服务器ID
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		log.Error("获取TURN服务器详情失败：无效的服务器ID")
		http.Error(w, "Invalid server ID", http.StatusBadRequest)
		return
	}
	serverID := parts[3]

	// TODO: 从数据库获取服务器信息
	// 这里暂时返回模拟数据
	server := models.TurnServer{
		ID:       serverID,
		URL:      "turn:example.com:3478",
		Username: "test",
		SpaceID:  "space_id_here",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"server": server,
	})
}