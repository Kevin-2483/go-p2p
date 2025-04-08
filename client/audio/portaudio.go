package audio

import (
	"errors"
	"fmt"
	"sync"

	"github.com/charmbracelet/log"
	"github.com/gordonklaus/portaudio"
)

// 初始化portaudio
func init() {
	if err := portaudio.Initialize(); err != nil {
		log.Error("初始化PortAudio失败", "error", err)
	}
}

// PortAudioSource 基于PortAudio的音频源实现
type PortAudioSource struct {
	config      AudioConfig
	stream      *portaudio.Stream
	deviceInfo  *portaudio.DeviceInfo
	buffer      []int16
	bufferSize  int
	readPos     int
	writePos    int
	bufferCount int
	mutex       sync.Mutex
	running     bool
}

// NewAudioSource 创建新的音频源
func NewAudioSource(config AudioConfig) (AudioSource, error) {
	// 获取音频设备
	deviceInfo, err := getAudioDevice(config.InputDevice, true)
	if err != nil {
		return nil, err
	}

	// 缓冲区大小设置为5帧
	bufferSize := config.FrameSize * config.Channels * 5

	return &PortAudioSource{
		config:     config,
		deviceInfo: deviceInfo,
		buffer:     make([]int16, bufferSize),
		bufferSize: bufferSize,
	}, nil
}

// Start 开始音频捕获
func (s *PortAudioSource) Start() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.running {
		return nil
	}

	// 创建输入流参数
	inputParams := portaudio.StreamParameters{
		Input: portaudio.StreamDeviceParameters{
			Device:   s.deviceInfo,
			Channels: s.config.Channels,
			Latency:  s.deviceInfo.DefaultLowInputLatency,
		},
		SampleRate:      float64(s.config.SampleRate),
		FramesPerBuffer: s.config.FrameSize,
	}

	// 创建回调函数
	callback := func(in []int16, _ []int16) {
		s.mutex.Lock()
		defer s.mutex.Unlock()

		// 写入缓冲区
		for i := 0; i < len(in); i++ {
			s.buffer[s.writePos] = in[i]
			s.writePos = (s.writePos + 1) % s.bufferSize
			if s.writePos == s.readPos {
				// 缓冲区已满，移动读指针
				s.readPos = (s.readPos + 1) % s.bufferSize
			} else {
				s.bufferCount++
			}
		}
	}

	// 创建音频流
	stream, err := portaudio.OpenStream(inputParams, callback)
	if err != nil {
		return fmt.Errorf("打开音频输入流失败: %w", err)
	}

	// 启动流
	if err := stream.Start(); err != nil {
		stream.Close()
		return fmt.Errorf("启动音频输入流失败: %w", err)
	}

	s.stream = stream
	s.running = true
	s.readPos = 0
	s.writePos = 0
	s.bufferCount = 0

	log.Info("音频输入已启动", "device", s.deviceInfo.Name)
	return nil
}

// Stop 停止音频捕获
func (s *PortAudioSource) Stop() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.running {
		return nil
	}

	// 停止并关闭流
	if s.stream != nil {
		if err := s.stream.Stop(); err != nil {
			log.Error("停止音频输入流失败", "error", err)
		}
		if err := s.stream.Close(); err != nil {
			log.Error("关闭音频输入流失败", "error", err)
		}
		s.stream = nil
	}

	s.running = false
	log.Info("音频输入已停止")
	return nil
}

// Read 读取音频帧
func (s *PortAudioSource) Read(buffer []int16) (int, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.running {
		return 0, errors.New("音频输入未启动")
	}

	// 计算可读取的样本数
	available := s.bufferCount
	if available == 0 {
		return 0, nil
	}

	// 确保不超过缓冲区大小
	toRead := len(buffer)
	if toRead > available {
		toRead = available
	}

	// 读取数据
	for i := 0; i < toRead; i++ {
		buffer[i] = s.buffer[s.readPos]
		s.readPos = (s.readPos + 1) % s.bufferSize
		s.bufferCount--
	}

	return toRead / s.config.Channels, nil
}

// GetDeviceList 获取可用输入设备列表
func (s *PortAudioSource) GetDeviceList() ([]string, error) {
	return getAudioDeviceList(true)
}

// PortAudioSink 基于PortAudio的音频接收器实现
type PortAudioSink struct {
	config     AudioConfig
	stream     *portaudio.Stream
	deviceInfo *portaudio.DeviceInfo
	buffer     []int16
	bufferSize int
	readPos    int
	writePos   int
	bufferCount int
	mutex      sync.Mutex
	running    bool
}

// NewAudioSink 创建新的音频接收器
func NewAudioSink(config AudioConfig) (AudioSink, error) {
	// 获取音频设备
	deviceInfo, err := getAudioDevice(config.OutputDevice, false)
	if err != nil {
		return nil, err
	}

	// 缓冲区大小设置为5帧
	bufferSize := config.FrameSize * config.Channels * 5

	return &PortAudioSink{
		config:     config,
		deviceInfo: deviceInfo,
		buffer:     make([]int16, bufferSize),
		bufferSize: bufferSize,
	}, nil
}

