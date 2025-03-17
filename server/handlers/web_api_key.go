package handlers

import (
	"encoding/json"
	"net/http"
	"server/db"
	"server/models"
	"time"
	"encoding/pem"
	"crypto/x509"
	"fmt"

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
		SpaceID     string `json:"space_id"`
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

	if request.SpaceID == "" {
		http.Error(w, "空间ID不能为空", http.StatusBadRequest)
		return
	}

	// 验证空间是否存在且属于当前用户
	space, err := db.GetSpaceByID(request.SpaceID)
	if err != nil {
		log.Error("获取空间信息失败", "error", err)
		http.Error(w, "获取空间信息失败", http.StatusInternalServerError)
		return
	}
	if space == nil {
		http.Error(w, "指定的空间不存在", http.StatusBadRequest)
		return
	}
	if space.OwnerID != user.ID {
		http.Error(w, "无权访问该空间", http.StatusForbidden)
		return
	}

	// 生成新的WebAPIKey
	apiKey := models.NewWebAPIKey(user.ID, request.Name, request.Description, request.SpaceID)

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

// ValidatePublicKey 验证PEM格式的RSA公钥
func ValidatePublicKey(publicKey string) error {
    if publicKey == "" {
        return fmt.Errorf("缺少必要的 publickey 参数")
    }

    block, _ := pem.Decode([]byte(publicKey))
    if block == nil || block.Type != "PUBLIC KEY" {
        return fmt.Errorf("无效的公钥格式：需要PEM格式的PUBLIC KEY")
    }

    _, err := x509.ParsePKIXPublicKey(block.Bytes)
    if err != nil {
        return fmt.Errorf("无效的公钥格式: %v", err)
    }

    return nil
}

// HandleGetWebAPIKey 获取单个WebAPIKey详细信息
func HandleGetWebAPIKey(w http.ResponseWriter, r *http.Request) {
	// 解析查询参数
	key := r.URL.Query().Get("key")
	publickey := r.URL.Query().Get("publickey")
	log.Info("获取WebAPIKey详细信息", "key", key)

	// 验证公钥
	if err := ValidatePublicKey(publickey); err != nil {
		log.Error("公钥验证失败", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 验证WebAPIKey是否有效
	apiKey, err := ValidateWebAPIKey(key)
	if err != nil {
		log.Error("验证WebAPIKey失败", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if apiKey == nil {
		http.Error(w, "无效的WebAPIKey", http.StatusUnauthorized)
		return
	}

	// 创建新的客户端
	client := &models.Client{
		ID:          uuid.New().String(),
		OwnerID:     apiKey.UserID,
		SpaceID:     apiKey.SpaceID,
		PublicKey:   publickey,
		Name:        apiKey.Name,
		Description: apiKey.Description,
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
	log.Info("WebAPIKey验证成功，客户端创建成功", "key_id", key, "user_id", apiKey.UserID, "client_id", client.ID)
}

// ValidateWebAPIKey 验证WebAPIKey并返回
func ValidateWebAPIKey(key string) (*models.WebAPIKey, error) {
	// 获取WebAPIKey
	apiKey, err := db.GetWebAPIKeyByKey(key)
	if err != nil {
		return nil, err
	}
	if apiKey == nil {
		return nil, nil
	}

	// 检查是否已使用
	if apiKey.Used {
		return nil, nil
	}

	// 检查是否过期
	if time.Now().After(apiKey.ExpiresAt) {
		return nil, nil
	}

	return apiKey, nil
}
