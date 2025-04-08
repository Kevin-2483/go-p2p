package config

import (
	"github.com/BurntSushi/toml"
)

// Config 配置结构
type Config struct {
	Server struct {
		Host string
		Port int
	}
	WebSocket struct {
		Path           string
		PingInterval   int `toml:"ping_interval"`
		ReconnectDelay int `toml:"reconnect_delay"`
	}
	Client struct {
		ID         string `toml:"id"`
		PublicKey  string `toml:"public_key"`
		PrivateKey string `toml:"private_key"`
	}
	Audio struct {
		Enabled        bool   `toml:"enabled"`
		InputDevice    string `toml:"input_device"`
		OutputDevice   string `toml:"output_device"`
		CaptureSystem  bool   `toml:"capture_system"`  // 是否捕获系统音频
		MixWithMic     bool   `toml:"mix_with_mic"`    // 是否将系统音频与麦克风混合
		SampleRate     int    `toml:"sample_rate"`
		Channels       int    `toml:"channels"`
		FrameSize      int    `toml:"frame_size"`
		BitrateKbps    int    `toml:"bitrate_kbps"`
		OpusComplexity int    `toml:"opus_complexity"`
	}
}

// LoadConfig 从文件加载配置
func LoadConfig(configPath string) (*Config, error) {
	var config Config
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		return nil, err
	}
	return &config, nil
}
