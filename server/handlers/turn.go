package handlers

import (
	"encoding/json"
	"net/http"
	"server/db"
	"server/models"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
)

// HandleTurnCreate 处理TURN服务器配置创建
func HandleTurnCreate(w http.ResponseWriter, r *http.Request) {
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

	var turn models.TurnServer
	if err := json.NewDecoder(r.Body).Decode(&turn); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 设置TURN服务器配置ID和所有者ID
	turn.ID = uuid.New().String()
	turn.OwnerID = user.ID

	// 保存TURN服务器配置
	if err := db.SaveTurn(&turn); err != nil {
		log.Error("创建TURN服务器配置失败", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Info("TURN服务器配置创建成功", "turn_id", turn.ID, "owner_id", turn.OwnerID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data":   turn,
	})
}

// HandleTurnList 处理TURN服务器配置列表查询
func HandleTurnList(w http.ResponseWriter, r *http.Request) {
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

	// 获取用户的所有TURN服务器配置
	turns, err := db.GetTurnsByOwnerID(user.ID)
	if err != nil {
		log.Error("获取TURN服务器配置列表失败", "error", err)
		http.Error(w, "Failed to get turn server list", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data":   turns,
	})
}

// HandleTurnUpdate 处理TURN服务器配置更新
func HandleTurnUpdate(w http.ResponseWriter, r *http.Request) {
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

	var updateTurn models.TurnServer
	if err := json.NewDecoder(r.Body).Decode(&updateTurn); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 确保只能更新自己的TURN服务器配置
	updateTurn.OwnerID = user.ID

	// 更新TURN服务器配置
	if err := db.UpdateTurn(&updateTurn); err != nil {
		log.Error("更新TURN服务器配置失败", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Info("TURN服务器配置更新成功", "turn_id", updateTurn.ID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
	})
}

// HandleTurnDelete 处理TURN服务器配置删除
func HandleTurnDelete(w http.ResponseWriter, r *http.Request) {
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

	// 从URL中获取TURN服务器配置ID
	turnID := r.URL.Query().Get("id")
	if turnID == "" {
		http.Error(w, "Missing turn server ID", http.StatusBadRequest)
		return
	}

	// 删除TURN服务器配置
	if err := db.DeleteTurn(turnID, user.ID); err != nil {
		log.Error("删除TURN服务器配置失败", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Info("TURN服务器配置删除成功", "turn_id", turnID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
	})
}