package main

import (
	"net/http"

	"server/db"
	"server/handlers"
	"server/logger"

	"github.com/charmbracelet/log"
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
	http.HandleFunc("/api/register", handlers.HandleUserRegister)
	http.HandleFunc("/api/login", handlers.HandleUserLogin)
	http.HandleFunc("/api/admin/login", handlers.HandleAdminLogin)
	http.HandleFunc("/api/logout", handlers.RequireUser(handlers.HandleLogout))
	http.HandleFunc("/api/admin/logout", handlers.RequireAdmin(handlers.HandleAdminLogout))

	// 管理员管理API
	http.HandleFunc("/api/admin/register", handlers.RequireAdmin(handlers.HandleAdminRegister))
	http.HandleFunc("/api/admin/list", handlers.RequireAdmin(handlers.HandleAdminList))
	http.HandleFunc("/api/admin/update", handlers.RequireAuth(handlers.HandleAdminUpdate))
	http.HandleFunc("/api/admin/delete", handlers.RequireAuth(handlers.HandleAdminDelete))

	// 用户管理API
	http.HandleFunc("/api/users", handlers.RequireAdmin(handlers.HandleUserCreate))
	http.HandleFunc("/api/users/list", handlers.RequireAuth(handlers.HandleUserList))
	http.HandleFunc("/api/users/update", handlers.RequireAuth(handlers.HandleUserUpdate))
	http.HandleFunc("/api/users/delete", handlers.RequireAuth(handlers.HandleUserDelete))

	// 客户端管理API
	http.HandleFunc("/api/clients/list", handlers.RequireUser(handlers.HandleClientList))
	http.HandleFunc("/api/clients/update", handlers.RequireUser(handlers.HandleClientUpdate))
	http.HandleFunc("/api/clients/delete", handlers.RequireUser(handlers.HandleClientDelete))

	// 空间管理API
	http.HandleFunc("/api/spaces", handlers.RequireUser(handlers.HandleSpaceCreate))
	http.HandleFunc("/api/spaces/list", handlers.RequireUser(handlers.HandleSpaceList))
	http.HandleFunc("/api/spaces/update", handlers.RequireUser(handlers.HandleSpaceUpdate))
	http.HandleFunc("/api/spaces/delete", handlers.RequireUser(handlers.HandleSpaceDelete))

	// TURN服务器管理API
	http.HandleFunc("/api/turns", handlers.RequireUser(handlers.HandleTurnCreate))
	http.HandleFunc("/api/turns/list", handlers.RequireUser(handlers.HandleTurnList))
	http.HandleFunc("/api/turns/update", handlers.RequireUser(handlers.HandleTurnUpdate))
	http.HandleFunc("/api/turns/delete", handlers.RequireUser(handlers.HandleTurnDelete))

	// Postman配置文件API
	http.HandleFunc("/api/postman", handlers.HandlePostmanConfig)

	// 测试连接API
	http.HandleFunc("/api/test/connect", handlers.HandleTestConnect)

	// WebAPIKey管理API
	http.HandleFunc("/api/web_api_key/generate", handlers.RequireUser(handlers.GenerateWebAPIKey))
	http.HandleFunc("/api/web_api_keys", handlers.HandleGetWebAPIKey)

	// 启动服务器
	log.Info("服务器启动", "port", 8080)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("服务器启动失败", "error", err)
	}
}
