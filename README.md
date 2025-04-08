# WebRTC音频通信系统

这是一个基于WebRTC的P2P音频通信系统，支持在客户端之间建立直接连接并传输音频数据。

## 功能特点

- 基于WebRTC的P2P直接连接
- 支持高质量Opus音频编码
- 跨平台支持（Windows、macOS、Linux）
- 可配置的音频参数（采样率、通道数、比特率等）
- 灵活的设备选择（可选择特定的输入/输出设备）

## 安装依赖

安装系统级依赖：

### PortAudio

PortAudio是一个跨平台的音频I/O库，需要为各平台安装：

#### macOS

```bash
brew install portaudio
```

#### Linux (Ubuntu/Debian)

```bash
sudo apt-get install portaudio19-dev
```

#### Windows

```bash
# 使用MSYS2/MinGW
pacman -S mingw-w64-x86_64-portaudio
```

### Opus

Opus是一个高质量的音频编解码器：

#### macOS

```bash
brew install opus
```

#### Linux (Ubuntu/Debian)

```bash
sudo apt-get install libopus-dev
```

#### Windows

```bash
# 使用MSYS2/MinGW
pacman -S mingw-w64-x86_64-opus
```

## 安装Go依赖

```bash
go mod download
```

## 配置

配置文件采用TOML格式，请参考`example-config.toml`了解详细设置。

主要音频配置项：

```toml
[Audio]
enabled = true                # 是否启用音频
input_device = ""             # 输入设备名称
output_device = ""            # 输出设备名称
sample_rate = 48000           # 采样率(Hz)
channels = 2                  # 通道数
frame_size = 960              # 帧大小
bitrate_kbps = 64             # 比特率(kbps)
opus_complexity = 10          # Opus编码复杂度
```

## 使用方法

1. 复制`example-config.toml`到`config.toml`并按需修改
2. 运行客户端：
   ```bash
   go run main.go
   ```

## 获取音频设备列表

系统会在启动时自动检测可用的音频设备。如果需要查看可用设备列表，可以使用以下命令：

```bash
go run tools/list_devices.go
``` 