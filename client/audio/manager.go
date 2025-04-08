package audio

import (
	"client/config"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"gopkg.in/hraban/opus.v2"
)

const (
	// 音频常量
	defaultSampleRate    = 48000
	defaultChannels      = 2
	defaultFrameSize     = 960 // 20ms at 48kHz
	defaultBitrateKbps   = 64  // 64 kbps
	defaultOpusComplexity = 10

	// OPUS相关常量
	maxFrameSize = 48000 * 60 / 1000 // 60ms at 48kHz
	
	// Opus应用类型
	opusAppVoip = 2048 // OPUS_APPLICATION_VOIP
)

// Manager 音频管理器
type Manager struct {
	config      AudioConfig
	encoder     *opus.Encoder
	decoder     *opus.Decoder
	audioSource AudioSource
	audioSink   AudioSink
	tracks      map[string]*webrtc.TrackLocalStaticSample
	stopChan    chan struct{}
	mu          sync.RWMutex
	running     bool
}

// NewManager 创建新的音频管理器
func NewManager(cfg *config.Config) (*Manager, error) {
	// 从配置中加载音频配置
	audioConfig := AudioConfig{
		Enabled:       cfg.Audio.Enabled,
		InputDevice:   cfg.Audio.InputDevice,
		OutputDevice:  cfg.Audio.OutputDevice,
		SampleRate:    cfg.Audio.SampleRate,
		Channels:      cfg.Audio.Channels,
		FrameSize:     cfg.Audio.FrameSize,
		BitrateKbps:   cfg.Audio.BitrateKbps,
		OpusComplexity: cfg.Audio.OpusComplexity,
	}

	// 设置默认值
	if audioConfig.SampleRate == 0 {
		audioConfig.SampleRate = defaultSampleRate
	}
	if audioConfig.Channels == 0 {
		audioConfig.Channels = defaultChannels
	}
	if audioConfig.FrameSize == 0 {
		audioConfig.FrameSize = defaultFrameSize
	}
	if audioConfig.BitrateKbps == 0 {
		audioConfig.BitrateKbps = defaultBitrateKbps
	}
	if audioConfig.OpusComplexity == 0 {
		audioConfig.OpusComplexity = defaultOpusComplexity
	}

	// 创建音频接收器
	audioSink, err := NewAudioSink(audioConfig)
	if err != nil {
		return nil, fmt.Errorf("创建音频接收器失败: %w", err)
	}

	// 创建适当的音频源
	var audioSource AudioSource
	
	// 判断是否需要捕获系统音频
	if cfg.Audio.CaptureSystem {
		log.Info("配置为捕获系统音频")
		
		if cfg.Audio.MixWithMic {
			log.Info("将系统音频与麦克风混合")
			
			// 创建麦克风音频源
			micSource, err := NewAudioSource(audioConfig)
			if err != nil {
				return nil, fmt.Errorf("创建麦克风音频源失败: %w", err)
			}
			
			// 创建系统音频源
			sysSource, err := NewLoopbackSource(audioConfig)
			if err != nil {
				log.Warn("创建系统音频源失败，只使用麦克风", "error", err)
				audioSource = micSource
			} else {
				// 创建混合音频源
				audioSource, err = NewMixerSource(audioConfig, []AudioSource{micSource, sysSource})
				if err != nil {
					return nil, fmt.Errorf("创建混合音频源失败: %w", err)
				}
			}
		} else {
			// 只捕获系统音频
			audioSource, err = NewLoopbackSource(audioConfig)
			if err != nil {
				return nil, fmt.Errorf("创建系统音频源失败: %w", err)
			}
		}
	} else {
		// 创建标准麦克风音频源
		audioSource, err = NewAudioSource(audioConfig)
		if err != nil {
			return nil, fmt.Errorf("创建音频源失败: %w", err)
		}
	}

	// 创建Opus编码器
	encoder, err := opus.NewEncoder(audioConfig.SampleRate, audioConfig.Channels, opusAppVoip)
	if err != nil {
		return nil, fmt.Errorf("创建Opus编码器失败: %w", err)
	}

	// 设置编码器参数
	encoder.SetBitrate(1000 * audioConfig.BitrateKbps)
	encoder.SetComplexity(audioConfig.OpusComplexity)

	// 创建Opus解码器
	decoder, err := opus.NewDecoder(audioConfig.SampleRate, audioConfig.Channels)
	if err != nil {
		return nil, fmt.Errorf("创建Opus解码器失败: %w", err)
	}

	return &Manager{
		config:      audioConfig,
		encoder:     encoder,
		decoder:     decoder,
		audioSource: audioSource,
		audioSink:   audioSink,
		tracks:      make(map[string]*webrtc.TrackLocalStaticSample),
		stopChan:    make(chan struct{}),
	}, nil
}

// Start 开始音频处理
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.config.Enabled {
		return errors.New("音频功能未启用")
	}

	if m.running {
		return nil // 已经运行中
	}

	// 初始化音频源
	if err := m.audioSource.Start(); err != nil {
		return fmt.Errorf("启动音频源失败: %w", err)
	}

	// 初始化音频接收器
	if err := m.audioSink.Start(); err != nil {
		m.audioSource.Stop()
		return fmt.Errorf("启动音频接收器失败: %w", err)
	}

	m.running = true
	go m.captureAndEncodeLoop()

	log.Info("音频系统已启动", "sampleRate", m.config.SampleRate, "channels", m.config.Channels)
	return nil
}

