package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/log"
	"server/handlers"
	"server/db"
)

func main() {
	// 创建logs文件夹
	logsDir := "logs"
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		log.Fatal("创建日志目录失败", "error", err)
	}

	// 创建日志文件
	timestamp := time.Now().Format("2006-01-02-15-04-05")
	logFile := filepath.Join(logsDir, fmt.Sprintf("%s.log", timestamp))
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal("创建日志文件失败", "error", err)
	}
	defer file.Close()

	// 配置日志记录器，同时输出到文件和控制台
	log.SetOutput(io.MultiWriter(
		file,
		os.Stdout,
	))
	log.SetLevel(log.DebugLevel)

	// 初始化数据库
	if err := db.Init(); err != nil {
		log.Fatal("数据库初始化失败", "error", err)
	}

	// 设置路由
	http.HandleFunc("/ws", handlers.HandleWebSocket)
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