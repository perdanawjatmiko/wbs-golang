package telegram

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"wbs-golang/backend/internal/config"
	"wbs-golang/backend/internal/modules/notifications"
)

func TestTelegramLocalServer(t *testing.T) {
	for _, test := range []struct {
		status int
		body   string
		retry  bool
		after  time.Duration
	}{{200, `{"ok":true,"result":{"message_id":42}}`, false, 0}, {429, `{"ok":false,"error_code":429,"parameters":{"retry_after":120}}`, true, 120 * time.Second}, {503, `{"ok":false}`, true, 0}, {401, `{"ok":false}`, false, 0}, {200, `{"ok":true}`, false, 0}, {502, `not json`, true, 0}} {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			if r.Method != "POST" || !strings.Contains(string(body), `"chat_id":"-100"`) {
				t.Error("invalid request")
			}
			w.WriteHeader(test.status)
			fmt.Fprint(w, test.body)
		}))
		c := New(config.TelegramConfig{Enabled: true, Token: "123:fake", ChatID: "-100", Timeout: time.Second})
		c.base = server.URL
		c.http = server.Client()
		id, err := c.Send(context.Background(), notifications.Message{Text: "safe operational alert"})
		if test.status == 200 && strings.Contains(test.body, `42`) {
			if err != nil || id != "42" {
				t.Fatal(err)
			}
		} else {
			if err == nil {
				t.Fatal("expected failure")
			}
			f := notifications.Classify(err)
			if f.Retryable != test.retry || f.RetryAfter != test.after || strings.Contains(f.Error(), "fake") {
				t.Fatal(f)
			}
		}
		server.Close()
	}
	c := New(config.TelegramConfig{})
	if _, err := c.Send(context.Background(), notifications.Message{}); notifications.Classify(err).Code != "CHANNEL_DISABLED" {
		t.Fatal(err)
	}
}
