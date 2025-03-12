package handlers

import (
	"encoding/json"
	"net/http"
	"server/db"
	"server/models"
	"github.com/charmbracelet/log"
	"github.com/google/uuid"
)

// HandleUserCreate 处理用户创建
func HandleUserCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var user models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 设置用户ID
	user.ID = uuid.New().String()

	// 保存用户信息
	if err := db.SaveUser(&user); err != nil {
		log.Error("创建用户失败", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Info("用户创建成功", "username", user.Username)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
	})
}

// HandleUserList 处理用户列表查询
func HandleUserList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取当前用户或管理员身份
	admin, ok := r.Context().Value(AdminKey).(*models.Admin)
	if !ok || admin == nil {
		// 非管理员只能查看自己的信息
		user, ok := r.Context().Value(UserKey).(*models.User)
		if !ok || user == nil {
			http.Error(w, "未授权的访问", http.StatusUnauthorized)
			return
		}
		// 返回用户自己的信息
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"data":   []models.User{*user},
		})
		return
	}

	// 管理员可以查看所有用户列表
	users, err := db.GetUserList()
	if err != nil {
		log.Error("获取用户列表失败", "error", err)
		http.Error(w, "Failed to get user list", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data":   users,
	})
}

// HandleUserUpdate 处理用户信息更新
func HandleUserUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var updateUser models.User
	if err := json.NewDecoder(r.Body).Decode(&updateUser); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 获取当前用户或管理员身份
	admin := r.Context().Value(AdminKey).(*models.Admin)
	if admin == nil {
		// 非管理员只能更新自己的信息
		user := r.Context().Value(UserKey).(*models.User)
		if user == nil || user.ID != updateUser.ID {
			http.Error(w, "未授权的操作", http.StatusForbidden)
			return
		}
	}

	// 更新用户信息
	if err := db.UpdateUser(&updateUser); err != nil {
		log.Error("更新用户信息失败", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Info("用户信息更新成功", "username", updateUser.Username)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
	})
}

// HandleUserDelete 处理用户删除
func HandleUserDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取当前用户或管理员身份
	admin := r.Context().Value(AdminKey).(*models.Admin)
	if admin == nil {
		// 非管理员只能删除自己的账号
		user := r.Context().Value(UserKey).(*models.User)
		if user == nil {
			http.Error(w, "未授权的操作", http.StatusForbidden)
			return
		}
		// 删除自己的账号
		if err := db.DeleteUser(user.ID); err != nil {
			log.Error("删除用户失败", "error", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Info("用户删除成功", "user_id", user.ID)
	} else {
		// 管理员可以删除指定用户
		userID := r.URL.Query().Get("id")
		if userID == "" {
			http.Error(w, "Missing user ID", http.StatusBadRequest)
			return
		}
		if err := db.DeleteUser(userID); err != nil {
			log.Error("删除用户失败", "error", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Info("管理员删除用户成功", "user_id", userID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
	})
}

// HandleRegister 处理用户注册请求
func HandleUserRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var user models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		log.Error("用户注册失败：无效的请求体", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 生成用户ID
	user.ID = uuid.New().String()
	log.Info("开始创建新用户", "user_id", user.ID, "username", user.Username)

	// 保存用户信息到数据库
	if err := db.SaveUser(&user); err != nil {
		log.Error("用户注册失败", "error", err)
		if err.Error() == "用户名已存在" {
			http.Error(w, "Username already exists", http.StatusConflict)
			return
		}
		if err.Error() == "邮箱已被注册" {
			http.Error(w, "Email already registered", http.StatusConflict)
			return
		}
		http.Error(w, "Failed to save user", http.StatusInternalServerError)
		return
	}

	// 创建新的会话
	session, err := db.CreateSession(user.ID)
	if err != nil {
		log.Error("创建会话失败", "error", err)
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// 设置Cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    session.Token,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	// 返回用户信息和会话令牌
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"user": user,
		"token": session.Token,
	})
	log.Info("用户注册成功", "username", user.Username, "email", user.Email)
}