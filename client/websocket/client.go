package websocket

import (
	"fmt"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"client/config"
	"client/webrtc"

	"github.com/charmbracelet/log"
	"github.com/gorilla/websocket"
)

// Client WebSocket客户端结构
type Client struct {
	conn           *websocket.Conn
	config         *config.Config
	pingStartTime  time.Time
	mu             sync.RWMutex  // 单一互斥锁用于保护conn和控制通道
	writeMu        sync.Mutex    // 写操作的互斥锁
	done           chan struct{} // 连接状态通知通道
	control        chan struct{} // 控制通道，用于停止心跳等操作
	isReconnecting atomic.Bool   // 使用原子操作标记重连状态
	webrtcClient   interface{}   // WebRTC客户端引用
}

// SetWebRTCClient 设置WebRTC客户端
func (c *Client) SetWebRTCClient(client interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.webrtcClient = client
}

// NewClient 创建新的WebSocket客户端
func NewClient(cfg *config.Config) *Client {
	return &Client{
		config:  cfg,
		done:    make(chan struct{}),
		control: make(chan struct{}),
	}
}

// Connect 连接到WebSocket服务器
func (c *Client) Connect() error {
	// 使用原子操作检查是否已在重连中，避免重复连接
	if !c.isReconnecting.CompareAndSwap(false, true) {
		log.Debug("已有连接操作在进行中")
		return nil
	}
	defer c.isReconnecting.Store(false)

	// 关闭现有资源
	c.closeResources()

	// 初始化新的控制通道
	c.mu.Lock()
	c.control = make(chan struct{})
	c.done = make(chan struct{})
	c.mu.Unlock()

	// 构建WebSocket URL
	u := url.URL{
		Scheme: "ws",
		Host:   fmt.Sprintf("%s:%d", c.config.Server.Host, c.config.Server.Port),
		Path:   c.config.WebSocket.Path,
	}

	log.Info("正在连接到服务器", "url", u.String())

	// 建立WebSocket连接
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}

	// 更新连接
	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()
	log.Info("已成功连接到服务器")

	// 执行身份验证
	authenticator := NewAuthenticator(conn, c.config)
	if err := authenticator.Authenticate(); err != nil {
		return err
	}

	// 启动消息处理和心跳检测
	go c.handleMessages()
	go c.startPingLoop()

	return nil
}

// closeResources 安全地关闭资源
func (c *Client) closeResources() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 关闭现有连接
	if c.conn != nil {
		// 尝试发送关闭消息，忽略错误
		_ = c.conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		c.conn.Close()
		c.conn = nil
	}

	// 关闭控制通道
	select {
	case <-c.control:
		// 通道已关闭
	default:
		close(c.control)
	}
}

// Close 关闭客户端
func (c *Client) Close() {
	// 关闭资源
	c.closeResources()

	// 通知连接已关闭
	c.mu.Lock()
	select {
	case <-c.done:
		// 通道已关闭
	default:
		close(c.done)
	}
	c.mu.Unlock()
}

// Done 返回连接关闭通知通道
func (c *Client) Done() <-chan struct{} {
	c.mu.RLock()
	done := c.done
	c.mu.RUnlock()
	return done
}

// MessageHandler 消息处理器结构体
type MessageHandler struct {
	client   *Client
	handlers map[string]func(msg map[string]interface{})
}

// NewMessageHandler 创建新的消息处理器
func NewMessageHandler(client *Client) *MessageHandler {
	h := &MessageHandler{
		client:   client,
		handlers: make(map[string]func(msg map[string]interface{})),
	}

	// 注册消息处理函数
	h.handlers["pong"] = h.handlePong
	h.handlers["connect"] = h.handleConnect
	h.handlers["offer"] = h.handleOffer
	h.handlers["answer"] = h.handleAnswer
	h.handlers["ice_candidates"] = h.handleICECandidates

	return h
}

// handleConnect 处理connect消息
func (h *MessageHandler) handleConnect(msg map[string]interface{}) {
	if h.client.webrtcClient == nil {
		log.Error("WebRTC客户端未初始化")
		return
	}
	// 创建WebRTC消息处理器
	webrtcHandler := webrtc.NewMessageHandler(h.client.webrtcClient.(*webrtc.Client))
	// 调用WebRTC消息处理器的HandleConnect方法
	webrtcHandler.HandleConnect(msg)
}

// handleOffer 处理offer消息
func (h *MessageHandler) handleOffer(msg map[string]interface{}) {
	if h.client.webrtcClient == nil {
		log.Error("WebRTC客户端未初始化")
		return
	}
	// 创建WebRTC消息处理器
	webrtcHandler := webrtc.NewMessageHandler(h.client.webrtcClient.(*webrtc.Client))
	// 调用WebRTC消息处理器的HandleOffer方法
	webrtcHandler.HandleOffer(msg)
}

