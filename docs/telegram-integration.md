# Telegram integration

Disabled by default. Configure TELEGRAM_ENABLED, TELEGRAM_BOT_TOKEN, numeric TELEGRAM_CHAT_ID (group IDs commonly negative), TELEGRAM_TIMEOUT(default10s). Give the configured bot permission to send to that group outside this development workflow. Never place token/chat secrets in frontend variables or logs.

The client uses the fixed HTTPS Telegram Bot API sendMessage endpoint; destination cannot come from a request or event payload. Redirects are refused. Bounded HTTP timeout and64KiB response limit apply. It validates HTTP status, JSON ok/result/message_id; 429 respects parameters.retry_after,5xx/network are retryable and permanent4xx fail safely. Raw errors/provider responses are never persisted.

Only REPORT_CREATED sends a plain-text group alert: report code, report type, status Open, created timestamp. No parse_mode, reporter name/contact, report detail, attachment URL, external/internal notes, admin metadata or secret. Status changes and notes never send Telegram alerts. There are no periodic summaries or additional bots. Automated tests use a local TLS test server and fake token; no Telegram requests occur.
