package models

// Message 表示WebRTC信令消息
type Message struct {
	Type          string         `json:"type"` // offer, answer, candidate, ping, connect, ice_candidates
	SDP           string         `json:"sdp,omitempty"`
	Candidate     string         `json:"candidate,omitempty"`
	SDPMLineIndex int            `json:"sdpMLineIndex,omitempty"`
	SDPMid        string         `json:"sdpMid,omitempty"`
	Data          interface{}    `json:"data,omitempty"`           // 用于传输ping延迟等通用数据
	TargetID      string         `json:"target_id,omitempty"`      // 目标客户端ID
	SourceID      string         `json:"source_id,omitempty"`      // 源客户端ID
	SpaceID       string         `json:"space_id,omitempty"`       // 空间ID
	ICECandidates []ICECandidate `json:"ice_candidates,omitempty"` // ICE候选列表
	FromClientID  string         `json:"from_client_id,omitempty"` // 发送者客户端ID
}

// ICECandidate 表示WebRTC ICE候选信息
type ICECandidate struct {
	Candidate     string `json:"candidate"`
	SDPMLineIndex uint16 `json:"sdpMLineIndex"`
	SDPMid        string `json:"sdpMid"`
}
