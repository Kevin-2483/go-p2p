package handlers

import (
	"encoding/json"
	"net/http"
	"server/db"
	"server/models"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
)

// HandleSpaceCreate 处理空间创建
func HandleSpaceCreate(w http.ResponseWriter, r *http.Request) {
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

	var space models.Space
	if err := json.NewDecoder(r.Body).Decode(&space); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 设置空间ID和所有者ID
	space.ID = uuid.New().String()
	space.OwnerID = user.ID

	// 保存空间信息
	if err := db.SaveSpace(&space); err != nil {
		log.Error("创建空间失败", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Info("空间创建成功", "space_id", space.ID, "owner_id", space.OwnerID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data":   space,
	})
}

// HandleSpaceList 处理空间列表查询
func HandleSpaceList(w http.ResponseWriter, r *http.Request) {
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

	// 获取用户的所有空间
	spaces, err := db.GetSpacesByOwnerID(user.ID)
	if err != nil {
		log.Error("获取空间列表失败", "error", err)
		http.Error(w, "Failed to get space list", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data":   spaces,
	})
}

// HandleSpaceUpdate 处理空间信息更新
func HandleSpaceUpdate(w http.ResponseWriter, r *http.Request) {
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

	var updateSpace models.Space
	if err := json.NewDecoder(r.Body).Decode(&updateSpace); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 确保只能更新自己的空间
	updateSpace.OwnerID = user.ID

	// 更新空间信息
	if err := db.UpdateSpace(&updateSpace); err != nil {
		log.Error("更新空间信息失败", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Info("空间信息更新成功", "space_id", updateSpace.ID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
	})
}

// HandleSpaceDelete 处理空间删除
func HandleSpaceDelete(w http.ResponseWriter, r *http.Request) {
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

	// 从URL中获取空间ID
	spaceID := r.URL.Query().Get("id")
	if spaceID == "" {
		http.Error(w, "Missing space ID", http.StatusBadRequest)
		return
	}

	// 删除空间
	if err := db.DeleteSpace(spaceID, user.ID); err != nil {
		log.Error("删除空间失败", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Info("空间删除成功", "space_id", spaceID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
	})
}