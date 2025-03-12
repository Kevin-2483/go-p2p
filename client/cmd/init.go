package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// initCmd 表示init命令
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "生成默认配置文件",
	Long:  `生成一个包含基本配置的config.toml文件。`,
	Run: func(cmd *cobra.Command, args []string) {
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
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}