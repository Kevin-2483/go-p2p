package websocket

import (
	"fmt"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"client/config"

	"github.com/charmbracelet/log"
	"github.com/gorilla/websocket"
)

// Client WebSocket客户端结构
type Client struct {
	conn           *websocket.Conn
	config         *config.Config
	pingStartTime  time.Time
	mu             sync.RWMutex       // 单一互斥锁用于保护conn和控制通道
	writeMu        sync.Mutex         // 写操作的互斥锁
	done           chan struct{}      // 连接状态通知通道
	control        chan struct{}      // 控制通道，用于停止心跳等操作
	isReconnecting atomic.Bool        // 使用原子操作标记重连状态
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

		// 处理pong消息
		if msgType, ok := msg["type"].(string); ok && msgType == "pong" {
			if timestamp, ok := msg["data"].(float64); ok {
				delay := time.Since(c.pingStartTime).Milliseconds()
				serverTime := time.UnixMilli(int64(timestamp))
				log.Info("网络延迟", "delay", fmt.Sprintf("%dms", delay), "server_time", serverTime)
			}
			continue
		}

		log.Info("收到业务消息", "message", msg)
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

// startPingLoop 启动心跳检测循环
func (c *Client) startPingLoop() {
	defer func() {
		if r := recover(); r != nil {
			log.Error("心跳循环发生panic", "error", r)
		}
	}()

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

			// 发送ping消息
			c.pingStartTime = time.Now()
			
			// 使用写锁保护写操作
			c.writeMu.Lock()
			err := conn.WriteJSON(map[string]interface{}{
				"type": "ping",
				"data": c.pingStartTime.UnixMilli(),
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