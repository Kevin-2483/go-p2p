package websocket

import (
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/pion/webrtc/v3"
)

// 全局变量，用于存储WebRTC连接和ICE候选
var (
	peerConnections = make(map[string]*webrtc.PeerConnection)
	peerConnectionsMu sync.Mutex
	iceCandidates = make(map[string][]webrtc.ICECandidateInit)
	iceCandidatesMu sync.Mutex
)

// handleP2PConnect 处理P2P连接请求
func (c *Client) handleP2PConnect(msg map[string]interface{}) {
	// 获取连接信息
	sourceID, _ := msg["source_id"].(string)
	targetID, _ := msg["target_id"].(string)
	spaceID, _ := msg["space_id"].(string)
	
	// 获取STUN/TURN服务器配置
	var iceServers []webrtc.ICEServer
	
	// 添加默认STUN服务器
	iceServers = append(iceServers, webrtc.ICEServer{
		URLs: []string{"stun:stun.l.google.com:19302"},
	})
	
	// 从消息中获取TURN服务器配置
	if data, ok := msg["data"].(map[string]interface{}); ok {
		if turnServers, ok := data["turn_servers"].([]interface{}); ok {
			for _, server := range turnServers {
				if turnConfig, ok := server.(map[string]interface{}); ok {
					url, _ := turnConfig["url"].(string)
					username, _ := turnConfig["username"].(string)
					password, _ := turnConfig["password"].(string)
					
					if url != "" {
						iceServers = append(iceServers, webrtc.ICEServer{
							URLs:       []string{url},
							Username:   username,
							Credential: password,
						})
					}
				}
			}
		}
	}
	
	// 判断自己是源客户端还是目标客户端
	// 如果消息中的sourceID是空的，说明自己是发起连接的源客户端
	// 如果消息中的sourceID不为空，说明自己是接收连接的目标客户端
	isSource := sourceID == ""
	isTarget := sourceID != ""
	
	if isSource {
		// 使用WebSocket连接时服务器分配的客户端ID
		c.mu.RLock()
		sourceID = c.config.Client.ID
		c.mu.RUnlock()
		
		// 如果没有获取到ID，生成一个临时ID
		if sourceID == "" {
			sourceID = "client_" + time.Now().Format("20060102150405")
		}
		log.Info("作为源客户端开始P2P连接", "target_id", targetID, "source_id", sourceID)
		// 作为源客户端，主动发起连接
		c.initiateP2PConnection(sourceID, targetID, spaceID, iceServers)
	} else if isTarget {
		log.Info("作为目标客户端开始P2P连接", "source_id", sourceID)
		// 作为目标客户端，等待连接
		c.acceptP2PConnection(sourceID, targetID, spaceID, iceServers)
	} else {
		log.Warn("收到P2P连接请求，但无法确定角色", "source_id", sourceID, "target_id", targetID)
	}
}

