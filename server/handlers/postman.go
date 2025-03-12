package handlers

import (
	"encoding/json"
	"os"
	"net/http"
	"path/filepath"

	"github.com/charmbracelet/log"
)

// HandlePostmanConfig 处理Postman配置文件的请求
func HandlePostmanConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 读取Postman配置文件
	filePath := filepath.Join(".", "postman_collection.json")
	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Error("读取Postman配置文件失败", "error", err)
		http.Error(w, "Failed to read Postman configuration", http.StatusInternalServerError)
		return
	}

	// 验证JSON格式
	var jsonData interface{}
	if err := json.Unmarshal(content, &jsonData); err != nil {
		log.Error("Postman配置文件格式无效", "error", err)
		http.Error(w, "Invalid Postman configuration format", http.StatusInternalServerError)
		return
	}

	// 设置响应头
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=postman_collection.json")

	// 返回配置文件内容
	w.Write(content)
	log.Info("成功返回Postman配置文件")
}