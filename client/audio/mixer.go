package audio

import (
	"errors"
	"sync"
)

// MixerSource 混合多个音频源的音频源
type MixerSource struct {
	config      AudioConfig
	sources     []AudioSource
	buffer      []int16
	mixBuffer   []int32 // 使用32位整数避免混合时溢出
	bufferSize  int
	mutex       sync.Mutex
	running     bool
}

// NewMixerSource 创建新的混合音频源
func NewMixerSource(config AudioConfig, sources []AudioSource) (AudioSource, error) {
	if len(sources) == 0 {
		return nil, errors.New("至少需要一个音频源")
	}
	
	// 创建缓冲区
	bufferSize := config.FrameSize * config.Channels
	
	return &MixerSource{
		config:     config,
		sources:    sources,
		buffer:     make([]int16, bufferSize),
		mixBuffer:  make([]int32, bufferSize),
		bufferSize: bufferSize,
	}, nil
}

// Start 启动所有音频源
func (m *MixerSource) Start() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if m.running {
		return nil
	}
	
	// 启动所有音频源
	for _, source := range m.sources {
		if err := source.Start(); err != nil {
			// 停止已启动的源
			for _, s := range m.sources {
				s.Stop()
			}
			return err
		}
	}
	
	m.running = true
	return nil
}

// Stop 停止所有音频源
func (m *MixerSource) Stop() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if !m.running {
		return nil
	}
	
	// 停止所有音频源
	for _, source := range m.sources {
		source.Stop()
	}
	
	m.running = false
	return nil
}

// Read 从所有音频源读取并混合
func (m *MixerSource) Read(buffer []int16) (int, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if !m.running {
		return 0, errors.New("混合器未启动")
	}
	
	// 清空混合缓冲区
	for i := range m.mixBuffer {
		m.mixBuffer[i] = 0
	}
	
	// 从所有源读取并混合
	maxSamples := 0
	for _, source := range m.sources {
		// 读取到临时缓冲区
		samples, err := source.Read(m.buffer)
		if err != nil {
			return 0, err
		}
		
		// 更新最大样本数
		if samples > maxSamples {
			maxSamples = samples
		}
		
		// 混合音频（简单相加，可能需要音量调整）
		totalSamples := samples * m.config.Channels
		for i := 0; i < totalSamples; i++ {
			m.mixBuffer[i] += int32(m.buffer[i])
		}
	}
	
	// 不要超出缓冲区大小
	if maxSamples * m.config.Channels > len(buffer) {
		maxSamples = len(buffer) / m.config.Channels
	}
	
	// 将32位混合结果限制在16位范围内并复制到输出缓冲区
	totalSamples := maxSamples * m.config.Channels
	for i := 0; i < totalSamples; i++ {
		// 限制在int16范围内 (-32768 到 32767)
		if m.mixBuffer[i] > 32767 {
			buffer[i] = 32767
		} else if m.mixBuffer[i] < -32768 {
			buffer[i] = -32768
		} else {
			buffer[i] = int16(m.mixBuffer[i])
		}
	}
	
	return maxSamples, nil
}

// GetDeviceList 返回所有源的设备列表
func (m *MixerSource) GetDeviceList() ([]string, error) {
	if len(m.sources) == 0 {
		return []string{}, nil
	}
	
	// 返回第一个源的设备列表
	return m.sources[0].GetDeviceList()
} 