// handleAnswer 处理answer消息
func (h *MessageHandler) handleAnswer(msg map[string]interface{}) {
	if h.client.webrtcClient == nil {
		log.Error("WebRTC客户端未初始化")
		return
	}
	// 创建WebRTC消息处理器
	webrtcHandler := webrtc.NewMessageHandler(h.client.webrtcClient.(*webrtc.Client))
	// 调用WebRTC消息处理器的HandleAnswer方法
	webrtcHandler.HandleAnswer(msg)
}

// handleICECandidates 处理ice_candidates消息
func (h *MessageHandler) handleICECandidates(msg map[string]interface{}) {
	if h.client.webrtcClient == nil {
		log.Error("WebRTC客户端未初始化")
		return
	}
	// 创建WebRTC消息处理器
	webrtcHandler := webrtc.NewMessageHandler(h.client.webrtcClient.(*webrtc.Client))
	// 调用WebRTC消息处理器的HandleICECandidates方法
	webrtcHandler.HandleICECandidates(msg)
}

// handlePong 处理pong消息
func (h *MessageHandler) handlePong(msg map[string]interface{}) {
	if timestamp, ok := msg["data"].(float64); ok {
		delay := time.Since(h.client.pingStartTime).Milliseconds()
		serverTime := time.UnixMilli(int64(timestamp))
		log.Info("网络延迟", "delay", fmt.Sprintf("%dms", delay), "server_time", serverTime)
	}
}

// handleMessages 处理接收到的消息
func (c *Client) handleMessages() {
	defer func() {
		if r := recover(); r != nil {
			log.Error("消息处理发生panic", "error", r)
			c.triggerReconnect()
		}
	}()

	// 获取当前连接的本地引用
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return
	}

	// 创建消息处理器
	msgHandler := NewMessageHandler(c)

	for {
		var msg map[string]interface{}
		if err := conn.ReadJSON(&msg); err != nil {
			log.Error("读取消息失败", "error", err)
			c.triggerReconnect()
			return
		}

		if len(msg) == 0 {
			log.Warn("收到空消息")
			c.triggerReconnect()
			return
		}

		log.Debug("收到消息", "message", msg)

		// 处理WebSocket消息
		if msgType, ok := msg["type"].(string); ok {
			if handler, exists := msgHandler.handlers[msgType]; exists {
				handler(msg)
				continue
			}
			log.Info("收到业务消息", "message", msg)
		}
	}
}

// triggerReconnect 触发重连
func (c *Client) triggerReconnect() {
	// 只有当没有重连进行时才触发
	if c.isReconnecting.CompareAndSwap(false, true) {
		defer c.isReconnecting.Store(false)

		// 关闭资源
		c.closeResources()

		// 通知连接已关闭
		c.mu.Lock()
		select {
		case <-c.done:
			// 通道已关闭
		default:
			close(c.done)
		}
		c.mu.Unlock()

		log.Info("连接已断开，等待重连")
	}
}

// SendJSON 发送JSON消息
func (c *Client) SendJSON(msg map[string]interface{}) error {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("连接未建立")
	}

	// 使用写锁保护写操作
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	return conn.WriteJSON(msg)
}

// startPingLoop 启动心跳检测循环
func (c *Client) startPingLoop() {
	defer func() {
		if r := recover(); r != nil {
			log.Error("心跳循环发生panic", "error", r)
		}
	}()

	// 如果PingInterval为0，则禁用心跳检测
	if c.config.WebSocket.PingInterval == 0 {
		log.Info("心跳检测已禁用")
		return
	}

	ticker := time.NewTicker(time.Duration(c.config.WebSocket.PingInterval) * time.Second)
	defer ticker.Stop()

	// 获取当前控制通道的本地引用
	c.mu.RLock()
	control := c.control
	c.mu.RUnlock()

	for {
		select {
		case <-control:
			log.Debug("心跳循环收到停止信号")
			return
		case <-ticker.C:
			// 使用读锁获取当前连接
			c.mu.RLock()
			conn := c.conn
			c.mu.RUnlock()

			if conn == nil {
				return
			}

			// 获取WebRTC连接状态
			webrtcStatus := make(map[string]string)
			if c.webrtcClient != nil {
				if client, ok := c.webrtcClient.(*webrtc.Client); ok {
					for id, pc := range client.GetPeerConnections() {
						webrtcStatus[id] = pc.ConnectionState().String()
					}
				}
			}

			// 发送ping消息
			c.pingStartTime = time.Now()

			// 使用写锁保护写操作
			c.writeMu.Lock()
			err := conn.WriteJSON(map[string]interface{}{
				"type": "ping",
				"data": map[string]interface{}{
					"timestamp":     c.pingStartTime.UnixMilli(),
					"webrtc_status": webrtcStatus,
				},
			})
			c.writeMu.Unlock()

			if err != nil {
				log.Error("发送ping消息失败", "error", err)
				c.triggerReconnect()
				return
			}
		}
	}
}
