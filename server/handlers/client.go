package handlers

import (
	"encoding/json"
	"net/http"
	"server/db"
	"server/models"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
)

// HandleClientCreate 处理客户端创建
func HandleClientCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取当前用户
	user := r.Context().Value(UserKey).(*models.User)
	if user == nil {
		http.Error(w, "未授权的访问", http.StatusUnauthorized)
		return
	}

	var client models.Client
	if err := json.NewDecoder(r.Body).Decode(&client); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 设置客户端ID和所有者ID
	client.ID = uuid.New().String()
	client.OwnerID = user.ID

	// 保存客户端信息
	if err := db.SaveClient(&client); err != nil {
		log.Error("创建客户端失败", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Info("客户端创建成功", "client_id", client.ID, "owner_id", client.OwnerID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data":   client,
	})
}

// HandleClientList 处理客户端列表查询
func HandleClientList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取当前用户
	user := r.Context().Value(UserKey).(*models.User)
	if user == nil {
		http.Error(w, "未授权的访问", http.StatusUnauthorized)
		return
	}

	// 获取用户的所有客户端
	clients, err := db.GetClientsByOwnerID(user.ID)
	if err != nil {
		log.Error("获取客户端列表失败", "error", err)
		http.Error(w, "Failed to get client list", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data":   clients,
	})
}

// HandleClientUpdate 处理客户端信息更新
func HandleClientUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取当前用户
	user := r.Context().Value(UserKey).(*models.User)
	if user == nil {
		http.Error(w, "未授权的访问", http.StatusUnauthorized)
		return
	}

	var updateClient models.Client
	if err := json.NewDecoder(r.Body).Decode(&updateClient); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 确保只能更新自己的客户端
	updateClient.OwnerID = user.ID

	// 更新客户端信息
	if err := db.UpdateClient(&updateClient); err != nil {
		log.Error("更新客户端信息失败", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Info("客户端信息更新成功", "client_id", updateClient.ID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
	})
}

// HandleClientDelete 处理客户端删除
func HandleClientDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取当前用户
	user := r.Context().Value(UserKey).(*models.User)
	if user == nil {
		http.Error(w, "未授权的访问", http.StatusUnauthorized)
		return
	}

	// 从URL中获取客户端ID
	clientID := r.URL.Query().Get("id")
	if clientID == "" {
		http.Error(w, "Missing client ID", http.StatusBadRequest)
		return
	}

	// 删除客户端
	if err := db.DeleteClient(clientID, user.ID); err != nil {
		log.Error("删除客户端失败", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Info("客户端删除成功", "client_id", clientID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
	})
}