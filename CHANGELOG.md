# Changelog

## Unreleased

- 统一 HTTP 限流、幂等感知重试、`Retry-After` 解析、错误处理和日志脱敏。
- 增加客户端/限流器关闭状态与配置错误构造器。
- 加固 Webhook 请求大小、解压、验证、背压、去重和优雅关闭。
- 修正 WebSocket resume SN 提交时机、事件缺口、队列溢出、解压上限与连接生命周期。
- 增加常见 v1.1.1 位置参数兼容、Legacy 返回入口及未确认端点的 `ErrUnsupportedEndpoint` 占位。
- 增加 101 个官方端点契约测试、兼容编译测试和 Go 1.21/1.26 CI。
- 经真实只读接口验证，兼容用户 `decorations_id_map` 中标量、数组、空值和空数组的混合响应，并补充帖子频道类型常量。
- 升级 gorilla/websocket 1.5.3、logrus 1.9.4、testify 1.11.1。
