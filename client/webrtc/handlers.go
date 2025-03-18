package webrtc

import (
	"encoding/json"

	"github.com/charmbracelet/log"
	"github.com/pion/webrtc/v3"
)

// MessageHandler WebRTC消息处理器
type MessageHandler struct {
	client *Client
}

// NewMessageHandler 创建新的WebRTC消息处理器
func NewMessageHandler(client *Client) *MessageHandler {
	return &MessageHandler{
		client: client,
	}
}

// HandleConnect 处理连接请求消息
func (h *MessageHandler) HandleConnect(msg map[string]interface{}) {
	// 提取消息中的源客户端ID和目标客户端ID
	sourceID, _ := msg["source_id"].(string)
	targetID, _ := msg["target_id"].(string)
	spaceID, _ := msg["space_id"].(string)

	if sourceID == "" || targetID == "" || spaceID == "" {
		log.Error("连接请求参数不完整")
		return
	}

	log.Info("收到连接请求", "source_id", sourceID, "target_id", targetID, "space_id", spaceID)

	// 确定本地客户端是发起方还是接收方
	isInitiator := h.client.config.Client.ID == sourceID

	if isInitiator {
		// 作为发起方，创建offer
		log.Info("作为发起方创建连接", "target_id", targetID)
		h.createOffer(targetID)
	} else {
		// 作为接收方，等待offer
		log.Info("作为接收方等待连接", "source_id", sourceID)
	}
}

// HandleOffer 处理offer消息
func (h *MessageHandler) HandleOffer(msg map[string]interface{}) {
	// 提取消息中的源客户端ID和SDP
	sourceID, _ := msg["source_id"].(string)
	sdp, _ := msg["sdp"].(string)

	if sourceID == "" || sdp == "" {
		log.Error("offer参数不完整")
		return
	}

	log.Info("收到offer", "source_id", sourceID)

	// 创建或获取PeerConnection
	pc, err := h.client.GetPeerConnection(sourceID)
	if err != nil {
		log.Error("创建PeerConnection失败", "error", err)
		return
	}

	// 设置远程描述
	offer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  sdp,
	}

	if err := pc.SetRemoteDescription(offer); err != nil {
		log.Error("设置远程描述失败", "error", err)
		return
	}

	// 创建answer
	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		log.Error("创建answer失败", "error", err)
		return
	}

	// 设置本地描述
	if err := pc.SetLocalDescription(answer); err != nil {
		log.Error("设置本地描述失败", "error", err)
		return
	}

	// 发送answer
	answerMsg := map[string]interface{}{
		"type":      "answer",
		"target_id": sourceID,
		"source_id": h.client.config.Client.ID,
		"sdp":       answer.SDP,
	}

	// 通过WebSocket发送answer
	h.client.SendJSON(answerMsg)
	log.Info("发送answer", "target_id", sourceID)
}

// HandleAnswer 处理answer消息
func (h *MessageHandler) HandleAnswer(msg map[string]interface{}) {
	// 提取消息中的源客户端ID和SDP
	sourceID, _ := msg["source_id"].(string)
	sdp, _ := msg["sdp"].(string)

	if sourceID == "" || sdp == "" {
		log.Error("answer参数不完整")
		return
	}

	log.Info("收到answer", "source_id", sourceID)

	// 获取PeerConnection
	pc, err := h.client.GetPeerConnection(sourceID)
	if err != nil {
		log.Error("获取PeerConnection失败", "error", err)
		return
	}

	// 设置远程描述
	answer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  sdp,
	}

	if err := pc.SetRemoteDescription(answer); err != nil {
		log.Error("设置远程描述失败", "error", err)
		return
	}

	log.Info("已设置远程描述", "target_id", sourceID)
}

// HandleICECandidates 处理ICE候选消息
func (h *MessageHandler) HandleICECandidates(msg map[string]interface{}) {
	// 提取消息中的源客户端ID和ICE候选列表
	sourceID, _ := msg["source_id"].(string)
	targetID, _ := msg["target_id"].(string)

	// 获取本地客户端ID
	localClientID := h.client.config.Client.ID

	// 确定对方ID
	peerID := ""
	if localClientID == targetID {
		// 如果本地客户端是目标，那么对方是源
		peerID = sourceID
	} else if localClientID == sourceID {
		// 如果本地客户端是源，那么对方是目标
		peerID = targetID
	} else {
		log.Error("无法确定ICE候选消息的来源", "local_id", localClientID, "source_id", sourceID, "target_id", targetID)
		return
	}

	// 获取ICE候选列表
	iceCandidatesData, ok := msg["ice_candidates"]
	if !ok {
		log.Error("ICE候选列表为空")
		return
	}

	// 将ICE候选列表转换为JSON
	iceCandidatesJSON, err := json.Marshal(iceCandidatesData)
	if err != nil {
		log.Error("ICE候选列表序列化失败", "error", err)
		return
	}

	// 解析ICE候选列表
	var iceCandidates []ICECandidate
	if err := json.Unmarshal(iceCandidatesJSON, &iceCandidates); err != nil {
		log.Error("ICE候选列表反序列化失败", "error", err)
		return
	}

	if len(iceCandidates) == 0 {
		log.Error("ICE候选列表为空")
		return
	}

	log.Info("收到ICE候选", "peer_id", peerID, "candidates_count", len(iceCandidates))

	// 获取PeerConnection
	pc, err := h.client.GetPeerConnection(peerID)
	if err != nil {
		log.Error("获取PeerConnection失败", "error", err)
		return
	}

	// 添加ICE候选
	for _, candidate := range iceCandidates {
		if err := pc.AddICECandidate(webrtc.ICECandidateInit{
			Candidate:     candidate.Candidate,
			SDPMLineIndex: &candidate.SDPMLineIndex,
			SDPMid:        &candidate.SDPMid,
		}); err != nil {
			log.Error("添加ICE候选失败", "error", err)
		}
	}
}

// createOffer 创建并发送offer
func (h *MessageHandler) createOffer(targetID string) {
	// 创建或获取PeerConnection
	pc, err := h.client.GetPeerConnection(targetID)
	if err != nil {
		log.Error("创建PeerConnection失败", "error", err)
		return
	}

	// 创建offer
	offer, err := pc.CreateOffer(nil)
	if err != nil {
		log.Error("创建offer失败", "error", err)
		return
	}

	// 设置本地描述
	if err := pc.SetLocalDescription(offer); err != nil {
		log.Error("设置本地描述失败", "error", err)
		return
	}

	// 发送offer
	offerMsg := map[string]interface{}{
		"type":      "offer",
		"target_id": targetID,
		"source_id": h.client.config.Client.ID,
		"sdp":       offer.SDP,
	}

	// 通过WebSocket发送offer
	h.client.SendJSON(offerMsg)
	log.Info("发送offer", "target_id", targetID)
}
