package models

// Message 表示WebRTC信令消息
type Message struct {
	Type       string      `json:"type"` // offer, answer, candidate, ping
	SDP        string      `json:"sdp,omitempty"`
	Candidate  string      `json:"candidate,omitempty"`
	SDPMLineIndex int       `json:"sdpMLineIndex,omitempty"`
	SDPMid      string     `json:"sdpMid,omitempty"`
	Data        interface{} `json:"data,omitempty"`    // 用于传输ping延迟等通用数据
}