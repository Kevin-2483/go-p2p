package websocket

import (
	"github.com/charmbracelet/log"
	"github.com/pion/webrtc/v3"
)

// sendICECandidates 发送收集到的ICE候选到服务器
func (c *Client) sendICECandidates(sourceID, targetID, spaceID string) {
	// 获取收集到的ICE候选
	iceCandidatesMu.Lock()
	candidates, ok := iceCandidates[targetID]
	if !ok || len(candidates) == 0 {
		iceCandidatesMu.Unlock()
		log.Warn("没有收集到ICE候选")
		return
	}

	// 转换为服务器需要的格式
	var iceCandidatesForServer []map[string]interface{}
	for _, candidate := range candidates {
		iceCandidatesForServer = append(iceCandidatesForServer, map[string]interface{}{
			"candidate":     candidate.Candidate,
			"sdpMLineIndex": candidate.SDPMLineIndex,
			"sdpMid":        candidate.SDPMid,
		})
	}

	// 清空已发送的候选
	iceCandidates[targetID] = []webrtc.ICECandidateInit{}
	iceCandidatesMu.Unlock()

	// 发送ICE候选到服务器
	c.writeMu.Lock()
	err := c.conn.WriteJSON(map[string]interface{}{
		"type":           "ice_candidates",
		"source_id":      sourceID,
		"target_id":      targetID,
		"space_id":       spaceID,
		"ice_candidates": iceCandidatesForServer,
	})
	c.writeMu.Unlock()

	if err != nil {
		log.Error("发送ICE候选失败", "error", err)
		return
	}

	log.Info("已发送ICE候选到服务器", "count", len(iceCandidatesForServer))
}

// handleOffer 处理收到的offer
func (c *Client) handleOffer(msg map[string]interface{}) {
	sourceID, _ := msg["source_id"].(string)
	targetID, _ := msg["target_id"].(string)
	spaceID, _ := msg["space_id"].(string)
	sdp, _ := msg["sdp"].(string)

	if sourceID == "" || targetID == "" || spaceID == "" || sdp == "" {
		log.Error("offer消息缺少必要参数")
		return
	}

	// 获取PeerConnection
	peerConnectionsMu.Lock()
	peerConnection, ok := peerConnections[sourceID]
	peerConnectionsMu.Unlock()

	if !ok || peerConnection == nil {
		log.Error("未找到对应的PeerConnection")
		return
	}

	// 设置远程描述
	offer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  sdp,
	}

	err := peerConnection.SetRemoteDescription(offer)
	if err != nil {
		log.Error("设置远程描述失败", "error", err)
		return
	}

	// 创建answer
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		log.Error("创建answer失败", "error", err)
		return
	}

	// 设置本地描述
	err = peerConnection.SetLocalDescription(answer)
	if err != nil {
		log.Error("设置本地描述失败", "error", err)
		return
	}

	// 发送answer到服务器
	c.writeMu.Lock()
	err = c.conn.WriteJSON(map[string]interface{}{
		"type":      "answer",
		"sdp":       answer.SDP,
		"source_id": targetID,
		"target_id": sourceID,
		"space_id":  spaceID,
	})
	c.writeMu.Unlock()

	if err != nil {
		log.Error("发送answer失败", "error", err)
		return
	}

	log.Info("已发送answer到服务器")
}

// handleAnswer 处理收到的answer
func (c *Client) handleAnswer(msg map[string]interface{}) {
	sourceID, _ := msg["source_id"].(string)
	targetID, _ := msg["target_id"].(string)
	sdp, _ := msg["sdp"].(string)

	if sourceID == "" || targetID == "" || sdp == "" {
		log.Error("answer消息缺少必要参数")
		return
	}

	// 获取PeerConnection
	peerConnectionsMu.Lock()
	peerConnection, ok := peerConnections[targetID]
	peerConnectionsMu.Unlock()

	if !ok || peerConnection == nil {
		log.Error("未找到对应的PeerConnection")
		return
	}

	// 设置远程描述
	answer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  sdp,
	}

	err := peerConnection.SetRemoteDescription(answer)
	if err != nil {
		log.Error("设置远程描述失败", "error", err)
		return
	}

	log.Info("已设置远程answer描述")
}

// handleICECandidates 处理收到的ICE候选列表
func (c *Client) handleICECandidates(msg map[string]interface{}) {
	sourceID, _ := msg["source_id"].(string)
	targetID, _ := msg["target_id"].(string)

	if sourceID == "" || targetID == "" {
		log.Error("ICE候选消息缺少必要参数")
		return
	}

	// 获取ICE候选列表
	iceCandidatesData, ok := msg["ice_candidates"].([]interface{})
	if !ok || len(iceCandidatesData) == 0 {
		log.Error("ICE候选列表为空或格式错误")
		return
	}

	// 获取PeerConnection
	peerConnectionsMu.Lock()
	peerConnection, ok := peerConnections[sourceID]
	peerConnectionsMu.Unlock()

	if !ok || peerConnection == nil {
		log.Error("未找到对应的PeerConnection")
		return
	}

	// 添加ICE候选
	for _, candidateData := range iceCandidatesData {
		if candidateMap, ok := candidateData.(map[string]interface{}); ok {
			candidate, _ := candidateMap["candidate"].(string)
			sdpMLineIndex, _ := candidateMap["sdpMLineIndex"].(float64)
			sdpMid, _ := candidateMap["sdpMid"].(string)

			if candidate == "" {
				continue
			}

			// 转换为指针类型
			sdpMLineIndexPtr := uint16(sdpMLineIndex)
			sdpMidPtr := sdpMid

			iceCandidate := webrtc.ICECandidateInit{
				Candidate:     candidate,
				SDPMLineIndex: &sdpMLineIndexPtr,
				SDPMid:        &sdpMidPtr,
			}

			err := peerConnection.AddICECandidate(iceCandidate)
			if err != nil {
				log.Error("添加ICE候选失败", "error", err)
				continue
			}
		}
	}

	log.Info("已添加ICE候选", "count", len(iceCandidatesData))
}
