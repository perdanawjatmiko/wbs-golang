package config

import "testing"

func TestNotificationConfiguration(t *testing.T) {
	for _, values := range []map[string]string{{"SMTP_ENABLED": "yes"}, {"SMTP_ENABLED": "true"}, {"SMTP_TLS_MODE": "none"}, {"TELEGRAM_ENABLED": "true", "TELEGRAM_BOT_TOKEN": "secret"}, {"NOTIFICATION_WORKER_CONCURRENCY": "100"}, {"NOTIFICATION_WORKER_LEASE_DURATION": "20s"}, {"NOTIFICATION_RETRY_INTERVALS": "0s"}, {"SSE_REPLAY_LIMIT": "1001"}} {
		values["DATABASE_URL"] = "postgres://qa@localhost/test"
		if _, e := Parse(func(k string) string { return values[k] }); e == nil {
			t.Fatalf("accepted bad config keys %v", values)
		}
	}
	v := map[string]string{"DATABASE_URL": "postgres://qa@localhost/test", "SMTP_ENABLED": "true", "SMTP_HOST": "localhost", "SMTP_FROM_ADDRESS": "wbs@example.test", "TELEGRAM_ENABLED": "true", "TELEGRAM_BOT_TOKEN": "123:fake_token", "TELEGRAM_CHAT_ID": "-100123"}
	c, e := Parse(func(k string) string { return v[k] })
	if e != nil || !c.Notifications.SMTP.Enabled || len(c.Notifications.Retry) != 5 {
		t.Fatal(e)
	}
}
