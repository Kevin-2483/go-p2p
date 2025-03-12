package handlers

import (
	"log"

	"github.com/pion/webrtc/v3"
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
		log.Println("Failed to create peer connection:", err)
		return
	}

	// 设置远程描述
	offer := webrtc.SessionDescription{}
	offer.Type = webrtc.SDPTypeOffer
	offer.SDP = msg.SDP

	if err := peerConnection.SetRemoteDescription(offer); err != nil {
		log.Println("Failed to set remote description:", err)
		return
	}

	// 创建answer
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		log.Println("Failed to create answer:", err)
		return
	}

	// 设置本地描述
	if err := peerConnection.SetLocalDescription(answer); err != nil {
		log.Println("Failed to set local description:", err)
		return
	}

	// 发送answer给客户端
	response := models.Message{
		Type: "answer",
		SDP:  answer.SDP,
	}

	if err := client.Conn.WriteJSON(response); err != nil {
		log.Println("Failed to send answer:", err)
		return
	}
}

// handleICECandidate 处理ICE候选者
func handleICECandidate(client *models.Client, msg *models.Message) {
	// TODO: 处理ICE候选者
	// 在实际应用中，这里需要将ICE候选者添加到对应的PeerConnection中
}