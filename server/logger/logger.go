package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/log"
)

// InitLogger 初始化日志记录器
func InitLogger(logsDir string) error {
	// 创建logs文件夹
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %w", err)
	}

	// 创建日志文件
	timestamp := time.Now().Format("2006-01-02-15-04-05")
	logFile := filepath.Join(logsDir, fmt.Sprintf("%s.log", timestamp))
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("创建日志文件失败: %w", err)
	}

	// 配置日志记录器，同时输出到文件和控制台
	log.SetOutput(io.MultiWriter(
		file,
		os.Stdout,
	))
	log.SetLevel(log.DebugLevel)

	return nil
}