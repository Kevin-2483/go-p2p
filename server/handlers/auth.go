package handlers

import (
	"encoding/json"
	"net/http"
	"server/db"
	"server/models"
	"time"

	"github.com/charmbracelet/log"
	"golang.org/x/crypto/bcrypt"
)

// LoginRequest 登录请求结构
type LoginRequest struct {
	Account  string `json:"account"` // 可以是用户名或邮箱
	Password string `json:"password"`
}

// LoginResponse 登录响应结构
type LoginResponse struct {
	Status string      `json:"status"`
	User   *models.User  `json:"user,omitempty"`
	Admin  *models.Admin `json:"admin,omitempty"`
}

// HandleUserLogin 处理用户登录
func HandleUserLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 验证用户凭据
	user := &models.User{}
	var err error

	// 尝试使用邮箱登录
	user, err = db.GetUserByEmail(req.Account)
	if err != nil || user == nil {
		// 如果邮箱登录失败，尝试使用用户名登录
		user, err = db.GetUserByUsername(req.Account)
		if err != nil || user == nil {
			log.Error("用户登录失败：账号不存在", "account", req.Account, "error", err)
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		log.Error("用户登录失败：密码错误", "account", req.Account)
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// 创建新的会话
	session, err := db.CreateSession(user.ID)
	if err != nil {
		log.Error("创建用户会话失败", "user_id", user.ID, "error", err)
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}
	log.Info("用户登录成功", "username", user.Username, "user_id", user.ID)

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

	// 返回用户信息
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(LoginResponse{
		Status: "success",
		User:   user,
	})
}

// HandleAdminLogin 处理管理员登录
func HandleAdminLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 验证管理员凭据
	admin := &models.Admin{}
	admin, err := db.GetAdminByUsername(req.Account)
	if err != nil || admin == nil {
		log.Error("管理员登录失败：用户名不存在", "username", req.Account, "error", err)
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}
	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(req.Password)); err != nil {
		log.Error("管理员登录失败：密码错误", "username", req.Account)
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// 创建新的会话
	session, err := db.CreateSession(admin.ID)
	if err != nil {
		log.Error("创建管理员会话失败", "admin_id", admin.ID, "error", err)
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}
	log.Info("管理员登录成功", "username", admin.Username, "admin_id", admin.ID)

	// 设置Cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "admin_session_token",
		Value:    session.Token,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	// 返回管理员信息
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(LoginResponse{
		Status: "success",
		Admin:  admin,
	})
}

// HandleLogout 处理用户登出
func HandleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取会话Cookie
	cookie, err := r.Cookie("session_token")
	if err == nil {
		// 删除会话记录
		if err := db.DeleteSession(cookie.Value); err != nil {
			log.Error("删除用户会话失败", "session_token", cookie.Value, "error", err)
		} else {
			log.Info("用户登出成功", "session_token", cookie.Value)
		}
	}

	// 删除Cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		Expires:  time.Now(),
		HttpOnly: true,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
	})
}

// HandleAdminLogout 处理管理员登出
func HandleAdminLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取管理员会话Cookie
	cookie, err := r.Cookie("admin_session_token")
	if err == nil {
		// 删除会话记录
		if err := db.DeleteSession(cookie.Value); err != nil {
			log.Error("删除管理员会话失败", "session_token", cookie.Value, "error", err)
		} else {
			log.Info("管理员登出成功", "session_token", cookie.Value)
		}
	}

	// 删除Cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "admin_session_token",
		Value:    "",
		Path:     "/",
		Expires:  time.Now(),
		HttpOnly: true,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
	})
}