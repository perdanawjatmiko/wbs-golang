package config

import (
	"fmt"
	"net"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type SMTPConfig struct {
	Enabled                                                        bool
	Host, Port, Username, Password, FromAddress, FromName, TLSMode string
	Timeout                                                        time.Duration
}
type TelegramConfig struct {
	Enabled       bool
	Token, ChatID string
	Timeout       time.Duration
}
type NotificationsConfig struct {
	SMTP                                                              SMTPConfig
	Telegram                                                          TelegramConfig
	WorkerEnabled                                                     bool
	BatchSize, Concurrency, MaxAttempts, SSEMaxConnections, SSEReplay int
	PollInterval, LeaseDuration, SSEPoll, SSEHeartbeat                time.Duration
	Retry                                                             []time.Duration
}

func parseNotifications(get func(string) string) (NotificationsConfig, error) {
	n := NotificationsConfig{}
	value := func(k, d string) string {
		if v := get(k); v != "" {
			return v
		}
		return d
	}
	for _, f := range []struct {
		k, d string
		p    *bool
	}{{"SMTP_ENABLED", "false", &n.SMTP.Enabled}, {"TELEGRAM_ENABLED", "false", &n.Telegram.Enabled}, {"NOTIFICATION_WORKER_ENABLED", "true", &n.WorkerEnabled}} {
		v, e := strconv.ParseBool(value(f.k, f.d))
		if e != nil {
			return n, fmt.Errorf("%s must be boolean", f.k)
		}
		*f.p = v
	}
	for _, f := range []struct {
		k, d     string
		p        *int
		min, max int
	}{{"NOTIFICATION_WORKER_BATCH_SIZE", "25", &n.BatchSize, 1, 100}, {"NOTIFICATION_WORKER_CONCURRENCY", "5", &n.Concurrency, 1, 32}, {"NOTIFICATION_MAX_ATTEMPTS", "5", &n.MaxAttempts, 1, 20}, {"SSE_MAX_CONNECTIONS", "100", &n.SSEMaxConnections, 1, 10000}, {"SSE_REPLAY_LIMIT", "100", &n.SSEReplay, 1, 1000}} {
		v, e := strconv.Atoi(value(f.k, f.d))
		if e != nil || v < f.min || v > f.max {
			return n, fmt.Errorf("%s is outside allowed range", f.k)
		}
		*f.p = v
	}
	for _, f := range []struct {
		k, d string
		p    *time.Duration
	}{{"SMTP_TIMEOUT", "15s", &n.SMTP.Timeout}, {"TELEGRAM_TIMEOUT", "10s", &n.Telegram.Timeout}, {"NOTIFICATION_WORKER_POLL_INTERVAL", "2s", &n.PollInterval}, {"NOTIFICATION_WORKER_LEASE_DURATION", "60s", &n.LeaseDuration}, {"SSE_POLL_INTERVAL", "2s", &n.SSEPoll}, {"SSE_HEARTBEAT_INTERVAL", "15s", &n.SSEHeartbeat}} {
		v, e := time.ParseDuration(value(f.k, f.d))
		if e != nil || v < 100*time.Millisecond || v > 24*time.Hour {
			return n, fmt.Errorf("%s must be 100ms..24h", f.k)
		}
		*f.p = v
	}
	if n.LeaseDuration < n.SMTP.Timeout+10*time.Second || n.LeaseDuration < n.Telegram.Timeout+10*time.Second {
		return n, fmt.Errorf("notification lease must exceed provider timeouts by at least 10s")
	}
	for _, s := range strings.Split(value("NOTIFICATION_RETRY_INTERVALS", "30s,2m,10m,30m,2h"), ",") {
		v, e := time.ParseDuration(s)
		if e != nil || v < time.Second || v > 24*time.Hour {
			return n, fmt.Errorf("NOTIFICATION_RETRY_INTERVALS requires 1s..24h durations")
		}
		n.Retry = append(n.Retry, v)
	}
	if len(n.Retry) > 20 {
		return n, fmt.Errorf("too many retry intervals")
	}
	n.SMTP.Host = get("SMTP_HOST")
	n.SMTP.Port = value("SMTP_PORT", "587")
	n.SMTP.Username = get("SMTP_USERNAME")
	n.SMTP.Password = get("SMTP_PASSWORD")
	n.SMTP.FromAddress = get("SMTP_FROM_ADDRESS")
	n.SMTP.FromName = value("SMTP_FROM_NAME", "WBS")
	n.SMTP.TLSMode = value("SMTP_TLS_MODE", "starttls")
	if n.SMTP.TLSMode != "starttls" && n.SMTP.TLSMode != "implicit" {
		return n, fmt.Errorf("SMTP_TLS_MODE must be starttls or implicit")
	}
	port, e := strconv.Atoi(n.SMTP.Port)
	if e != nil || port < 1 || port > 65535 {
		return n, fmt.Errorf("SMTP_PORT must be 1..65535")
	}
	if n.SMTP.Enabled {
		a, e := mail.ParseAddress(n.SMTP.FromAddress)
		if e != nil || a.Address != n.SMTP.FromAddress {
			return n, fmt.Errorf("SMTP_FROM_ADDRESS must be a valid sender")
		}
		if n.SMTP.Host == "" || strings.ContainsAny(n.SMTP.Host, "/\\\r\n :") && net.ParseIP(n.SMTP.Host) == nil {
			return n, fmt.Errorf("SMTP_HOST must be a hostname or IP")
		}
		if strings.ContainsAny(n.SMTP.FromName, "\r\n") {
			return n, fmt.Errorf("SMTP_FROM_NAME contains invalid characters")
		}
		if (n.SMTP.Username == "") != (n.SMTP.Password == "") {
			return n, fmt.Errorf("SMTP username and password must be configured together")
		}
	}
	n.Telegram.Token = get("TELEGRAM_BOT_TOKEN")
	n.Telegram.ChatID = get("TELEGRAM_CHAT_ID")
	if n.Telegram.Enabled && (!regexp.MustCompile(`^[0-9]+:[A-Za-z0-9_-]+$`).MatchString(n.Telegram.Token) || !regexp.MustCompile(`^-?[0-9]+$`).MatchString(n.Telegram.ChatID)) {
		return n, fmt.Errorf("Telegram requires a valid bot token and numeric chat ID")
	}
	return n, nil
}
