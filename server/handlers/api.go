package handlers

import (
	"encoding/json"
	"net/http"
	"server/db"
	"server/models"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
)

// HandleRegister 处理用户注册请求
func HandleRegister(w http.ResponseWriter, r *http.Request) {
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