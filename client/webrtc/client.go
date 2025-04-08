package webrtc

import (
	"sync"

	"client/audio"
	"client/config"

	"github.com/charmbracelet/log"
	"github.com/pion/webrtc/v3"
)

// Client WebRTC客户端结构
type Client struct {
	config          *config.Config
	websocketClient interface{} // 使用interface{}避免循环导入
	peerConnections map[string]*webrtc.PeerConnection
	audioManager    audio.AudioManager
	mu              sync.RWMutex
}

// NewClient 创建新的WebRTC客户端
func NewClient(cfg *config.Config, wsClient interface{}) *Client {
	// 创建音频管理器
	var audioManager audio.AudioManager
	if cfg.Audio.Enabled {
		var err error
		audioManager, err = audio.NewManager(cfg)
		if err != nil {
			log.Error("创建音频管理器失败", "error", err)
		} else {
			// 启动音频系统
			if err := audioManager.Start(); err != nil {
				log.Error("启动音频系统失败", "error", err)
				audioManager = nil
			}
		}
	}

	return &Client{
		config:          cfg,
		websocketClient: wsClient,
		peerConnections: make(map[string]*webrtc.PeerConnection),
		audioManager:    audioManager,
	}
}

// GetPeerConnection 获取或创建与指定客户端的PeerConnection
func (c *Client) GetPeerConnection(targetID string) (*webrtc.PeerConnection, error) {
	c.mu.RLock()
	pc, exists := c.peerConnections[targetID]
	c.mu.RUnlock()

	if exists {
		return pc, nil
	}

	// 创建新的PeerConnection配置
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	// 创建新的PeerConnection
	var err error
	pc, err = webrtc.NewPeerConnection(config)
	if err != nil {
		return nil, err
	}

	// 创建数据通道 - 这是必要的，以确保ICE信息正确交换
	_, err = pc.CreateDataChannel("data", nil)
	if err != nil {
		log.Error("创建数据通道失败", "error", err)
		return nil, err
	}

	// 如果音频管理器可用，添加音频轨道
	if c.audioManager != nil {
		_, err = c.audioManager.AddTrack(pc)
		if err != nil {
			log.Error("添加音频轨道失败", "error", err)
		}
	}

	// 设置轨道处理
	pc.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Info("收到媒体轨道", "kind", track.Kind().String(), "id", track.ID())

		// 如果是音频轨道并且音频管理器可用
		if track.Kind() == webrtc.RTPCodecTypeAudio && c.audioManager != nil {
			c.audioManager.OnTrack(track)
		}
	})

	// 设置ICE候选收集处理
	pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}

		c.handleICECandidate(targetID, candidate)
	})

	// 设置连接状态变化处理
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Info("WebRTC连接状态变化", "state", state.String(), "target_id", targetID)

		if state == webrtc.PeerConnectionStateFailed || state == webrtc.PeerConnectionStateClosed {
			c.mu.Lock()
			delete(c.peerConnections, targetID)
			c.mu.Unlock()
		}
	})

	// 保存PeerConnection
	c.mu.Lock()
	c.peerConnections[targetID] = pc
	c.mu.Unlock()

	return pc, nil
}

// handleICECandidate 处理ICE候选并发送到信令服务器
func (c *Client) handleICECandidate(targetID string, candidate *webrtc.ICECandidate) {
	if candidate == nil {
		return
	}

	// 获取候选者信息
	candidateJSON := candidate.ToJSON()

	// 构建ICE候选消息
	iceCandidate := ICECandidate{
		Candidate:     candidateJSON.Candidate,
		SDPMLineIndex: uint16(*candidateJSON.SDPMLineIndex), // 类型转换为uint16
		SDPMid:        *candidateJSON.SDPMid,                // 类型转换
	}

	// 发送ICE候选到信令服务器
	c.sendICECandidates(targetID, []ICECandidate{iceCandidate})
}

// sendICECandidates 发送ICE候选到信令服务器
func (c *Client) sendICECandidates(targetID string, candidates []ICECandidate) {
	// 构建消息
	msg := map[string]interface{}{
		"type":           "ice_candidates",
		"target_id":      targetID,
		"source_id":      c.config.Client.ID,
		"ice_candidates": candidates,
	}

	// 通过WebSocket发送消息
	c.SendJSON(msg)
}

// SendJSON 通过WebSocket发送JSON消息
func (c *Client) SendJSON(msg map[string]interface{}) {
	c.mu.RLock()
	conn := c.websocketClient
	c.mu.RUnlock()

	// 使用类型断言获取具体类型
	if conn != nil {
		// 这里我们假设websocketClient实现了一个SendJSON方法
		// 实际实现时需要根据具体接口调整
		type jsonSender interface {
			SendJSON(msg map[string]interface{}) error
		}

		if sender, ok := conn.(jsonSender); ok {
			err := sender.SendJSON(msg)
			if err != nil {
				log.Error("发送消息失败", "error", err)
			}
		} else {
			log.Error("websocketClient不支持SendJSON方法")
		}
	}
}

// GetPeerConnections 获取所有PeerConnection
func (c *Client) GetPeerConnections() map[string]*webrtc.PeerConnection {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 创建一个新的map来存储PeerConnection的副本
	pcs := make(map[string]*webrtc.PeerConnection)
	for id, pc := range c.peerConnections {
		pcs[id] = pc
	}
	return pcs
}

// Close 关闭所有PeerConnection
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 关闭所有PeerConnection
	for id, pc := range c.peerConnections {
		if pc != nil {
			pc.Close()
		}
		delete(c.peerConnections, id)
	}

	// 关闭音频系统
	if c.audioManager != nil {
		c.audioManager.Stop()
	}
}
