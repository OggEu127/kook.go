# 测试指南

## 离线验收

项目最低支持 Go 1.21，发布门禁在 Go 1.21.13 和 Go 1.26.5 上运行。完整离线检查：

```bash
go test -short ./...
go test -race -short ./...
go vet ./...
golangci-lint run ./...
govulncheck ./...
go build ./...
git diff --check
```

锁定的质量工具版本为 golangci-lint 2.12.2 和 govulncheck 1.1.4。依赖应在 Go 1.21 下执行 `go mod tidy`，避免意外提高最低 Go 版本。

## 真实只读集成测试

不带 `-short` 的 `TestKOOKReadOnlyIntegration` 只读取现有资源，不创建、修改或删除 KOOK 数据。它也覆盖 `invite/invitees` 的受邀用户与留存统计查询。运行前安全注入下列环境变量：

```text
KOOK_TOKEN
KOOK_TEST_GUILD_ID
KOOK_TEST_TEXT_CHANNEL_ID
KOOK_TEST_VOICE_CHANNEL_ID
KOOK_TEST_THREAD_CHANNEL_ID
KOOK_TEST_USER_ID
KOOK_TEST_CHAT_CODE
KOOK_TEST_MESSAGE_ID
KOOK_TEST_ROLE_ID
KOOK_TEST_EMOJI_ID
```

然后运行：

```bash
go test ./kook -run '^TestKOOKReadOnlyIntegration$' -count=1 -v
```

不要把凭据写入 `.env`、测试夹具、命令行参数、日志或仓库文件。CI 不运行真实集成测试。

## 显式排除的测试

写入测试只有设置 `KOOK_ENABLE_MUTATION_TESTS=1` 才会执行；OAuth 授权码测试只有设置 `KOOK_ENABLE_OAUTH_TEST=1` 才会执行。它们不属于本项目的默认验收，也不应使用生产服务器或一次性授权码自动运行。
