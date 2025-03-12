package main

import (
	"net/http"

	"github.com/charmbracelet/log"
	"server/handlers"
	"server/db"
	"server/logger"
)

func main() {
	// 初始化日志记录器
	if err := logger.InitLogger("logs"); err != nil {
		log.Fatal("初始化日志记录器失败", "error", err)
	}

	// 初始化数据库
	if err := db.Init(); err != nil {
		log.Fatal("数据库初始化失败", "error", err)
	}

	// 设置路由
	http.HandleFunc("/ws/client", handlers.HandleWebSocket)
	http.HandleFunc("/ws/info", handlers.HandleInfoWebSocket)
	http.HandleFunc("/api/register", handlers.HandleRegister)
	http.HandleFunc("/api/login", handlers.HandleUserLogin)
	http.HandleFunc("/api/admin/login", handlers.HandleAdminLogin)
	http.HandleFunc("/api/logout", handlers.RequireUser(handlers.HandleLogout))
	http.HandleFunc("/api/admin/logout", handlers.RequireAdmin(handlers.HandleAdminLogout))

	// 管理员管理API
	http.HandleFunc("/api/admin/register", handlers.RequireAuth(handlers.HandleAdminRegister))
	http.HandleFunc("/api/admin/list", handlers.RequireAuth(handlers.HandleAdminList))
	http.HandleFunc("/api/admin/update", handlers.RequireAuth(handlers.HandleAdminUpdate))
	http.HandleFunc("/api/admin/delete", handlers.RequireAuth(handlers.HandleAdminDelete))

	// 用户管理API
	http.HandleFunc("/api/users", handlers.RequireAdmin(handlers.HandleUserCreate))
	http.HandleFunc("/api/users/list", handlers.RequireAuth(handlers.HandleUserList))
	http.HandleFunc("/api/users/update", handlers.RequireAuth(handlers.HandleUserUpdate))
	http.HandleFunc("/api/users/delete", handlers.RequireAuth(handlers.HandleUserDelete))

	// 空间管理API
	http.HandleFunc("/api/spaces", handlers.HandleSpaces)
	http.HandleFunc("/api/spaces/", handlers.HandleSpaceDetail)

	// TURN服务器管理API
	http.HandleFunc("/api/turn-servers", handlers.HandleTurnServers)
	http.HandleFunc("/api/turn-servers/", handlers.HandleTurnServerDetail)

	// Postman配置文件API
	http.HandleFunc("/api/postman", handlers.HandlePostmanConfig)

	// 启动服务器
	log.Info("服务器启动", "port", 8080)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("服务器启动失败", "error", err)
	}
}