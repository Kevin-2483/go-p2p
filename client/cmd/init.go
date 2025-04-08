package cmd

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/BurntSushi/toml"
	"github.com/spf13/cobra"
)

// Config 配置结构体
type Config struct {
	Server struct {
		Host string
		Port int
	} `toml:"server"`
	WebSocket struct {
		Path           string `toml:"path"`
		PingInterval   int    `toml:"ping_interval"`
		ReconnectDelay int    `toml:"reconnect_delay"`
	} `toml:"websocket"`
	Client struct {
		ID         string `toml:"id"`
		PrivateKey string `toml:"private_key"`
		PublicKey  string `toml:"public_key"`
	} `toml:"client"`
}

// initCmd 表示init命令
var initCmd = &cobra.Command{
	Use:   "init [web_api_key]",
	Short: "初始化客户端配置",
	Long:  `使用web api key初始化客户端配置，包括生成密钥对和配置文件。`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			// 生成默认配置
			generateDefaultConfig()
			return
		}

		// 使用web api key初始化
		webAPIKey := args[0]
		fmt.Printf("开始使用WebAPIKey初始化客户端：%s\n", webAPIKey)
		if err := initializeWithWebAPIKey(webAPIKey); err != nil {
			fmt.Printf("初始化失败：%v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

// generateDefaultConfig 生成默认配置文件
func generateDefaultConfig() {
	// 默认配置文件内容
	configContent := `# WebSocket服务器配置
[server]
host = "localhost"  # 服务器主机名或IP地址
port = 8080        # 服务器端口号

# WebSocket连接配置
[websocket]
path = "/ws/client"  # WebSocket连接路径
ping_interval = 3     # 心跳检测间隔（秒）
reconnect_delay = 5   # 重连延迟（秒）

# 音频配置
[audio]
enabled = true
input_device = ""  # 留空使用系统默认设备
output_device = "" # 留空使用系统默认设备
capture_system = false        # 是否捕获系统音频输出
mix_with_mic = false          # 是否将系统音频与麦克风混合
sample_rate = 48000           # 采样率(Hz)
channels = 1                  # 通道数，1=单声道，2=立体声
frame_size = 960              # 帧大小，20ms@48kHz=960
bitrate_kbps = 64             # 比特率(kbps)
opus_complexity = 10          # Opus编码复杂度(0-10)
`

	// 检查文件是否已存在
	if _, err := os.Stat("config.toml"); err == nil {
		fmt.Println("配置文件config.toml已存在，跳过生成")
		return
	}

	// 写入配置文件
	if err := os.WriteFile("config.toml", []byte(configContent), 0644); err != nil {
		fmt.Printf("生成配置文件失败：%v\n", err)
		os.Exit(1)
	}

	fmt.Println("配置文件已生成：config.toml")
}

// initializeWithWebAPIKey 使用web api key初始化客户端
func initializeWithWebAPIKey(webAPIKey string) error {
	// 生成密钥对
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Printf("生成RSA密钥对失败：%v\n", err)
		os.Exit(1)
	}
	pubKey := &privKey.PublicKey
	if err != nil {
		fmt.Printf("生成密钥对失败：%v\n", err)
		os.Exit(1)
	}

	// 编码密钥
	privKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privKey),
	})
	pubKeyDER, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		fmt.Printf("编码公钥失败：%v\n", err)
		os.Exit(1)
	}
	pubKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyDER,
	})
	privKeyStr := string(privKeyPEM)
	pubKeyStr := string(pubKeyPEM)

	// 获取WebAPIKey信息
	parsedURL, err := url.Parse(webAPIKey)
	if err != nil {
		return fmt.Errorf("解析web_api_key失败：%v", err)
	}

	// 添加公钥查询参数
	query := parsedURL.Query()
	query.Add("publickey", pubKeyStr)
	parsedURL.RawQuery = query.Encode()

	resp, err := http.Get(parsedURL.String())
	if err != nil {
		return fmt.Errorf("获取WebAPIKey信息失败：%v", err)
	}
	defer resp.Body.Close()

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("服务器返回错误状态码：%d 响应内容：%s", resp.StatusCode, body)
	}

	// 检查响应内容类型
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("意外的内容类型：%s 响应内容：%s", contentType, body)
	}

	// 记录原始响应体用于调试
	rawBody, _ := io.ReadAll(resp.Body)
	fmt.Printf("原始响应内容：%s\n", rawBody)

	// 重新填充响应体以便解码
	resp.Body = io.NopCloser(bytes.NewReader(rawBody))

	// 解析WebAPIKey响应
	var apiKeyResp WebAPIKeyResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiKeyResp); err != nil {
		return fmt.Errorf("解析WebAPIKey响应失败：%v 原始内容：%s", err, rawBody)
	}

	// 创建配置
	config := Config{}
	// 从解析的URL获取服务器配置
	config.Server.Host = parsedURL.Hostname()
	portStr := parsedURL.Port()
	if portStr == "" {
		if parsedURL.Scheme == "https" {
			portStr = "443"
		} else {
			portStr = "80"
		}
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("无效的端口号：%s", portStr)
	}
	config.Server.Port = port

	// 从API响应获取websocket配置
	websocketConfig := apiKeyResp.Data.WebSocket
	config.WebSocket.Path = websocketConfig.Path
	config.WebSocket.PingInterval = websocketConfig.PingInterval
	config.WebSocket.ReconnectDelay = websocketConfig.ReconnectDelay

	config.Client.ID = apiKeyResp.Data.ClientID
	config.Client.PrivateKey = privKeyStr
	config.Client.PublicKey = pubKeyStr

	// 编码配置
	buf := new(bytes.Buffer)
	if err := toml.NewEncoder(buf).Encode(config); err != nil {
		fmt.Printf("编码配置失败：%v\n", err)
		os.Exit(1)
	}

	// 写入配置文件
	configPath := "config.toml"
	if err := os.WriteFile(configPath, buf.Bytes(), 0644); err != nil {
		fmt.Printf("写入配置文件失败：%v\n", err)
		os.Exit(1)
	}

	fmt.Printf("客户端初始化成功，配置文件已保存到：%s\n", configPath)
	return nil // 添加返回nil表示成功
}

// WebAPIKeyResponse 服务器返回的WebAPIKey信息
type WebAPIKeyResponse struct {
	Status string `json:"status"`
	Data   struct {
		Server struct {
			Host string `json:"host"`
			Port int    `json:"port"`
		} `json:"server"`
		WebSocket struct {
			Path           string `json:"path"`
			PingInterval   int    `json:"ping_interval"`
			ReconnectDelay int    `json:"reconnect_delay"`
		} `json:"websocket"`
		ClientID string `json:"client_id"`
	} `json:"data"`
}
