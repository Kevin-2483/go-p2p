package handlers

import (

	"github.com/pion/webrtc/v3"
	"github.com/charmbracelet/log"
	"server/models"
)

var peerConnectionConfig = webrtc.Configuration{
	ICEServers: []webrtc.ICEServer{
		{
			URLs: []string{"stun:stun.l.google.com:19302"},
		},
	},
}

// handleOffer 处理来自客户端的offer
func handleOffer(client *models.Client, msg *models.Message) {
	// 创建新的PeerConnection
	peerConnection, err := webrtc.NewPeerConnection(peerConnectionConfig)
	if err != nil {
		log.Error("Failed to create peer connection:", err)
		return
	}

	// 设置远程描述
	offer := webrtc.SessionDescription{}
	offer.Type = webrtc.SDPTypeOffer
	offer.SDP = msg.SDP

	if err := peerConnection.SetRemoteDescription(offer); err != nil {
		log.Error("Failed to set remote description:", err)
		return
	}

	// 创建answer
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		log.Error("Failed to create answer:", err)
		return
	}

	// 设置本地描述
	if err := peerConnection.SetLocalDescription(answer); err != nil {
		log.Error("Failed to set local description:", err)
		return
	}

	// 发送answer给客户端
	response := models.Message{
		Type: "answer",
		SDP:  answer.SDP,
	}

	if err := client.Conn.WriteJSON(response); err != nil {
		log.Error("Failed to send answer:", err)
		return
	}
}

// handleICECandidate 处理ICE候选者
func handleICECandidate(client *models.Client, msg *models.Message) {
	// 如果有目标客户端，则转发ICE候选
	if msg.TargetID != "" {
		clientsLock.RLock()
		targetClient, exists := clients[msg.TargetID]
		clientsLock.RUnlock()

		if exists && targetClient.Conn != nil {
			// 设置源客户端ID
			msg.SourceID = client.ID

			// 转发ICE候选给目标客户端
			if err := targetClient.Conn.WriteJSON(msg); err != nil {
				log.Error("转发ICE候选失败", "error", err)
			}
		}
	}
}