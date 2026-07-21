# 安全策略

## 报告漏洞

请通过 GitHub Security Advisory 私下报告安全问题。报告中应包含受影响版本、复现步骤、影响范围和建议修复；不要在公开 Issue 中提交有效 KOOK token、OAuth code、client secret、Webhook 验证令牌或 gateway 凭据。

## 凭据与日志

- 只通过运行环境的机密管理或环境变量注入凭据，不要提交到仓库。
- SDK 的 HTTP 日志只输出脱敏 URL、方法、状态码、请求 ID 和字节数，不输出请求体或响应体。
- Webhook 必须配置非空 verify token；无效旧构造器会产生失败关闭实例。
- 非幂等重试默认关闭。启用 `RetryNonIdempotent` 前应评估重复写入风险。

## 支持范围

安全修复以当前主分支和最新发布版本为优先。Go 最低版本为 1.21；建议使用仍受 Go 项目支持的补丁版本，并定期运行 `govulncheck ./...`。