// initiateP2PConnection 作为源客户端发起P2P连接
func (c *Client) initiateP2PConnection(sourceID, targetID, spaceID string, iceServers []webrtc.ICEServer) {
	log.Info("开始发起P2P连接", "target_id", targetID)
	
	// 创建WebRTC配置
	config := webrtc.Configuration{
		ICEServers: iceServers,
	}
	
	// 创建PeerConnection
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		log.Error("创建PeerConnection失败", "error", err)
		return
	}
	
	// 保存PeerConnection
	peerConnectionsMu.Lock()
	peerConnections[targetID] = peerConnection
	peerConnectionsMu.Unlock()
	
	// 监听ICE候选
	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}
		
		// 将ICE候选添加到缓存
		iceCandidatesMu.Lock()
		if _, ok := iceCandidates[targetID]; !ok {
			iceCandidates[targetID] = []webrtc.ICECandidateInit{}
		}
		iceCandidates[targetID] = append(iceCandidates[targetID], candidate.ToJSON())
		iceCandidatesMu.Unlock()
		
		log.Debug("收集到ICE候选", "candidate", candidate.ToJSON())
	})
	
	// 监听连接状态变化
	peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Info("连接状态变化", "state", state.String())
		
		switch state {
		case webrtc.PeerConnectionStateConnected:
			log.Info("P2P连接已建立")
		case webrtc.PeerConnectionStateFailed, webrtc.PeerConnectionStateClosed:
			log.Info("P2P连接已关闭或失败")
			// 清理资源
			peerConnectionsMu.Lock()
			delete(peerConnections, targetID)
			peerConnectionsMu.Unlock()
			
			iceCandidatesMu.Lock()
			delete(iceCandidates, targetID)
			iceCandidatesMu.Unlock()
		}
	})
	
	// 创建数据通道（可选）
	_, err = peerConnection.CreateDataChannel("data", nil)
	if err != nil {
		log.Error("创建数据通道失败", "error", err)
		return
	}
	
	// 创建offer
	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		log.Error("创建offer失败", "error", err)
		return
	}
	
	// 设置本地描述
	err = peerConnection.SetLocalDescription(offer)
	if err != nil {
		log.Error("设置本地描述失败", "error", err)
		return
	}
	
	// 通过WebSocket发送offer给服务器
	c.writeMu.Lock()
	err = c.conn.WriteJSON(map[string]interface{}{
		"type":      "offer",
		"sdp":       offer.SDP,
		"source_id":  sourceID,
		"target_id":  targetID,
		"space_id":   spaceID,
	})
	c.writeMu.Unlock()
	
	if err != nil {
		log.Error("发送offer失败", "error", err)
		return
	}
	
	// 启动一个goroutine，等待一段时间后收集并发送ICE候选
	go func() {
		// 等待ICE候选收集
		time.Sleep(2 * time.Second)
		
		// 发送收集到的ICE候选
		c.sendICECandidates(sourceID, targetID, spaceID)
	}()
	
	log.Info("P2P连接初始化完成，等待ICE候选收集")
}

// acceptP2PConnection 作为目标客户端接受P2P连接
func (c *Client) acceptP2PConnection(sourceID, targetID, spaceID string, iceServers []webrtc.ICEServer) {
	log.Info("准备接受P2P连接", "source_id", sourceID)
	
	// 创建WebRTC配置
	config := webrtc.Configuration{
		ICEServers: iceServers,
	}
	
	// 创建PeerConnection
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		log.Error("创建PeerConnection失败", "error", err)
		return
	}
	
	// 保存PeerConnection
	peerConnectionsMu.Lock()
	peerConnections[sourceID] = peerConnection
	peerConnectionsMu.Unlock()
	
	// 监听ICE候选
	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}
		
		// 将ICE候选添加到缓存
		iceCandidatesMu.Lock()
		if _, ok := iceCandidates[sourceID]; !ok {
			iceCandidates[sourceID] = []webrtc.ICECandidateInit{}
		}
		iceCandidates[sourceID] = append(iceCandidates[sourceID], candidate.ToJSON())
		iceCandidatesMu.Unlock()
		
		log.Debug("收集到ICE候选", "candidate", candidate.ToJSON())
	})
	
	// 监听连接状态变化
	peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Info("连接状态变化", "state", state.String())
		
		switch state {
		case webrtc.PeerConnectionStateConnected:
			log.Info("P2P连接已建立")
		case webrtc.PeerConnectionStateFailed, webrtc.PeerConnectionStateClosed:
			log.Info("P2P连接已关闭或失败")
			// 清理资源
			peerConnectionsMu.Lock()
			delete(peerConnections, sourceID)
			peerConnectionsMu.Unlock()
			
			iceCandidatesMu.Lock()
			delete(iceCandidates, sourceID)
			iceCandidatesMu.Unlock()
		}
	})
	
	// 监听数据通道
	peerConnection.OnDataChannel(func(dc *webrtc.DataChannel) {
		log.Info("收到数据通道", "label", dc.Label())
		
		dc.OnOpen(func() {
			log.Info("数据通道已打开")
		})
		
		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			log.Info("收到数据通道消息", "length", len(msg.Data))
		})
	})
	
	// 启动一个goroutine，等待一段时间后收集并发送ICE候选
	go func() {
		// 等待ICE候选收集
		time.Sleep(2 * time.Second)
		
		// 发送收集到的ICE候选
		c.sendICECandidates(targetID, sourceID, spaceID)
	}()
	
	log.Info("P2P连接准备就绪，等待接收offer")
}