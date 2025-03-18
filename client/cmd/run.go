package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"client/config"
	"client/logger"
	"client/webrtc"
	"client/websocket"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var configPath string

// runCmd 表示run命令
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "运行WebSocket客户端",
	Long:  `启动WebSocket客户端并连接到服务器。`,
	Run: func(cmd *cobra.Command, args []string) {
		// 初始化日志记录器
		if err := logger.InitLogger("logs"); err != nil {
			log.Fatal("初始化日志记录器失败", "error", err)
		}

		// 加载配置文件
		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			log.Fatal("配置文件加载失败", "error", err)
		}

		// 创建WebSocket客户端
		wsClient := websocket.NewClient(cfg)

		// 创建WebRTC客户端
		webrtcClient := webrtc.NewClient(cfg, wsClient)

		// 设置WebRTC客户端到WebSocket客户端
		wsClient.SetWebRTCClient(webrtcClient)

		// 设置中断信号处理
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

		// 创建一个可取消的context
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// 启动连接循环
		go func() {
			for {
				select {
				case <-ctx.Done():
					log.Info("收到停止信号，终止连接循环")
					return
				default:
					// 连接到服务器
					if err := wsClient.Connect(); err != nil {
						log.Error("连接失败", "error", err)
						select {
						case <-ctx.Done():
							return
						case <-time.After(time.Duration(cfg.WebSocket.ReconnectDelay) * time.Second):
							continue
						}
					}
					// 连接成功后等待连接关闭
					<-wsClient.Done()
					log.Info("连接已关闭，准备重连")
					// 添加小延迟，避免立即重连
					time.Sleep(time.Second)
				}
			}
		}()

		// 等待中断信号
		sig := <-interrupt
		log.Info("收到中断信号，正在关闭连接...", "signal", sig)
		cancel() // 取消context，停止重连

		// 设置一个超时，确保有足够时间干净地关闭
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		// 关闭客户端连接
		wsClient.Close()

		// 关闭WebRTC客户端
		webrtcClient.Close()

		// 等待一会儿，确保资源清理完成
		select {
		case <-shutdownCtx.Done():
			log.Warn("关闭超时，强制退出")
		case <-time.After(500 * time.Millisecond):
			log.Info("客户端已正常关闭")
		}
	},
}

func init() {
	runCmd.Flags().StringVarP(&configPath, "config", "c", "config.toml", "配置文件路径")
	rootCmd.AddCommand(runCmd)
}
