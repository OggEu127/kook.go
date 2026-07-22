# KOOK Go SDK

一个面向 [KOOK API v3](https://developer.kookapp.cn/doc/reference) 的 Go SDK，封装 HTTP API、WebSocket 事件连接和安全 Webhook 处理。模块保持 v1 路径，最低支持 Go 1.21。

## 使用指南和文档

- [完整使用与部署指南](https://oggeu.com/posts/Docs/kook-go-sdk-usage-guide)
- [可编译示例](examples)
- [测试指南](TESTING.md)
- [安全策略](SECURITY.md)
- [变更记录](CHANGELOG.md)

## 安装

```bash
go get github.com/OggEu127/kook.go@v1.2.0
```

README 描述当前分支。使用已发布版本时，建议固定版本号，并以对应 Git tag 中的文档为准。

## 快速开始

```go
client, err := kook.NewClientWithError(os.Getenv("KOOK_TOKEN"))
if err != nil {
	log.Fatal(err)
}
defer func() { _ = client.Close() }()

ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

me, err := client.User.GetMe(ctx)
if err != nil {
	log.Fatal(err)
}
log.Printf("bot: %s (%s)", me.Username, me.ID)
```

`NewClientWithError` 会返回空 Token、nil HTTP client/logger，以及非法重试或限流配置产生的错误。`Client.Close` 可以重复调用；关闭后的请求返回 `ErrClientClosed`。

## WebSocket 事件

普通消息按 `MessageType` 分发，系统事件按 `extra.type` 中的 `SystemEventType` 分发：

```go
ws, err := kook.NewWebSocketClientWithError(client, false)
if err != nil {
	log.Fatal(err)
}
defer func() { _ = ws.Close() }()

ws.OnMessage(kook.MessageTypeText, func(event *kook.MessageEvent) {
	content, err := event.TextContent()
	if err == nil {
		log.Println(content)
	}
})

if err := ws.Connect(); err != nil {
	log.Fatal(err)
}
// Connect 建立连接后返回；应用应继续运行，退出时调用 Close。
```

WebSocket 使用有界事件队列。只有处理器执行完成后才提交用于恢复连接的 SN；队列溢出、乱序缓冲溢出或序列缺口超时会关闭当前连接并从已提交 SN 重放。Resume 支持 `sn=0` 和确认前到达的重放事件；后台重连使用有上限的指数退避并持续到成功或 `Close`。重放可能产生重复事件，因此业务处理器应具备幂等性并尽快返回。

## Webhook

```go
handler, err := kook.NewWebhookHandlerWithError(
	client,
	os.Getenv("KOOK_ENCRYPT_KEY"),
	os.Getenv("KOOK_VERIFY_TOKEN"),
)
if err != nil {
	log.Fatal(err)
}

// 注册 OnMessage、OnSystemEvent 或 OnAnyEvent 后启动服务。
// 退出时使用带截止时间的 context 调用 handler.Shutdown(ctx)。
```

Webhook 默认拒绝空 `verify_token`，限制压缩请求体为 1 MiB、解压后为 8 MiB，并使用容量 256 的单 worker 顺序队列。默认只有在业务处理器成功返回并提交去重状态后才响应 200；panic、队列满和处理中重复请求不会被永久去重，KOOK 可以重投。事件键同时包含 SN 和负载摘要，避免 SN 回卷碰撞。短 Encrypt Key 按 KOOK 算法补零到 32 字节；超过 32 字节会在构造时拒绝。

多实例默认确认模式需要实现 `WebhookTransactionalDeduplicator`，并通过 `WithWebhookTransactionalDeduplicator` 注入；预留令牌会防止过期 worker 误提交新租约。仅兼容旧 `WebhookDeduplicator` 的存储必须显式选择 `WithWebhookAckMode(kook.WebhookAckAfterEnqueue)`，并自行保证入队后持久化。内建服务器使用私有 mux 和 HTTP 超时；完整程序见 [webhook_bot](examples/webhook_bot/main.go)。

## 邀请用户与留存统计

```go
status := kook.InviteeStatusAll
invitees, err := client.Invite.GetInvitees(ctx, kook.InviteeListParams{
	GuildID:  guildID,
	Status:   &status,
	Page:     1,
	PageSize: 20,
})
```

`GetInvitees` 对应官方 `GET invite/invitees`，支持邀请码、邀请 URL、服务器、状态和时间范围筛选，返回受邀用户列表以及 `Count`、`KeepCount`、`LossCount` 统计。

## 请求、限流与重试

- JSON、multipart 和 Badge 二进制请求共享限流、重试与错误解析逻辑。
- `GET`、`HEAD`、`OPTIONS`、`PUT`、`DELETE` 属于可安全重试的方法；被多层包装的超时和常见连接错误也会被识别。
- `POST`、`PATCH` 默认仅在服务端明确返回 429 时重试；`RetryNonIdempotent` 会放宽限制，也会增加重复写入风险。
- `MaxRetries` 表示首次请求之外的最大重试次数；`MaxRetries: 3` 最多执行 4 次请求。
- `Retry-After` 支持秒数和 HTTP-date，等待可以被 `context` 取消。
- 日志不会输出请求体、响应体或凭据；URL 中的 Token、授权码和 Client Secret 会被脱敏。

## v1.1.1 兼容边界

常见旧位置参数调用和当前 Params 调用可以同时使用；非法参数组合返回 `ValidationError`，不会 panic。`Role`/`Emoji` 与 `GuildRole`/`GuildEmoji` 分别指向同一服务实例。

返回值不兼容的方法保留当前签名，并提供 `UpdateEmojiLegacy`、`HandleFriendRequestLegacy`、`UpdateIntimacyLegacy` 和 `ReplyThreadLegacy`。历史上存在但未被当前官方文档确认的端点只保留编译兼容入口，调用返回可由 `errors.Is(err, kook.ErrUnsupportedEndpoint)` 判断的错误，并且不会发送网络请求。

## 测试

默认离线验收不需要 KOOK 凭据：

```bash
go test -short ./...
go test -race -short ./...
go vet ./...
go build ./...
git diff --check
```

安装 golangci-lint 2.12.2 和 govulncheck 1.1.4 后，可以运行完整检查：

```bash
make verify
```

真实只读集成测试所需环境变量，以及默认禁用的写入测试和 OAuth 测试，见 [TESTING.md](TESTING.md)。不要把 KOOK Token、OAuth 授权码或其他凭据写入仓库、命令行参数或日志。

许可证见 [LICENSE](LICENSE)。
