package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"wbs-golang/backend/internal/config"
	"wbs-golang/backend/internal/modules/notifications"
)

type Client struct {
	config config.TelegramConfig
	http   *http.Client
	base   string
}

func New(c config.TelegramConfig) *Client {
	return &Client{c, &http.Client{Timeout: c.Timeout, CheckRedirect: func(*http.Request, []*http.Request) error { return fmt.Errorf("redirect refused") }}, "https://api.telegram.org"}
}
func (c *Client) Send(ctx context.Context, m notifications.Message) (string, error) {
	if !c.config.Enabled {
		return "", &notifications.Failure{Code: "CHANNEL_DISABLED"}
	}
	body, _ := json.Marshal(map[string]string{"chat_id": c.config.ChatID, "text": m.Text})
	req, err := http.NewRequestWithContext(ctx, "POST", c.base+"/bot"+c.config.Token+"/sendMessage", strings.NewReader(string(body)))
	if err != nil {
		return "", &notifications.Failure{Code: "TELEGRAM_CONFIG"}
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return "", &notifications.Failure{Code: "TELEGRAM_UNAVAILABLE", Retryable: true}
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 65537))
	if err != nil || len(data) > 65536 {
		return "", &notifications.Failure{Code: "TELEGRAM_RESPONSE_INVALID", Retryable: true}
	}
	var v struct {
		OK        bool `json:"ok"`
		ErrorCode int  `json:"error_code"`
		Result    struct {
			ID int64 `json:"message_id"`
		} `json:"result"`
		Parameters struct {
			After int `json:"retry_after"`
		} `json:"parameters"`
	}
	if err = json.Unmarshal(data, &v); err != nil {
		return "", &notifications.Failure{Code: "TELEGRAM_RESPONSE_INVALID", Retryable: res.StatusCode >= 500}
	}
	if res.StatusCode == 429 || v.ErrorCode == 429 {
		seconds := v.Parameters.After
		if seconds < 1 {
			seconds = 1
		}
		if seconds > 86400 {
			seconds = 86400
		}
		after := time.Duration(seconds) * time.Second
		return "", &notifications.Failure{Code: "TELEGRAM_RATE_LIMIT", Retryable: true, RetryAfter: after}
	}
	if res.StatusCode >= 500 {
		return "", &notifications.Failure{Code: "TELEGRAM_UNAVAILABLE", Retryable: true}
	}
	if res.StatusCode != 200 || !v.OK || v.Result.ID == 0 {
		return "", &notifications.Failure{Code: "TELEGRAM_REJECTED"}
	}
	return fmt.Sprint(v.Result.ID), nil
}
