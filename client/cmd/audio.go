package cmd

import (
	"client/audio"
	"client/config"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var (
	// 音频测试参数
	inputDevice string
	testType    string
	recordDuration int // 录制时长（秒）
)

// 音频测试命令
var audioCmd = &cobra.Command{
	Use:   "audio",
	Short: "测试音频设备",
	Long:  `测试音频输入输出设备，可以进行录音测试、播放测试或回环测试。`,
	Run: func(cmd *cobra.Command, args []string) {
		// 根据测试类型执行不同测试
		switch testType {
		case "loopback":
			testAudioLoopback()
		case "input":
			testAudioInput()
		case "system":
			testSystemCapture()
		case "full":
			testFullAudio()
		default:
			testFullAudio() // 默认进行完整测试
		}
	},
}

func init() {
	rootCmd.AddCommand(audioCmd)

	// 添加命令行参数
	audioCmd.Flags().StringVarP(&inputDevice, "input", "i", "", "输入设备名称（留空使用配置文件中的设置）")
	audioCmd.Flags().StringVarP(&testType, "type", "t", "full", "测试类型: full(完整测试)、loopback(回环)、input(录音)、system(系统声音)")
	audioCmd.Flags().IntVarP(&recordDuration, "record", "r", 5, "录制时长（秒）")
	// 复用run命令中的configPath变量
	audioCmd.Flags().StringVarP(&configPath, "config", "c", "config.toml", "配置文件路径")
}

// testFullAudio 完整测试，先测试麦克风录音，然后播放录制的音频
func testFullAudio() {
	fmt.Println("启动完整音频测试")
	fmt.Println("------------------------------------------------------")
	fmt.Println("第一阶段: 麦克风测试")
	fmt.Println("请对着麦克风说话。录制", recordDuration, "秒音频...")
	fmt.Println("------------------------------------------------------")
	
	// 加载配置
	cfg, err := loadAudioConfig()
	if err != nil {
		log.Error("加载配置失败", "error", err)
		return
	}
	
	// 如果提供了输入设备参数，覆盖配置
	if inputDevice != "" {
		cfg.Audio.InputDevice = inputDevice
	}
	
	// 创建音频配置
	audioConfig := audio.AudioConfig{
		Enabled:       true,
		InputDevice:   cfg.Audio.InputDevice,
		OutputDevice:  cfg.Audio.OutputDevice,
		SampleRate:    cfg.Audio.SampleRate,
		Channels:      cfg.Audio.Channels,
		FrameSize:     cfg.Audio.FrameSize,
		BitrateKbps:   cfg.Audio.BitrateKbps,
		OpusComplexity: cfg.Audio.OpusComplexity,
	}
	
	// 创建音频源
	source, err := audio.NewAudioSource(audioConfig)
	if err != nil {
		log.Error("创建音频源失败", "error", err)
		return
	}
	
	// 启动输入设备
	if err := source.Start(); err != nil {
		log.Error("启动音频输入失败", "error", err)
		return
	}
	defer source.Stop()
	
	// 创建一个大的缓冲区用于存储录制的样本
	// 假设最大录制时间为recordDuration秒
	maxSamples := audioConfig.SampleRate * audioConfig.Channels * recordDuration
	recordedBuffer := make([]int16, 0, maxSamples)
	
	// 创建输入缓冲区
	frameSize := audioConfig.FrameSize
	channels := audioConfig.Channels
	buffer := make([]int16, frameSize*channels)
	
	// 创建中断信号通道
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	// 创建录制完成通道
	recordDone := make(chan struct{})
	
	// 读取和显示音量循环
	go func() {
		startTime := time.Now()
		for {
			// 检查是否录制足够时间
			if time.Since(startTime) >= time.Duration(recordDuration)*time.Second {
				close(recordDone)
				return
			}
			
			samples, err := source.Read(buffer)
			if err != nil {
				log.Error("读取音频失败", "error", err)
				continue
			}
			
			if samples == 0 {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			
			// 计算音量级别
			var sum int64
			totalSamples := samples * channels
			for i := 0; i < totalSamples; i++ {
				if buffer[i] < 0 {
					sum += int64(-buffer[i])
				} else {
					sum += int64(buffer[i])
				}
				
				// 将样本添加到录制缓冲区
				recordedBuffer = append(recordedBuffer, buffer[i])
			}
			avg := sum / int64(totalSamples)
			
			// 显示音量指示器
			meter := ""
			level := avg / 328 // 32767/100
			for i := 0; i < int(level); i++ {
				meter += "█"
			}
			
			// 显示剩余时间
			remaining := recordDuration - int(time.Since(startTime).Seconds())
			fmt.Printf("\r音量: %-50s %3d%% | 剩余时间: %ds", meter, level, remaining)
		}
	}()
	
	// 等待录制完成或中断
	select {
	case <-sigChan:
		fmt.Println("\n测试被中断")
		return
	case <-recordDone:
		fmt.Println("\n录制完成！")
	}
	
	// 如果没有录制到任何音频，结束测试
	if len(recordedBuffer) == 0 {
		fmt.Println("未检测到有效音频输入，测试结束")
		return
	}
	
	// 计算录制的音频时长（秒）
	recordedSeconds := float64(len(recordedBuffer)) / float64(audioConfig.SampleRate * audioConfig.Channels)
	
	fmt.Println("\n------------------------------------------------------")
	fmt.Println("第二阶段: 扬声器测试")
	fmt.Printf("播放刚才录制的声音... %.1f 秒的录音\n", recordedSeconds)
	fmt.Println("------------------------------------------------------")
	
	// 创建音频接收器
	sink, err := audio.NewAudioSink(audioConfig)
	if err != nil {
		log.Error("创建音频接收器失败", "error", err)
		return
	}
	
	// 启动输出设备
	if err := sink.Start(); err != nil {
		log.Error("启动音频输出失败", "error", err)
		return
	}
	defer sink.Stop()
	
	// 创建播放完成通道
	playbackDone := make(chan struct{})
	
	// 播放录制的声音
	go func() {
		chunkSize := frameSize * channels
		totalChunks := (len(recordedBuffer) + chunkSize - 1) / chunkSize // 向上取整
		
		for i := 0; i < totalChunks; i++ {
			start := i * chunkSize
			end := start + chunkSize
			if end > len(recordedBuffer) {
				end = len(recordedBuffer)
			}
			
			// 写入音频
			if err := sink.Write(recordedBuffer[start:end]); err != nil {
				log.Error("写入音频失败", "error", err)
				break
			}
			
			// 显示播放进度
			progress := (i + 1) * 100 / totalChunks
			progressBar := strings.Repeat("=", progress/2) + ">"
			fmt.Printf("\r播放进度: [%-50s] %3d%%", progressBar, progress)
			
			// 短暂延迟，确保音频平滑播放
			time.Sleep(time.Duration(float64(end-start) * 1000 / float64(audioConfig.SampleRate*channels)) * time.Millisecond)
		}
		
		fmt.Printf("\r播放进度: [%s] 100%%\n", strings.Repeat("=", 50))
		close(playbackDone)
	}()
	
	// 等待播放完成或中断
	select {
	case <-sigChan:
		fmt.Println("\n测试被中断")
	case <-playbackDone:
		fmt.Println("播放完成！")
	}
	
	fmt.Println("\n------------------------------------------------------")
	fmt.Println("音频测试完成")
	fmt.Println("如果您能听到您录制的声音，则表示音频设备工作正常。")
	fmt.Println("------------------------------------------------------")
}

// testAudioLoopback 测试音频回环（麦克风到扬声器）
func testAudioLoopback() {
	loopbackDuration := 10 // 默认回环测试持续10秒
	
	fmt.Println("启动音频回环测试 (麦克风 -> 扬声器)")
	fmt.Println("测试将持续", loopbackDuration, "秒，或按Ctrl+C结束")
	
	// 加载配置
	cfg, err := loadAudioConfig()
	if err != nil {
		log.Error("加载配置失败", "error", err)
		return
	}
	
	// 如果提供了输入设备参数，覆盖配置
	if inputDevice != "" {
		cfg.Audio.InputDevice = inputDevice
	}
	
	// 创建音频配置
	audioConfig := audio.AudioConfig{
		Enabled:       true,
		InputDevice:   cfg.Audio.InputDevice,
		OutputDevice:  cfg.Audio.OutputDevice,
		SampleRate:    cfg.Audio.SampleRate,
		Channels:      cfg.Audio.Channels,
		FrameSize:     cfg.Audio.FrameSize,
		BitrateKbps:   cfg.Audio.BitrateKbps,
		OpusComplexity: cfg.Audio.OpusComplexity,
	}
	
	// 创建音频源和接收器
	source, err := audio.NewAudioSource(audioConfig)
	if err != nil {
		log.Error("创建音频源失败", "error", err)
		return
	}
	
	sink, err := audio.NewAudioSink(audioConfig)
	if err != nil {
		log.Error("创建音频接收器失败", "error", err)
		return
	}
	
	// 启动设备
	if err := source.Start(); err != nil {
		log.Error("启动音频输入失败", "error", err)
		return
	}
	defer source.Stop()
	
	if err := sink.Start(); err != nil {
		log.Error("启动音频输出失败", "error", err)
		source.Stop()
		return
	}
	defer sink.Stop()
	
	// 创建信号通道以捕获中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	// 创建定时器
	timer := time.NewTimer(time.Duration(loopbackDuration) * time.Second)
	
	// 创建缓冲区
	frameSize := audioConfig.FrameSize
	channels := audioConfig.Channels
	buffer := make([]int16, frameSize*channels)
	
	// 读取和回放循环
	go func() {
		for {
			samples, err := source.Read(buffer)
			if err != nil {
				log.Error("读取音频失败", "error", err)
				continue
			}
			
			if samples == 0 {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			
			if err := sink.Write(buffer[:samples*channels]); err != nil {
				log.Error("写入音频失败", "error", err)
			}
		}
	}()
	
	// 等待中断或定时器
	select {
	case <-sigChan:
		fmt.Println("\n测试被中断")
	case <-timer.C:
		fmt.Println("\n测试完成")
	}
}

// testAudioInput 测试音频输入设备
func testAudioInput() {
	inputTestDuration := 10 // 默认输入测试持续10秒
	
	fmt.Println("启动音频输入测试 (仅麦克风)")
	fmt.Println("测试将持续", inputTestDuration, "秒，或按Ctrl+C结束")
	fmt.Println("当检测到音频输入时，将显示音量级别")
	
	// 加载配置
	cfg, err := loadAudioConfig()
	if err != nil {
		log.Error("加载配置失败", "error", err)
		return
	}
	
	// 如果提供了输入设备参数，覆盖配置
	if inputDevice != "" {
		cfg.Audio.InputDevice = inputDevice
	}
	
	// 创建音频配置
	audioConfig := audio.AudioConfig{
		Enabled:       true,
		InputDevice:   cfg.Audio.InputDevice,
		SampleRate:    cfg.Audio.SampleRate,
		Channels:      cfg.Audio.Channels,
		FrameSize:     cfg.Audio.FrameSize,
	}
	
	// 创建音频源
	source, err := audio.NewAudioSource(audioConfig)
	if err != nil {
		log.Error("创建音频源失败", "error", err)
		return
	}
	
	// 启动设备
	if err := source.Start(); err != nil {
		log.Error("启动音频输入失败", "error", err)
		return
	}
	defer source.Stop()
	
	// 创建信号通道以捕获中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	// 创建定时器
	timer := time.NewTimer(time.Duration(inputTestDuration) * time.Second)
	
	// 创建缓冲区
	frameSize := audioConfig.FrameSize
	channels := audioConfig.Channels
	buffer := make([]int16, frameSize*channels)
	
	// 读取循环
	go func() {
		for {
			samples, err := source.Read(buffer)
			if err != nil {
				log.Error("读取音频失败", "error", err)
				continue
			}
			
			if samples == 0 {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			
			// 计算音量级别
			var sum int64
			for i := 0; i < samples*channels; i++ {
				if buffer[i] < 0 {
					sum += int64(-buffer[i])
				} else {
					sum += int64(buffer[i])
				}
			}
			avg := sum / int64(samples*channels)
			
			// 显示音量指示器
			meter := ""
			level := avg / 328 // 32767/100
			for i := 0; i < int(level); i++ {
				meter += "█"
			}
			fmt.Printf("\r音量: %-50s %3d%%", meter, level)
		}
	}()
	
	// 等待中断或定时器
	select {
	case <-sigChan:
		fmt.Println("\n测试被中断")
	case <-timer.C:
		fmt.Println("\n测试完成")
	}
}

// testSystemCapture 测试系统音频捕获
func testSystemCapture() {
	systemCaptureDuration := 10 // 默认系统音频捕获测试持续10秒
	
	fmt.Println("启动系统音频捕获测试")
	fmt.Println("测试将持续", systemCaptureDuration, "秒，或按Ctrl+C结束")
	fmt.Println("请播放一些系统声音...")
	
	// 加载配置
	cfg, err := loadAudioConfig()
	if err != nil {
		log.Error("加载配置失败", "error", err)
		return
	}
	
	// 强制启用系统音频捕获
	cfg.Audio.CaptureSystem = true
	cfg.Audio.MixWithMic = false
	
	// 创建音频配置
	audioConfig := audio.AudioConfig{
		Enabled:       true,
		InputDevice:   cfg.Audio.InputDevice,
		OutputDevice:  cfg.Audio.OutputDevice,
		SampleRate:    cfg.Audio.SampleRate,
		Channels:      cfg.Audio.Channels,
		FrameSize:     cfg.Audio.FrameSize,
	}
	
	// 创建系统音频源
	source, err := audio.NewLoopbackSource(audioConfig)
	if err != nil {
		log.Error("创建系统音频源失败", "error", err)
		fmt.Println("您的系统可能不支持系统音频捕获。")
		fmt.Println("需要实现特定平台的音频回环捕获。")
		return
	}
	
	// 创建音频接收器
	sink, err := audio.NewAudioSink(audioConfig)
	if err != nil {
		log.Error("创建音频接收器失败", "error", err)
		return
	}
	
	// 启动设备
	if err := source.Start(); err != nil {
		log.Error("启动系统音频捕获失败", "error", err)
		return
	}
	defer source.Stop()
	
	if err := sink.Start(); err != nil {
		log.Error("启动音频输出失败", "error", err)
		source.Stop()
		return
	}
	defer sink.Stop()
	
	// 创建信号通道以捕获中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	// 创建定时器
	timer := time.NewTimer(time.Duration(systemCaptureDuration) * time.Second)
	
	// 创建缓冲区
	frameSize := audioConfig.FrameSize
	channels := audioConfig.Channels
	buffer := make([]int16, frameSize*channels)
	
	// 读取和回放循环
	go func() {
		for {
			samples, err := source.Read(buffer)
			if err != nil {
				log.Error("读取系统音频失败", "error", err)
				continue
			}
			
			if samples == 0 {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			
			// 计算音量级别
			var sum int64
			for i := 0; i < samples*channels; i++ {
				if buffer[i] < 0 {
					sum += int64(-buffer[i])
				} else {
					sum += int64(buffer[i])
				}
			}
			avg := sum / int64(samples*channels)
			
			// 显示音量指示器
			meter := ""
			level := avg / 328 // 32767/100
			for i := 0; i < int(level); i++ {
				meter += "█"
			}
			fmt.Printf("\r系统音量: %-50s %3d%%", meter, level)
			
			// 回放
			if err := sink.Write(buffer[:samples*channels]); err != nil {
				log.Error("写入音频失败", "error", err)
			}
		}
	}()
	
	// 等待中断或定时器
	select {
	case <-sigChan:
		fmt.Println("\n测试被中断")
	case <-timer.C:
		fmt.Println("\n测试完成")
	}
}

// loadAudioConfig 加载音频配置
func loadAudioConfig() (*config.Config, error) {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}
	
	// 确保音频设置有默认值
	if cfg.Audio.SampleRate == 0 {
		cfg.Audio.SampleRate = 48000
	}
	if cfg.Audio.Channels == 0 {
		cfg.Audio.Channels = 2
	}
	if cfg.Audio.FrameSize == 0 {
		cfg.Audio.FrameSize = 960
	}
	if cfg.Audio.BitrateKbps == 0 {
		cfg.Audio.BitrateKbps = 64
	}
	if cfg.Audio.OpusComplexity == 0 {
		cfg.Audio.OpusComplexity = 10
	}
	
	return cfg, nil
} 