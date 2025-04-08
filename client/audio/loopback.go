package audio

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
)

// LoopbackSource 系统音频回环捕获
type LoopbackSource struct {
	config      AudioConfig
	buffer      []int16
	bufferSize  int
	readPos     int
	writePos    int
	bufferCount int
	mutex       sync.Mutex
	running     bool
	
	// 平台特定的实现
	platformImpl loopbackImpl
}

// loopbackImpl 平台特定的回环实现接口
type loopbackImpl interface {
	initialize(config AudioConfig) error
	start() error
	stop() error
	read(buffer []int16) (int, error)
	getDeviceList() ([]string, error)
}

// NewLoopbackSource 创建系统音频回环捕获源
func NewLoopbackSource(config AudioConfig) (AudioSource, error) {
	// 缓冲区大小设置为5帧
	bufferSize := config.FrameSize * config.Channels * 5
	
	source := &LoopbackSource{
		config:     config,
		buffer:     make([]int16, bufferSize),
		bufferSize: bufferSize,
	}
	
	// 根据平台创建具体实现
	var impl loopbackImpl
	var err error
	
	switch runtime.GOOS {
	case "windows":
		impl, err = newWindowsLoopback(config)
	case "darwin":
		impl, err = newMacOSLoopback(config)
	case "linux":
		impl, err = newLinuxLoopback(config)
	default:
		return nil, fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}
	
	if err != nil {
		return nil, err
	}
	
	source.platformImpl = impl
	return source, nil
}

// Start 开始捕获系统音频
func (s *LoopbackSource) Start() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	if s.running {
		return nil
	}
	
	if err := s.platformImpl.initialize(s.config); err != nil {
		return fmt.Errorf("初始化音频回环失败: %w", err)
	}
	
	if err := s.platformImpl.start(); err != nil {
		return fmt.Errorf("启动音频回环失败: %w", err)
	}
	
	s.running = true
	s.readPos = 0
	s.writePos = 0
	s.bufferCount = 0
	
	return nil
}

// Stop 停止捕获系统音频
func (s *LoopbackSource) Stop() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	if !s.running {
		return nil
	}
	
	if err := s.platformImpl.stop(); err != nil {
		return fmt.Errorf("停止音频回环失败: %w", err)
	}
	
	s.running = false
	return nil
}

// Read 读取系统音频数据
func (s *LoopbackSource) Read(buffer []int16) (int, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	if !s.running {
		return 0, errors.New("音频回环未启动")
	}
	
	return s.platformImpl.read(buffer)
}

// GetDeviceList 获取可用的回环设备列表
func (s *LoopbackSource) GetDeviceList() ([]string, error) {
	return s.platformImpl.getDeviceList()
}

// 以下是平台特定的实现声明，具体实现会在各自的文件中

// WindowsLoopback Windows WASAPI回环实现
type WindowsLoopback struct {
	// Windows特定实现
}

func newWindowsLoopback(config AudioConfig) (loopbackImpl, error) {
	return nil, errors.New("Windows回环捕获未实现")
}

// MacOSLoopback macOS Audio Unit回环实现
type MacOSLoopback struct {
	// macOS特定实现
}

func newMacOSLoopback(config AudioConfig) (loopbackImpl, error) {
	return nil, errors.New("macOS回环捕获未实现")
}

// LinuxLoopback Linux PulseAudio/JACK回环实现
type LinuxLoopback struct {
	// Linux特定实现
}

func newLinuxLoopback(config AudioConfig) (loopbackImpl, error) {
	return nil, errors.New("Linux回环捕获未实现")
} 