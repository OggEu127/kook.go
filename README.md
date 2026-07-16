# KOOK Go SDK

一个面向 KOOK API v3 的 Go SDK，封装 HTTP API、WebSocket 事件连接和 Webhook 处理。

# SDK使用指南和文档

- [查看文档](https://oggeu.com/posts/Docs/kook-go-sdk-usage-guide)
- [可编译示例](examples)

## 安装

```bash
go get github.com/OggEu127/kook.go
```

## 快速开始

```go
client := kook.NewClient(os.Getenv("KOOK_TOKEN"))
defer client.Close()

me, err := client.User.GetMe(context.Background())
if err != nil {
	log.Fatal(err)
}
log.Printf("bot: %s (%s)", me.Username, me.ID)
```

WebSocket 事件按普通消息类型和 `extra.type` 系统事件名称分发：

```go
ws := kook.NewWebSocketClient(client, false)
ws.OnMessage(kook.MessageTypeText, func(event *kook.MessageEvent) {
	content, err := event.TextContent()
	if err == nil {
		log.Println(content)
	}
})
log.Fatal(ws.Connect())
```

离线检查使用 `go test -short ./...`。不带 `-short` 会运行真实 KOOK 只读测试，并要求详细指南中列出的测试环境变量。

要求 Go 1.21+
许可证见 [LICENSE](LICENSE)。
