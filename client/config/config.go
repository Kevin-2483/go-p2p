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
		Path          string
		PingInterval  int `toml:"ping_interval"`
		ReconnectDelay int `toml:"reconnect_delay"`
	}
}

// LoadConfig 从文件加载配置
func LoadConfig(configPath string) (*Config, error) {
	var config Config
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		return nil, err
	}
	config.WebSocket.PingInterval = 0;
	return &config, nil
}