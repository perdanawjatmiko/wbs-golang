package config

import (
	"strings"
	"testing"
)

func TestValidation(t *testing.T) {
	cases := []struct {
		name, key, value string
		bad              bool
	}{
		{"defaults", "", "", false}, {"port", "HTTP_PORT", "0", true}, {"duration", "HTTP_READ_TIMEOUT", "bad", true}, {"negative", "HTTP_IDLE_TIMEOUT", "-1s", true}, {"pool", "DB_MAX_OPEN_CONNS", "0", true}, {"idle", "DB_MAX_IDLE_CONNS", "21", true}, {"db", "DATABASE_URL", "secret", true}, {"origin", "CORS_ALLOWED_ORIGINS", "*", true}, {"origin path", "CORS_ALLOWED_ORIGINS", "https://example.com/path", true}, {"env", "APP_ENV", "unknown", true}, {"production TLS", "APP_ENV", "production", true}, {"limit", "HTTP_MAX_BODY_BYTES", "0", true}, {"ready bound", "DB_READY_TIMEOUT", "20s", true}, {"query bound", "DB_QUERY_TIMEOUT", "20s", true}, {"short session", "SESSION_TTL", "1s", true}, {"long session", "SESSION_TTL", "48h", true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			m := map[string]string{"DATABASE_URL": "postgres://user:private-password@localhost/db?sslmode=disable"}
			if tt.key != "" {
				m[tt.key] = tt.value
			}
			_, err := Parse(func(k string) string { return m[k] })
			if (err != nil) != tt.bad {
				t.Fatalf("unexpected validation: %v", err)
			}
			if err != nil && strings.Contains(err.Error(), "private-password") {
				t.Fatal("secret leaked")
			}
		})
	}
	t.Run("production valid", func(t *testing.T) {
		m := map[string]string{"APP_ENV": "production", "DATABASE_URL": "postgres://user:pass@db.example/db?sslmode=verify-full", "CORS_ALLOWED_ORIGINS": "https://example.com"}
		if _, err := Parse(func(k string) string { return m[k] }); err != nil {
			t.Fatal(err)
		}
	})
}