// Stop 停止音频处理
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return nil
	}

	close(m.stopChan)
	
	// 停止音频源和接收器
	m.audioSource.Stop()
	m.audioSink.Stop()

	// 重置状态
	m.running = false
	m.stopChan = make(chan struct{})

	log.Info("音频系统已停止")
	return nil
}

// AddTrack 添加音频轨道到WebRTC PeerConnection
func (m *Manager) AddTrack(pc *webrtc.PeerConnection) (*webrtc.TrackLocalStaticSample, error) {
	if !m.config.Enabled {
		return nil, errors.New("音频功能未启用")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 创建音频轨道
	audioTrack, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{
			MimeType:    webrtc.MimeTypeOpus,
			ClockRate:   uint32(m.config.SampleRate),
			Channels:    uint16(m.config.Channels),
			SDPFmtpLine: "minptime=10;useinbandfec=1",
		},
		"audio",
		"mediadevices",
	)
	if err != nil {
		return nil, fmt.Errorf("创建音频轨道失败: %w", err)
	}

	// 添加轨道到PeerConnection
	rtpSender, err := pc.AddTrack(audioTrack)
	if err != nil {
		return nil, fmt.Errorf("添加音频轨道失败: %w", err)
	}

	// 处理RTCP包
	go func() {
		rtcpBuf := make([]byte, 1500)
		for {
			if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
				if rtcpErr == io.EOF {
					return
				}
				log.Error("读取RTCP包失败", "error", rtcpErr)
			}
		}
	}()

	// 保存轨道
	peerID := pc.ConnectionState().String() // 使用连接状态字符串作为ID
	m.tracks[peerID] = audioTrack

	log.Info("添加音频轨道", "peerID", peerID)
	return audioTrack, nil
}

// OnTrack 处理接收到的音频轨道
func (m *Manager) OnTrack(track *webrtc.TrackRemote) {
	if !m.config.Enabled {
		return
	}

	// 仅处理Opus音频
	if track.Codec().MimeType != webrtc.MimeTypeOpus {
		return
	}

	log.Info("收到音频轨道", "trackID", track.ID(), "mimetype", track.Codec().MimeType)

	// 创建缓冲区接收RTP包
	buf := make([]byte, maxFrameSize*2) // 足够大的缓冲区
	pcmBuf := make([]int16, m.config.FrameSize*m.config.Channels)

	go func() {
		for {
			// 读取RTP包
			n, _, readErr := track.Read(buf)
			if readErr != nil {
				if readErr == io.EOF {
					return
				}
				log.Error("读取RTP包失败", "error", readErr)
				continue
			}

			if n == 0 {
				continue
			}

			// 解码Opus数据
			samplesDecoded, decodeErr := m.decoder.Decode(buf[:n], pcmBuf)
			if decodeErr != nil {
				log.Error("解码音频失败", "error", decodeErr)
				continue
			}

			if samplesDecoded == 0 {
				continue
			}

			// 播放解码后的PCM数据
			m.audioSink.Write(pcmBuf[:samplesDecoded*m.config.Channels])
		}
	}()
}

// captureAndEncodeLoop 捕获和编码音频循环
func (m *Manager) captureAndEncodeLoop() {
	// 创建PCM和编码后的缓冲区
	pcmBuf := make([]int16, m.config.FrameSize*m.config.Channels)
	encodedBuf := make([]byte, maxFrameSize*2)

	for {
		select {
		case <-m.stopChan:
			return
		default:
			// 读取音频数据
			samplesRead, err := m.audioSource.Read(pcmBuf)
			if err != nil {
				log.Error("读取音频数据失败", "error", err)
				time.Sleep(10 * time.Millisecond)
				continue
			}

			if samplesRead == 0 {
				time.Sleep(10 * time.Millisecond)
				continue
			}

			// 编码音频数据
			n, err := m.encoder.Encode(pcmBuf[:samplesRead*m.config.Channels], encodedBuf)
			if err != nil {
				log.Error("编码音频数据失败", "error", err)
				continue
			}

			if n == 0 {
				continue
			}

			// 创建音频包
			packet := &AudioPacket{
				Data:       encodedBuf[:n],
				SampleRate: m.config.SampleRate,
				Channels:   m.config.Channels,
				Timestamp:  time.Now().UnixNano(),
			}

			// 计算样本持续时间
			sampleDuration := time.Duration(samplesRead) * time.Second / time.Duration(m.config.SampleRate)

			// 将编码后的数据发送到所有轨道
			m.mu.RLock()
			for _, track := range m.tracks {
				if err := track.WriteSample(media.Sample{
					Data:     packet.Data,
					Duration: sampleDuration,
				}); err != nil {
					log.Error("写入音频样本失败", "error", err)
				}
			}
			m.mu.RUnlock()

			// 帧间延迟，避免CPU占用过高
			frameDuration := time.Duration(1000*m.config.FrameSize/m.config.SampleRate) * time.Millisecond
			time.Sleep(frameDuration / 2) // 减少一半等待时间，确保不会跳帧
		}
	}
} 