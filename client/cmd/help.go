package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// helpAudioCmd 音频功能帮助命令
var helpAudioCmd = &cobra.Command{
	Use:   "audio-help",
	Short: "显示音频功能的帮助信息",
	Long:  `详细说明音频功能的使用方法、配置选项和常见问题解决方案。`,
	Run: func(cmd *cobra.Command, args []string) {
		printAudioHelp()
	},
}

func init() {
	rootCmd.AddCommand(helpAudioCmd)
}

// printAudioHelp 打印音频功能帮助信息
func printAudioHelp() {
	help := `
音频功能使用指南
===============

1. 音频设备管理
---------------

列出系统上所有可用的音频设备:
  client devices

2. 音频测试
-----------

测试麦克风到扬声器的回环(默认测试):
  client audio --type loopback

测试麦克风并显示音量:
  client audio --type input

测试系统音频捕获(需要实现平台特定支持):
  client audio --type system

测试选项:
  --duration, -d: 测试持续时间（秒）
  --input, -i: 指定输入设备名称（留空使用配置中的设备）
  --config, -c: 指定配置文件路径

3. 配置音频
-----------

在配置文件中，音频设置项:

[Audio]
enabled = true                # 是否启用音频
input_device = ""             # 输入设备名称（空值表示使用默认设备）
output_device = ""            # 输出设备名称（空值表示使用默认设备）
capture_system = false        # 是否捕获系统音频输出
mix_with_mic = false          # 是否将系统音频与麦克风混合
sample_rate = 48000           # 采样率(Hz)
channels = 2                  # 通道数，1=单声道，2=立体声
frame_size = 960              # 帧大小，20ms@48kHz=960
bitrate_kbps = 64             # 比特率(kbps)
opus_complexity = 10          # Opus编码复杂度(0-10)

4. 常见问题
-----------

Q: 无法找到音频设备
A: 运行 "client devices" 确认系统能识别您的设备，尝试重新插拔设备

Q: 音质不佳
A: 尝试增加比特率(bitrate_kbps)或改用更好的麦克风

Q: 音频有延迟
A: 可以尝试减小帧大小(frame_size)，但会增加CPU使用率

Q: 系统音频捕获不工作
A: 系统音频捕获需要特定平台支持，目前需要实现
   Windows: 需要WASAPI Loopback支持
   macOS: 需要Audio Unit或虚拟音频设备
   Linux: 需要PulseAudio或JACK支持
`
	fmt.Println(help)
} 