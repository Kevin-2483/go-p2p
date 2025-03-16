package handlers

import (
	"encoding/json"
	"net/http"
	"server/db"
	"server/models"
	"time"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
)

// GenerateWebAPIKey 生成一个新的WebAPIKey
func GenerateWebAPIKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取当前用户
	user := r.Context().Value(UserKey).(*models.User)
	if user == nil {
		http.Error(w, "未授权的访问", http.StatusUnauthorized)
		return
	}

	// 解析请求体
	var request struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 验证请求参数
	if request.Name == "" {
		http.Error(w, "客户端名称不能为空", http.StatusBadRequest)
		return
	}

	// 生成新的WebAPIKey
	apiKey := models.NewWebAPIKey(user.ID, request.Name, request.Description)

	// 保存到数据库
	if err := db.SaveWebAPIKey(apiKey); err != nil {
		log.Error("保存WebAPIKey失败", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Info("WebAPIKey生成成功", "key_id", apiKey.ID, "user_id", apiKey.UserID, "name", apiKey.Name)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data":   apiKey,
	})
}

// HandleGetWebAPIKey 获取单个WebAPIKey详细信息
func HandleGetWebAPIKey(w http.ResponseWriter, r *http.Request) {
	// 解析查询参数
	key := r.URL.Query().Get("key")
	publickey := r.URL.Query().Get("publickey")
	log.Info("获取WebAPIKey详细信息", "key", key)

	// 验证WebAPIKey是否有效
	userID, err := ValidateWebAPIKey(key)
	if err != nil {
		log.Error("验证WebAPIKey失败", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if userID == "" {
		http.Error(w, "无效的WebAPIKey", http.StatusUnauthorized)
		return
	}

	// 创建新的客户端
	client := &models.Client{
		ID:      uuid.New().String(),
		OwnerID: userID,
		PublicKey: publickey,
	}

	// 保存客户端信息
	if err := db.SaveClient(client); err != nil {
		log.Error("创建客户端失败", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 标记WebAPIKey为已使用
	if err := db.MarkWebAPIKeyAsUsed(key); err != nil {
		log.Error("标记WebAPIKey已使用失败", "error", err)
		// 不中断流程，继续返回客户端信息
	}

	// 获取服务器端口
	port := 8080 // 默认端口

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"server": map[string]interface{}{
				"host": r.Host,
				"port": port,
			},
			"websocket": map[string]interface{}{
				"path":            "/ws/client",
				"ping_interval":   3,
				"reconnect_delay": 5,
			},
			"client_id": client.ID,
		},
	})
	log.Info("WebAPIKey验证成功，客户端创建成功", "key_id", key, "user_id", userID, "client_id", client.ID)
}

// ValidateWebAPIKey 验证WebAPIKey并返回关联的用户ID
func ValidateWebAPIKey(key string) (string, error) {
	// 获取WebAPIKey
	apiKey, err := db.GetWebAPIKeyByKey(key)
	if err != nil {
		return "", err
	}
	if apiKey == nil {
		return "", nil
	}

	// 检查是否已使用
	if apiKey.Used {
		return "", nil
	}

	// 检查是否过期
	if time.Now().After(apiKey.ExpiresAt) {
		return "", nil
	}

	return apiKey.UserID, nil
}
