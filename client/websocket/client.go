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
	connMutex      sync.Mutex
	writeMutex     sync.Mutex
	done           chan struct{}
	stopPing       chan struct{}
	isReconnecting atomic.Bool // 使用原子操作确保线程安全
}

// NewClient 创建新的WebSocket客户端
func NewClient(cfg *config.Config) *Client {
	return &Client{
		config: cfg,
		done:   make(chan struct{}),
	}
}

// Connect 连接到WebSocket服务器
func (c *Client) Connect() error {
	// 使用原子操作检查是否已在重连中
	if !c.isReconnecting.CompareAndSwap(false, true) {
		log.Debug("已有重连操作在进行中，跳过此次请求")
		return fmt.Errorf("reconnection already in progress")
	}
	defer c.isReconnecting.Store(false)

	// 获取锁，确保没有其他 goroutine 在同时修改 conn
	c.connMutex.Lock()
	defer c.connMutex.Unlock()

	// 先清理旧连接和相关资源
	c.cleanup()

	// 重新初始化done通道
	c.done = make(chan struct{})

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

	c.conn = conn // 连接成功后保存到 c.conn
	log.Info("已成功连接到服务器")

	// 初始化并启动心跳通道
	c.stopPing = make(chan struct{})

	// 启动消息处理
	go c.handleMessages()

	// 启动心跳检测
	go c.startPingLoop()

	return nil
}

// cleanup 清理连接和相关资源 - 需要在获取connMutex锁的情况下调用
func (c *Client) cleanup() {
	// 停止心跳循环
	if c.stopPing != nil {
		close(c.stopPing)
		c.stopPing = nil
	}

	// 关闭现有连接
	if c.conn != nil {
		// 尝试发送关闭消息，但不等待响应
		_ = c.conn.WriteMessage(websocket.CloseMessage, 
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		c.conn.Close()
		c.conn = nil
	}
}

// Close 安全地关闭客户端
func (c *Client) Close() {
	c.connMutex.Lock()
	defer c.connMutex.Unlock()
	
	// 清理资源
	c.cleanup()
	
	// 确保done通道已关闭，并且只关闭一次
	select {
	case <-c.done:
		// 通道已关闭，不需要再次关闭
	default:
		close(c.done)
	}
}

// Done 返回连接关闭通知通道
func (c *Client) Done() <-chan struct{} {
	return c.done
}

// handleMessages 处理接收到的消息
func (c *Client) handleMessages() {
	defer func() {
		if r := recover(); r != nil {
			log.Error("handleMessages发生panic", "error", r)
			go c.reconnect()
		}
	}()

	for {
		var msg map[string]interface{}
		
		// 获取连接的本地引用，避免锁持有时间过长
		c.connMutex.Lock()
		conn := c.conn
		c.connMutex.Unlock()
		
		if conn == nil {
			log.Warn("连接已关闭，消息处理终止")
			return
		}
		
		// 读取消息
		if err := conn.ReadJSON(&msg); err != nil {
			log.Error("读取消息失败", "error", err)
			go c.reconnect()
			return
		}

		if len(msg) == 0 {
			log.Warn("收到空消息，准备重连")
			go c.reconnect()
			return
		}

		log.Debug("收到原始消息", "raw", msg)

		// 处理pong消息
		if msgType, ok := msg["type"].(string); ok && msgType == "pong" {
			if timestamp, ok := msg["data"].(float64); ok {
				delay := time.Since(c.pingStartTime).Milliseconds()
				serverTime := time.UnixMilli(int64(timestamp))
				log.Info("网络延迟", "delay", fmt.Sprintf("%dms", delay), "server_time", serverTime)
			}
			continue
		}

		log.Info("收到消息", "message", msg)
	}
}

// reconnect 使用指数退避算法进行重连
func (c *Client) reconnect() {
	// 使用原子操作检查是否已在重连中
	if !c.isReconnecting.CompareAndSwap(false, true) {
		log.Debug("已有重连操作在进行中，跳过此次请求")
		return
	}
	defer c.isReconnecting.Store(false)

	// 确保在开始重连前关闭连接资源
	c.connMutex.Lock()
	c.cleanup()
	c.connMutex.Unlock()
	
	// 通知主循环连接已断开
	select {
	case <-c.done:
		// 通道已关闭，不需要再次关闭
	default:
		close(c.done)
	}
}

// startPingLoop 启动心跳检测循环
func (c *Client) startPingLoop() {
	defer func() {
		if r := recover(); r != nil {
			log.Error("pingLoop发生panic", "error", r)
		}
	}()

	ticker := time.NewTicker(time.Duration(c.config.WebSocket.PingInterval) * time.Second)
	defer ticker.Stop()

	// 获取stopPing引用，因为它可能会被替换
	c.connMutex.Lock()
	stopPing := c.stopPing
	c.connMutex.Unlock()

	if stopPing == nil {
		log.Warn("心跳停止通道为空，退出心跳循环")
		return
	}

	for {
		select {
		case <-stopPing:
			log.Debug("心跳循环收到停止信号")
			return
		case <-ticker.C:
			// 获取连接而不持有锁
			c.connMutex.Lock()
			conn := c.conn
			c.connMutex.Unlock()
			
			if conn == nil {
				log.Warn("连接已关闭，心跳循环终止")
				return
			}

			// 发送ping消息
			c.pingStartTime = time.Now()
			log.Debug("发送ping消息")

			// 使用写入锁保护WebSocket写操作
			c.writeMutex.Lock()
			err := conn.WriteJSON(map[string]interface{}{
				"type": "ping",
				"data": c.pingStartTime.UnixMilli(),
			})
			c.writeMutex.Unlock()

			if err != nil {
				log.Error("发送ping消息失败", "error", err)
				go c.reconnect()
				return
			}
		}
	}
}