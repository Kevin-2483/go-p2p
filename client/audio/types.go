package audio

import (
	"github.com/pion/webrtc/v3"
)

// AudioPacket 表示一个编码后的音频包
type AudioPacket struct {
	Data       []byte `json:"data"`       // 编码后的音频数据
	SampleRate int    `json:"sampleRate"` // 采样率
	Channels   int    `json:"channels"`   // 通道数
	Timestamp  int64  `json:"timestamp"`  // 时间戳
}

// AudioConfig 音频配置
type AudioConfig struct {
	Enabled       bool   // 是否启用音频
	InputDevice   string // 输入设备
	OutputDevice  string // 输出设备
	SampleRate    int    // 采样率
	Channels      int    // 通道数
	FrameSize     int    // 帧大小
	BitrateKbps   int    // 比特率(kbps)
	OpusComplexity int    // Opus编码复杂度
}

// AudioManager 音频管理接口
type AudioManager interface {
	Start() error
	Stop() error
	AddTrack(pc *webrtc.PeerConnection) (*webrtc.TrackLocalStaticSample, error)
	OnTrack(track *webrtc.TrackRemote)
} 