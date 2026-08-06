# Changelog

## v1.3.0 (2026-08-06)

- 增加可选的 SDK 生态客户端、stable/beta 版本检测、匿名在线实例租约、更新缓存/回调和 WebSocket 自动生命周期；SDK 版本更新为 1.3.0。
- 增加严格的 YAML 生态配置加载；`contribute_to_community` 默认开启，首次启用时非阻塞提示一次并持久化已提示状态，设置为 `false` 可关闭匿名心跳而保留版本查询。
- 增加可多副本部署的 `ecosystemd`，使用 PostgreSQL 保存版本历史、Redis 聚合在线租约，并提供公开统计、SVG 徽章、管理发布和健康检查接口。
- 增加生态服务 Docker/Compose、PostgreSQL/Redis 集成测试和 CI 验收；未显式配置生态服务时保持零上报。
- 统一 HTTP 限流、幂等感知重试、`Retry-After` 解析、错误处理和日志脱敏。
- 增加客户端/限流器关闭状态与配置错误构造器。
- 加固 Webhook 请求大小、解压、验证、背压、去重和优雅关闭。
- 修正 WebSocket resume SN 提交时机、事件缺口、队列溢出、解压上限与连接生命周期。
- 增加常见 v1.1.1 位置参数兼容、Legacy 返回入口及未确认端点的 `ErrUnsupportedEndpoint` 占位。
- 增加 101 个官方端点契约测试、兼容编译测试和 Go 1.21/1.26 CI。
- 经真实只读接口验证，兼容用户 `decorations_id_map` 中标量、数组、空值和空数组的混合响应，并补充帖子频道类型常量。
- 升级 gorilla/websocket 1.5.3、logrus 1.9.4、testify 1.11.1。
