package webrtc

// ICECandidate 表示WebRTC ICE候选信息
type ICECandidate struct {
	Candidate     string `json:"candidate"`
	SDPMLineIndex uint16 `json:"sdpMLineIndex"`
	SDPMid        string `json:"sdpMid"`
}
