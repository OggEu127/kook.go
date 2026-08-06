# KOOK Go SDK

一个面向 [KOOK API v3](https://developer.kookapp.cn/doc/reference) 的 Go SDK，封装 HTTP API、WebSocket 事件连接、安全 Webhook 处理，以及可选的 SDK 生态版本检测与匿名在线实例统计。模块保持 v1 路径，最低支持 Go 1.21。

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

## SDK 生态与版本检测

生态能力默认完全关闭，不会产生额外网络请求。部署自己的 `ecosystemd` 后，显式配置其 HTTPS 地址。启用生态服务后，匿名社区贡献默认开启；可以通过 YAML 配置关闭。

推荐从配置文件加载，参考 [`examples/ecosystem.example.yaml`](examples/ecosystem.example.yaml)：

```yaml
base_url: https://ecosystem.example.com
channel: stable
contribute_to_community: true
```

```go
ecosystemOptions, err := kook.LoadEcosystemOptions("ecosystem.yaml")
if err != nil {
	log.Fatal(err)
}

client, err := kook.NewClientWithError(
	os.Getenv("KOOK_TOKEN"),
	kook.WithEcosystem(ecosystemOptions),
)
```

`contribute_to_community` 省略时默认为 `true`。首次启用时，SDK 会非阻塞地显示一次匿名贡献说明，告知用户可在配置文件中将该字段改为 `false`；程序会继续按默认值运行，不等待终端输入。已提示状态默认记录在配置文件旁的 `.community-notice-v1` 标记中，后续启动不再提示。Docker 等临时文件系统可以通过 `notice_state_path` 将标记放入持久化卷。

设置为 `false` 后不会显示贡献提示，也不会生成实例 ID、发送心跳或登记在线租约，但 `CheckVersion` 和 `GetOnlineStats` 仍然可用。SDK 不会自动寻找配置文件，必须由机器人程序显式调用 `LoadEcosystemOptions`。

也可以直接在代码中配置：

```go
client, err := kook.NewClientWithError(
	os.Getenv("KOOK_TOKEN"),
	kook.WithEcosystem(kook.EcosystemOptions{
		BaseURL: "https://ecosystem.example.com",
		Channel: kook.ReleaseChannelStable,
		ContributeToCommunity: kook.CommunityContribution(true),
		NoticeStatePath: "/var/lib/kook-go-sdk/community-notice-v1",
		OnUpdateAvailable: func(status kook.SDKUpdateStatus) {
			log.Printf("SDK %s 可更新到 %s: %s",
				status.CurrentVersion, status.LatestVersion, status.ReleaseURL)
		},
	}),
)
if err != nil {
	log.Fatal(err)
}
defer func() { _ = client.Close() }()
```

配置后，WebSocket 只有在收到成功 Hello 后才自动登记在线租约；断线暂停续租，重连沿用当前进程内的匿名实例 ID。Webhook 机器人应在成功上线后显式管理生命周期：

```go
if err := client.Ecosystem.Start(ctx, kook.WebhookTransport); err != nil {
	log.Printf("生态统计暂时不可用: %v", err)
}
defer func() {
	stopCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = client.Ecosystem.Stop(stopCtx)
}()
```

主动版本检测和公开统计：

```go
status, err := client.Ecosystem.CheckVersion(ctx)
stats, err := client.Ecosystem.GetOnlineStats(ctx)
```

生态心跳只发送随机的进程内实例 ID、SDK/Go 版本、OS、架构、发布通道和 gateway/webhook 类型，不发送 KOOK Token、机器人 ID、服务器 ID 或消息。公开的 `online_instances` 是 90 秒租约窗口内的活跃 SDK `Client` 近似数量，不等同于唯一机器人数量。

SDK 只提示可用版本，不下载或替换机器人程序。升级仍需固定新的 Go module 版本并重新构建。

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

WebSocket 使用有界事件队列。只有处理器执行完成后才提交用于恢复连接的 SN；队列溢出、乱序缓冲溢出或序列缺口超时会关闭当前连接并尝试重连。重放可能产生重复事件，因此业务处理器应具备幂等性并尽快返回。

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

Webhook 默认拒绝空 `verify_token`，限制压缩请求体为 1 MiB、解压后为 8 MiB，并使用容量 256 的单 worker 顺序队列。队列满时返回 503，且不会写入去重状态。内建服务器使用私有 mux 和 HTTP 超时；完整程序见 [webhook_bot](examples/webhook_bot/main.go)。

## 请求、限流与重试

- JSON、multipart 和 Badge 二进制请求共享限流、重试与错误解析逻辑。
- `GET`、`HEAD`、`PUT`、`DELETE` 属于可安全重试的方法。
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

## 部署生态服务

中心服务入口为 `./cmd/ecosystemd`，依赖 PostgreSQL 和 Redis。它会在启动时用 PostgreSQL advisory lock 自动执行内嵌 migration。

本地启动：

```bash
docker compose -f docker-compose.ecosystem.yml up --build
```

生产环境必须在 HTTPS 反向代理后运行，并设置：

| 环境变量 | 说明 |
| --- | --- |
| `DATABASE_URL` | PostgreSQL DSN |
| `REDIS_URL` | Redis URL |
| `ECOSYSTEM_ADMIN_TOKEN` | 版本发布管理接口 Bearer Token |
| `ECOSYSTEM_RATE_LIMIT_SALT` | 至少 16 字符，各副本一致的来源 IP 哈希盐 |
| `ECOSYSTEM_LISTEN_ADDR` | 监听地址，默认 `:8080` |
| `ECOSYSTEM_RATE_LIMIT_PER_MINUTE` | 每来源 IP 心跳/注销请求上限，默认 600 |
| `TRUSTED_PROXY_CIDRS` | 可选，允许读取 `X-Forwarded-For` 的代理 CIDR/IP 列表 |
| `PUBLIC_BASE_URL` | 可选，配置时必须是 HTTPS URL |

管理员通过 `PUT /v1/admin/channels/{stable|beta}/releases/{version}` 发布清单，请求体包含 `minimum_supported_version`、HTTPS `release_url` 和可选 `message`。管理 Token 应通过部署平台的 secret 注入，不要写入仓库、日志或命令行参数。

公开接口：

- `GET /v1/sdk/releases/latest?channel=stable&current_version=1.3.0`
- `GET /v1/stats/online`
- `GET /v1/badges/online.svg`
- `GET /health/live` 与 `GET /health/ready`

README 徽章可使用：

```markdown
![kook.go online](https://ecosystem.example.com/v1/badges/online.svg)
```

匿名心跳只能通过限流减轻常规刷量，不能提供机器人身份级别的真实性保证。生产代理也应配置请求速率限制、TLS 和合理的访问日志保留周期。

真实只读集成测试所需环境变量，以及默认禁用的写入测试和 OAuth 测试，见 [TESTING.md](TESTING.md)。不要把 KOOK Token、OAuth 授权码或其他凭据写入仓库、命令行参数或日志。

许可证见 [LICENSE](LICENSE)。
