# Changelog

## Unreleased

## v1.2.0 - 2026-07-21

- 统一 HTTP 限流、幂等感知重试、`Retry-After` 解析、错误处理和日志脱敏。
- 修复被多层包装的超时和常见连接错误无法进入方法感知重试的问题。
- 增加客户端/限流器关闭状态与配置错误构造器。
- Webhook 默认改为处理成功后确认，增加事务型去重预留/提交/释放，用 SN 和负载摘要避免回卷碰撞。
- Webhook Encrypt Key 支持 KOOK 规则下的短密钥补零，并保留显式的入队后确认兼容模式。
- 修正 WebSocket resume SN 提交时机、`sn=0` Resume、ResumeAck 前重放事件、事件缺口、队列溢出、解压上限与连接生命周期。
- WebSocket 后台重连改为可取消、延迟有上限且持续到成功或关闭。
- 新增官方 `GET invite/invitees` 受邀用户与留存统计接口。
- 增加常见 v1.1.1 位置参数兼容、Legacy 返回入口及未确认端点的 `ErrUnsupportedEndpoint` 占位。
- 增加 102 个官方端点契约测试、兼容编译测试和 Go 1.21/1.26 CI。
- 经真实只读接口验证，兼容用户 `decorations_id_map` 中标量、数组、空值和空数组的混合响应，并补充帖子频道类型常量。
- 升级 gorilla/websocket 1.5.3、logrus 1.9.4、testify 1.11.1。
