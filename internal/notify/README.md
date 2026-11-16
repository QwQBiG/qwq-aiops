# 通知服务 / Notifications

运行入口通过 `InitNotificationService` 读取生效配置。钉钉使用 `DINGTALK_WEBHOOK`，Telegram 使用 `TELEGRAM_TOKEN` 和 `TELEGRAM_CHAT_ID`；两个 Telegram 字段必须同时配置。两个渠道均启用时，每个渠道都会收到消息。

命令行和需要确认发送结果的调用者使用同步接口，并传入当前操作的上下文：

```go
if err := notify.SendStatusReportContext(ctx, report); err != nil {
    return err
}
// 普通消息：notify.SendContext(ctx, title, content)
```

请求最多等待 10 秒，并响应调用者取消。成功要求 HTTP 响应和渠道业务结果均成功：钉钉的 `errcode` 为 `0`，Telegram 的 `ok` 为 `true`。部分渠道失败也会返回错误，调用者可以知道哪些渠道未完成发送。不会自动重发，避免重复通知。

钉钉以 Markdown 发送；Telegram 以纯文本发送，保留报告中的下划线、方括号等字符。`ValidateConfig` 只检查配置，`TestConnection` 会真正发送测试消息。兼容接口 `Send` 异步发送并记录错误，短生命周期命令应使用同步接口。

测试使用本地 HTTP 服务器覆盖成功、业务拒绝、HTTP 错误、取消和部分失败，无需真实机器人凭据：

```sh
go test ./internal/notify ./internal/deployment
```

The entrypoint calls `InitNotificationService` after loading the effective configuration. DingTalk uses `DINGTALK_WEBHOOK`; Telegram requires both `TELEGRAM_TOKEN` and `TELEGRAM_CHAT_ID`. When both channels are configured, both receive each message.

Use `SendContext(ctx, title, content)` or `SendStatusReportContext(ctx, report)` when the caller needs to wait for delivery. Requests have a 10-second timeout and honor cancellation. Success requires both HTTP success and the provider's success field (`errcode: 0` for DingTalk, `ok: true` for Telegram). A failure on any configured channel is returned, including partial delivery. There are no automatic retries.

DingTalk messages use Markdown. Telegram messages use plain text so report punctuation remains unchanged. `ValidateConfig` only inspects configuration; `TestConnection` sends a real test message. The compatibility `Send` function starts background delivery and logs failures, so short-lived commands should use the synchronous functions.

The tests above use local HTTP servers to verify success, provider rejection, HTTP errors, cancellation, and partial failure without live credentials.
