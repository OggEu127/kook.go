# KOOK Go SDK

一个面向 KOOK API v3 的 Go SDK，封装 HTTP API、WebSocket 事件连接和 Webhook 处理。

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

## 常用能力

- 用户、服务器、频道、角色和权限管理
- 频道消息、私聊消息、消息模板、管道消息
- 邀请、邀请用户列表、黑名单、表情、徽章
- 媒体资源上传、亲密度、帖子、好友、游戏动态
- Gateway、WebSocket、Webhook

## WebSocket

```go
ws := kook.NewWebSocketClient(client, false)

ws.OnEvent(kook.EventTypeTextMessage, func(event *kook.Event) {
	fmt.Println("收到消息:", event.Content)
})

if err := ws.Connect(); err != nil {
	log.Fatal(err)
}
```

WebSocket 内置 Resume、Reconnect、Hello/Pong 超时、SN 顺序分发和事件缓冲保护。

## 本地检查

```bash
go test ./...
go vet ./...
go build ./...
```

## 要求

- Go 1.21+
- KOOK 机器人 Token

## 说明

这是 KOOK API 的非官方 Go SDK，与 KOOK 官方没有从属关系。许可证见 [LICENSE](LICENSE)。
