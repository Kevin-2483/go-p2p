package models

// Message 表示WebRTC信令消息
type Message struct {
	Type       string `json:"type"` // offer, answer, candidate
	SDP        string `json:"sdp,omitempty"`
	Candidate  string `json:"candidate,omitempty"`
	SDPMLineIndex int    `json:"sdpMLineIndex,omitempty"`
	SDPMid      string `json:"sdpMid,omitempty"`
}