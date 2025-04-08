package cmd

import (
	"fmt"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gordonklaus/portaudio"
	"github.com/spf13/cobra"
)

// 列出音频设备命令
var devicesCmd = &cobra.Command{
	Use:   "devices",
	Short: "列出可用的音频设备",
	Long:  `列出系统中所有可用的音频输入和输出设备，包括默认设备、通道数、采样率和延迟信息。`,
	Run: func(cmd *cobra.Command, args []string) {
		listAudioDevices()
	},
}

func init() {
	rootCmd.AddCommand(devicesCmd)
}

// listAudioDevices 列出所有音频设备
func listAudioDevices() {
	// 初始化PortAudio
	if err := portaudio.Initialize(); err != nil {
		log.Error("初始化PortAudio失败", "error", err)
		return
	}
	defer portaudio.Terminate()

	// 获取所有设备
	devices, err := portaudio.Devices()
	if err != nil {
		log.Error("获取音频设备失败", "error", err)
		return
	}

	// 获取默认输入设备
	defaultInput, err := portaudio.DefaultInputDevice()
	if err != nil {
		log.Error("获取默认输入设备失败", "error", err)
	}

	// 获取默认输出设备
	defaultOutput, err := portaudio.DefaultOutputDevice()
	if err != nil {
		log.Error("获取默认输出设备失败", "error", err)
	}

	// 打印输入设备
	fmt.Println("=== 输入设备 ===")
	fmt.Println()
	for i, device := range devices {
		if device.MaxInputChannels > 0 {
			isDefault := ""
			if defaultInput != nil && device.Name == defaultInput.Name {
				isDefault = " (默认)"
			}
			fmt.Printf("[%d] %s%s\n", i, device.Name, isDefault)
			fmt.Printf("    输入通道: %d\n", device.MaxInputChannels)
			fmt.Printf("    采样率: %.0f Hz\n", device.DefaultSampleRate)
			
			// 正确处理time.Duration类型，将纳秒转换为毫秒
			lowLatency := float64(device.DefaultLowInputLatency) / float64(time.Millisecond)
			highLatency := float64(device.DefaultHighInputLatency) / float64(time.Millisecond)
			fmt.Printf("    延迟: 低=%.2f毫秒 高=%.2f毫秒\n", lowLatency, highLatency)
			fmt.Println()
		}
	}

	// 打印输出设备
	fmt.Println("=== 输出设备 ===")
	fmt.Println()
	for i, device := range devices {
		if device.MaxOutputChannels > 0 {
			isDefault := ""
			if defaultOutput != nil && device.Name == defaultOutput.Name {
				isDefault = " (默认)"
			}
			fmt.Printf("[%d] %s%s\n", i, device.Name, isDefault)
			fmt.Printf("    输出通道: %d\n", device.MaxOutputChannels)
			fmt.Printf("    采样率: %.0f Hz\n", device.DefaultSampleRate)
			
			// 正确处理time.Duration类型，将纳秒转换为毫秒
			lowLatency := float64(device.DefaultLowOutputLatency) / float64(time.Millisecond)
			highLatency := float64(device.DefaultHighOutputLatency) / float64(time.Millisecond)
			fmt.Printf("    延迟: 低=%.2f毫秒 高=%.2f毫秒\n", lowLatency, highLatency)
			fmt.Println()
		}
	}

	// 使用提示
	fmt.Println("=== 配置音频设备 ===")
	fmt.Println("在配置文件中设置input_device和output_device参数:")
	fmt.Println("例如:")
	fmt.Println("[Audio]")
	fmt.Println("input_device = \"设备名称\"  # 留空使用系统默认设备")
	fmt.Println("output_device = \"设备名称\" # 留空使用系统默认设备")
} 