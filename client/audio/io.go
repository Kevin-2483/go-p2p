package audio

// AudioSource 音频源接口
type AudioSource interface {
	// Start 开始音频捕获
	Start() error
	// Stop 停止音频捕获
	Stop() error
	// Read 读取音频帧，返回采样数
	Read(buffer []int16) (int, error)
	// GetDeviceList 获取可用设备列表
	GetDeviceList() ([]string, error)
}

// AudioSink 音频接收器接口
type AudioSink interface {
	// Start 开始音频播放
	Start() error
	// Stop 停止音频播放
	Stop() error
	// Write 写入音频帧，buffer长度应为采样数*通道数
	Write(buffer []int16) error
	// GetDeviceList 获取可用设备列表
	GetDeviceList() ([]string, error)
} 