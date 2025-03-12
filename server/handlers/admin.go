package handlers

import (
	"encoding/json"
	"net/http"
	"server/db"
	"server/models"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
)

// RegisterRequest 注册请求结构
type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// HandleAdminRegister 处理管理员注册
func HandleAdminRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 创建新管理员
	admin := &models.Admin{
		ID:       uuid.New().String(),
		Username: req.Username,
		Password: req.Password,
	}

	// 保存管理员信息
	if err := db.SaveAdmin(admin); err != nil {
		log.Error("管理员注册失败", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Info("管理员注册成功", "username", admin.Username)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
	})
}

// HandleAdminList 处理管理员列表查询
func HandleAdminList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 验证管理员身份
	admin := r.Context().Value(AdminKey).(*models.Admin)
	if admin == nil {
		http.Error(w, "需要管理员权限", http.StatusForbidden)
		return
	}

	// 获取管理员列表
	admins, err := db.GetAdminList()
	if err != nil {
		log.Error("获取管理员列表失败", "error", err)
		http.Error(w, "Failed to get admin list", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data":   admins,
	})
}

// HandleAdminUpdate 处理管理员信息更新
func HandleAdminUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 验证管理员身份
	currentAdmin := r.Context().Value(AdminKey).(*models.Admin)
	if currentAdmin == nil {
		http.Error(w, "需要管理员权限", http.StatusForbidden)
		return
	}

	var updateAdmin models.Admin
	if err := json.NewDecoder(r.Body).Decode(&updateAdmin); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 确保管理员只能更新自己的信息
	if currentAdmin.ID != updateAdmin.ID {
		http.Error(w, "只能更新自己的信息", http.StatusForbidden)
		return
	}

	// 更新管理员信息
	if err := db.UpdateAdmin(&updateAdmin); err != nil {
		log.Error("更新管理员信息失败", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Info("管理员信息更新成功", "username", updateAdmin.Username)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
	})
}

// HandleAdminDelete 处理管理员删除
func HandleAdminDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 验证管理员身份
	currentAdmin := r.Context().Value(AdminKey).(*models.Admin)
	if currentAdmin == nil {
		http.Error(w, "需要管理员权限", http.StatusForbidden)
		return
	}

	// 从URL中获取管理员ID
	adminID := r.URL.Query().Get("id")
	if adminID == "" {
		// 如果没有提供ID，则删除自己的账号
		adminID = currentAdmin.ID
	}

	// 确保管理员只能删除自己的账号
	if currentAdmin.ID != adminID {
		http.Error(w, "只能删除自己的账号", http.StatusForbidden)
		return
	}

	// 删除管理员
	if err := db.DeleteAdmin(adminID); err != nil {
		log.Error("删除管理员失败", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Info("管理员删除成功", "admin_id", adminID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
	})
}