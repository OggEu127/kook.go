# KOOK Go SDK

一个用于调用 KOOK API v3 的 Go SDK，封装了常用 HTTP 接口、WebSocket 事件连接和 Webhook 处理。

## 相关文档

- [SDK 使用文档]([https://blog.oggeu.com/posts/Docs/kook-go-sdk])

## 安装

```bash
go get github.com/OggEu127/kook.go
```

## 快速开始

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/OggEu127/kook.go/kook"
)

func main() {
	client := kook.NewClient("你的机器人Token")

	me, err := client.User.GetMe(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("当前机器人: %s#%s\n", me.Username, me.IdentifyNum)
}
```

## 发送消息

```go
msg, err := client.Message.SendMessage(context.Background(), kook.SendMessageParams{
	TargetID: "频道ID",
	Content:  "你好，KOOK！",
})
if err != nil {
	log.Fatal(err)
}

fmt.Println("消息ID:", msg.ID)
```

## 获取服务器和频道

```go
guilds, err := client.Guild.GetGuildList(context.Background(), 1, 20, "")
if err != nil {
	log.Fatal(err)
}

channels, err := client.Channel.GetChannelList(context.Background(), "服务器ID", 1, 50, "")
if err != nil {
	log.Fatal(err)
}

fmt.Println("服务器数量:", len(guilds.Items))
fmt.Println("频道数量:", len(channels.Items))
```

## 上传文件

```go
asset, err := client.Asset.UploadFile(context.Background(), "./image.png")
if err != nil {
	log.Fatal(err)
}

fmt.Println("资源地址:", asset.URL)
```

## WebSocket 事件

```go
ws := kook.NewWebSocketClient(client, false)

ws.OnEvent(kook.EventTypeTextMessage, func(event *kook.Event) {
	fmt.Println("收到消息:", event.Content)
})

if err := ws.Connect(); err != nil {
	log.Fatal(err)
}

select {}
```

## 错误处理

```go
_, err := client.User.GetMe(context.Background())
if err != nil {
	if apiErr, ok := kook.IsKOOKError(err); ok {
		fmt.Printf("KOOK API 错误: code=%d message=%s\n", apiErr.Code, apiErr.Message)
		return
	}

	fmt.Println("请求失败:", err)
}
```

## 支持的主要功能

- 用户信息
- 服务器和成员管理
- 频道管理
- 频道权限覆写
- 频道消息和私聊消息
- 私信会话
- 好友接口
- 角色权限
- 邀请、黑名单、表情、徽章
- 媒体资源上传
- 亲密度、用户动态、帖子
- Gateway、WebSocket、Webhook

## 本地测试

```bash
go test ./...
go build ./...
```

运行根目录示例：

```bash
export KOOK_BOT_TOKEN="你的机器人Token"
go run .
```

运行 examples：

```bash
export KOOK_TOKEN="你的机器人Token"
go run examples/simple_bot/main.go
```

## 项目结构

```text
kook.go/
├── kook/       # SDK 核心代码
├── examples/   # 示例程序
├── go.mod
└── README.md
```

## 要求

- Go 1.21 或更高版本
- KOOK 机器人 Token

## 许可证

本项目使用允许商用、但源码必须开放的自定义许可证，详见 [LICENSE](LICENSE)。

简单说明：

- 允许商用。
- 修改、分发或基于本项目制作衍生作品时，必须开放完整对应源代码。
- 不允许将修改后的版本或包含本项目代码的衍生作品闭源发布。
- 商用修改版必须提供修改说明文档，说明修改的文件、功能变化和版本信息。

## 说明

这是 KOOK API 的非官方 Go SDK，与 KOOK 官方没有从属关系。