// Start 开始音频播放
func (s *PortAudioSink) Start() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.running {
		return nil
	}

	// 创建输出流参数
	outputParams := portaudio.StreamParameters{
		Output: portaudio.StreamDeviceParameters{
			Device:   s.deviceInfo,
			Channels: s.config.Channels,
			Latency:  s.deviceInfo.DefaultLowOutputLatency,
		},
		SampleRate:      float64(s.config.SampleRate),
		FramesPerBuffer: s.config.FrameSize,
	}

	// 创建回调函数
	callback := func(_ []int16, out []int16) {
		s.mutex.Lock()
		defer s.mutex.Unlock()

		// 从缓冲区读取数据
		samplesToWrite := len(out)
		if s.bufferCount < samplesToWrite {
			// 缓冲区数据不足，用静音填充
			for i := 0; i < samplesToWrite; i++ {
				out[i] = 0
			}
			return
		}

		// 写出数据
		for i := 0; i < samplesToWrite; i++ {
			out[i] = s.buffer[s.readPos]
			s.readPos = (s.readPos + 1) % s.bufferSize
			s.bufferCount--
		}
	}

	// 创建音频流
	stream, err := portaudio.OpenStream(outputParams, callback)
	if err != nil {
		return fmt.Errorf("打开音频输出流失败: %w", err)
	}

	// 启动流
	if err := stream.Start(); err != nil {
		stream.Close()
		return fmt.Errorf("启动音频输出流失败: %w", err)
	}

	s.stream = stream
	s.running = true
	s.readPos = 0
	s.writePos = 0
	s.bufferCount = 0

	log.Info("音频输出已启动", "device", s.deviceInfo.Name)
	return nil
}

// Stop 停止音频播放
func (s *PortAudioSink) Stop() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.running {
		return nil
	}

	// 停止并关闭流
	if s.stream != nil {
		if err := s.stream.Stop(); err != nil {
			log.Error("停止音频输出流失败", "error", err)
		}
		if err := s.stream.Close(); err != nil {
			log.Error("关闭音频输出流失败", "error", err)
		}
		s.stream = nil
	}

	s.running = false
	log.Info("音频输出已停止")
	return nil
}

// Write 写入音频帧
func (s *PortAudioSink) Write(buffer []int16) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.running {
		return errors.New("音频输出未启动")
	}

	// 检查缓冲区是否有足够空间
	if len(buffer) > s.bufferSize-s.bufferCount {
		// 缓冲区已满，丢弃旧数据
		needed := len(buffer) - (s.bufferSize - s.bufferCount)
		s.readPos = (s.readPos + needed) % s.bufferSize
		s.bufferCount -= needed
	}

	// 写入数据
	for i := 0; i < len(buffer); i++ {
		s.buffer[s.writePos] = buffer[i]
		s.writePos = (s.writePos + 1) % s.bufferSize
		s.bufferCount++
	}

	return nil
}

// GetDeviceList 获取可用输出设备列表
func (s *PortAudioSink) GetDeviceList() ([]string, error) {
	return getAudioDeviceList(false)
}

// 获取可用音频设备列表
func getAudioDeviceList(isInput bool) ([]string, error) {
	devices, err := portaudio.Devices()
	if err != nil {
		return nil, fmt.Errorf("获取音频设备失败: %w", err)
	}

	var deviceList []string
	for _, device := range devices {
		if isInput && device.MaxInputChannels > 0 {
			deviceList = append(deviceList, device.Name)
		} else if !isInput && device.MaxOutputChannels > 0 {
			deviceList = append(deviceList, device.Name)
		}
	}

	return deviceList, nil
}

// 获取指定的音频设备
func getAudioDevice(deviceName string, isInput bool) (*portaudio.DeviceInfo, error) {
	// 获取所有设备
	devices, err := portaudio.Devices()
	if err != nil {
		return nil, fmt.Errorf("获取音频设备失败: %w", err)
	}

	// 如果未指定设备，使用默认设备
	if deviceName == "" {
		if isInput {
			return portaudio.DefaultInputDevice()
		}
		return portaudio.DefaultOutputDevice()
	}

	// 查找指定设备
	for _, device := range devices {
		if device.Name == deviceName {
			if isInput && device.MaxInputChannels > 0 {
				return device, nil
			} else if !isInput && device.MaxOutputChannels > 0 {
				return device, nil
			}
		}
	}

	return nil, fmt.Errorf("未找到音频设备: %s", deviceName)
}

// 程序退出时终止PortAudio
func Terminate() {
	if err := portaudio.Terminate(); err != nil {
		log.Error("终止PortAudio失败", "error", err)
	}
} 