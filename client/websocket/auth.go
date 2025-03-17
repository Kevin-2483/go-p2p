package websocket

import (
	"fmt"

	"client/config"
	"client/crypto"

	"github.com/charmbracelet/log"
	"github.com/gorilla/websocket"
)

// Authenticator WebSocket认证器结构
type Authenticator struct {
	conn   *websocket.Conn
	config *config.Config
}

// NewAuthenticator 创建新的WebSocket认证器
func NewAuthenticator(conn *websocket.Conn, cfg *config.Config) *Authenticator {
	return &Authenticator{
		conn:   conn,
		config: cfg,
	}
}

// Authenticate 执行WebSocket连接的身份验证
func (a *Authenticator) Authenticate() error {
	// 发送身份验证消息
	authMsg := map[string]interface{}{
		"type": "auth",
		"data": a.config.Client.ID,
	}
	if err := a.conn.WriteJSON(authMsg); err != nil {
		log.Error("发送身份验证消息失败", "error", err)
		return err
	}

	// 等待服务器的挑战
	var challengeMsg map[string]interface{}
	if err := a.conn.ReadJSON(&challengeMsg); err != nil {
		log.Error("读取服务器挑战失败", "error", err)
		return err
	}

	if challengeMsg["type"] != "challenge" || challengeMsg["data"] == nil {
		log.Error("无效的服务器挑战")
		return fmt.Errorf("无效的服务器挑战")
	}

	// 解密挑战
	encryptedChallenge, ok := challengeMsg["data"].(string)
	if !ok {
		log.Error("服务器挑战格式错误")
		return fmt.Errorf("服务器挑战格式错误")
	}

	challenge, err := crypto.DecryptWithPrivateKey(encryptedChallenge)
	if err != nil {
		log.Error("解密服务器挑战失败", "error", err)
		return err
	}

	// 发送挑战响应
	responseMsg := map[string]interface{}{
		"type": "challenge_response",
		"data": challenge,
	}
	if err := a.conn.WriteJSON(responseMsg); err != nil {
		log.Error("发送挑战响应失败", "error", err)
		return err
	}

	return nil
}
