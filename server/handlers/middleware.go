package handlers

import (
	"context"
	"net/http"
	"server/db"
	"server/models"
	"strings"
	"time"
)

// AuthContext 上下文键
type AuthContext string

const (
	SessionKey AuthContext = "session"
	UserKey    AuthContext = "user"
	AdminKey   AuthContext = "admin"
)

// AuthMiddleware 用于验证用户会话的中间件
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 优先从Cookie中获取token
		var token string
		cookie, err := r.Cookie("session_token")
		if err == nil && cookie.Value != "" {
			token = cookie.Value
		} else {
			// 从请求头中获取认证令牌
			auth := r.Header.Get("Authorization")
			if auth == "" {
				http.Error(w, "未提供认证令牌", http.StatusUnauthorized)
				return
			}

			// 解析Bearer令牌
			parts := strings.Split(auth, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "无效的认证令牌格式", http.StatusUnauthorized)
				return
			}
			token = parts[1]
		}

		// 从数据库验证会话令牌
		session, err := db.GetSessionByToken(token)
		if err != nil || session == nil {
			http.Error(w, "无效的会话", http.StatusUnauthorized)
			return
		}

		// 检查会话是否过期
		if session.ExpiresAt.Before(time.Now()) {
			http.Error(w, "会话已过期", http.StatusUnauthorized)
			return
		}

		// 将会话信息添加到请求上下文
		ctx := context.WithValue(r.Context(), SessionKey, session)
		r = r.WithContext(ctx)

		// 尝试获取用户信息
		user, err := db.GetUserByID(session.UserID)
		if err == nil && user != nil {
			ctx = context.WithValue(r.Context(), UserKey, user)
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
			return
		}

		// 尝试获取管理员信息
		admin, err := db.GetAdminByID(session.UserID)
		if err == nil && admin != nil {
			ctx = context.WithValue(r.Context(), AdminKey, admin)
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
			return
		}

		http.Error(w, "无效的用户身份", http.StatusUnauthorized)
	}
}

// RequireAuth 包装需要认证的处理函数
func RequireAuth(handler http.HandlerFunc) http.HandlerFunc {
	return AuthMiddleware(handler)
}

// RequireUser 验证用户身份并确保只能操作自己的数据
func RequireUser(handler http.HandlerFunc) http.HandlerFunc {
	return AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		// 获取当前用户
		user, ok := r.Context().Value(UserKey).(*models.User)
		if !ok || user == nil {
			http.Error(w, "需要用户权限", http.StatusForbidden)
			return
		}

		// 调用处理函数
		handler.ServeHTTP(w, r)
	})
}

// RequireAdmin 验证管理员身份
func RequireAdmin(handler http.HandlerFunc) http.HandlerFunc {
	return AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		// 获取当前管理员
		admin, ok := r.Context().Value(AdminKey).(*models.Admin)
		if !ok || admin == nil {
			http.Error(w, "需要管理员权限", http.StatusForbidden)
			return
		}

		// 调用处理函数
		handler.ServeHTTP(w, r)
	})
}