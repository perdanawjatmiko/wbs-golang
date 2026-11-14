package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppName, Env, Host, Port, DatabaseURL                   string
	Origins                                                 []string
	MaxOpen, MaxIdle                                        int
	Lifetime, IdleTime, ConnectTimeout, ReadyTimeout        time.Duration
	ReadTimeout, WriteTimeout, IdleTimeout, ShutdownTimeout time.Duration
	BodyLimit                                               int64
	SessionTTL, QueryTimeout                                time.Duration
	Reports                                                 ReportsConfig
	Notifications                                           NotificationsConfig
}

func Load() (Config, error) { return Parse(os.Getenv) }
func Parse(get func(string) string) (Config, error) {
	c := Config{}
	value := func(k, d string) string {
		if v := get(k); v != "" {
			return v
		}
		return d
	}
	c.AppName = value("APP_NAME", "wbs-backend")
	c.Env = value("APP_ENV", "local")
	c.Host = value("HTTP_HOST", "127.0.0.1")
	c.Port = value("HTTP_PORT", "8080")
	c.DatabaseURL = get("DATABASE_URL")
	if strings.TrimSpace(c.AppName) == "" {
		return c, fmt.Errorf("APP_NAME must not be blank")
	}
	if c.Env != "local" && c.Env != "test" && c.Env != "staging" && c.Env != "production" {
		return c, fmt.Errorf("APP_ENV must be local, test, staging or production")
	}
	port, e := strconv.Atoi(c.Port)
	if e != nil || port < 1 || port > 65535 {
		return c, fmt.Errorf("HTTP_PORT must be 1..65535")
	}
	if net.ParseIP(c.Host) == nil && c.Host != "localhost" {
		return c, fmt.Errorf("HTTP_HOST must be an IP address or localhost")
	}
	u, e := url.Parse(c.DatabaseURL)
	if e != nil || u == nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") || u.Hostname() == "" || u.Path == "" {
		return c, fmt.Errorf("DATABASE_URL must be a PostgreSQL URL with host and database")
	}
	if c.Env == "production" && (u.Query().Get("sslmode") != "verify-full" || u.Query().Get("sslnegotiation") == "direct") {
		return c, fmt.Errorf("production DATABASE_URL requires sslmode=verify-full and standard TLS negotiation")
	}
	for _, origin := range strings.Split(value("CORS_ALLOWED_ORIGINS", "http://localhost:3000"), ",") {
		origin = strings.TrimSpace(origin)
		o, err := url.Parse(origin)
		if err != nil || o == nil || (o.Scheme != "http" && o.Scheme != "https") || o.Host == "" || o.User != nil || o.RawQuery != "" || o.Fragment != "" || o.Path != "" || strings.Contains(origin, "*") {
			return c, fmt.Errorf("CORS_ALLOWED_ORIGINS must contain explicit HTTP origins without paths")
		}
		if c.Env == "production" && o.Scheme != "https" {
			return c, fmt.Errorf("production CORS origins require HTTPS")
		}
		c.Origins = append(c.Origins, origin)
	}
	for _, item := range []struct {
		k, d string
		p    *time.Duration
	}{
		{"SESSION_TTL", "8h", &c.SessionTTL}, {"DB_QUERY_TIMEOUT", "5s", &c.QueryTimeout}, {"DB_CONN_MAX_LIFETIME", "30m", &c.Lifetime}, {"DB_CONN_MAX_IDLE_TIME", "5m", &c.IdleTime}, {"DB_CONNECT_TIMEOUT", "5s", &c.ConnectTimeout}, {"DB_READY_TIMEOUT", "2s", &c.ReadyTimeout},
		{"HTTP_READ_TIMEOUT", "10s", &c.ReadTimeout}, {"HTTP_WRITE_TIMEOUT", "15s", &c.WriteTimeout}, {"HTTP_IDLE_TIMEOUT", "60s", &c.IdleTimeout}, {"HTTP_SHUTDOWN_TIMEOUT", "15s", &c.ShutdownTimeout},
	} {
		v, err := time.ParseDuration(value(item.k, item.d))
		if err != nil || v <= 0 {
			return c, fmt.Errorf("%s must be a positive duration", item.k)
		}
		*item.p = v
	}
	for _, item := range []struct {
		k, d string
		p    *int
	}{{"DB_MAX_OPEN_CONNS", "20", &c.MaxOpen}, {"DB_MAX_IDLE_CONNS", "5", &c.MaxIdle}} {
		v, err := strconv.Atoi(value(item.k, item.d))
		if err != nil || v < 0 || v > 2147483647 {
			return c, fmt.Errorf("%s must be a nonnegative 32-bit integer", item.k)
		}
		*item.p = v
	}
	if c.MaxOpen < 1 || c.MaxIdle > c.MaxOpen {
		return c, fmt.Errorf("pool limits require DB_MAX_OPEN_CONNS >= 1 and DB_MAX_IDLE_CONNS <= DB_MAX_OPEN_CONNS")
	}
	c.BodyLimit, e = strconv.ParseInt(value("HTTP_MAX_BODY_BYTES", "1048576"), 10, 64)
	if e != nil || c.BodyLimit < 1 {
		return c, fmt.Errorf("HTTP_MAX_BODY_BYTES must be positive")
	}
	if c.SessionTTL < time.Minute || c.SessionTTL > 24*time.Hour {
		return c, fmt.Errorf("SESSION_TTL must be between 1m and 24h")
	}
	if c.QueryTimeout >= c.WriteTimeout {
		return c, fmt.Errorf("DB_QUERY_TIMEOUT must be shorter than HTTP_WRITE_TIMEOUT")
	}
	if c.ReadyTimeout >= c.WriteTimeout {
		return c, fmt.Errorf("DB_READY_TIMEOUT must be shorter than HTTP_WRITE_TIMEOUT")
	}
	reports, err := parseReports(get)
	if err != nil {
		return c, err
	}
	if reports.RequestTimeout >= c.WriteTimeout {
		return c, fmt.Errorf("REPORT_REQUEST_TIMEOUT must be shorter than HTTP_WRITE_TIMEOUT")
	}
	c.Reports = reports
	c.Notifications, err = parseNotifications(get)
	if err != nil {
		return c, err
	}
	if c.Notifications.LeaseDuration < max(c.Notifications.SMTP.Timeout, c.Notifications.Telegram.Timeout)+2*c.QueryTimeout+5*time.Second {
		return c, fmt.Errorf("notification lease must cover provider timeout, database queries and a 5s margin")
	}
	return c, nil
}

type ReportsConfig struct {
	Timezone, UploadDir               string
	MaxFileBytes, MaxRequestBytes     int64
	AllowedTypes                      []string
	MaxFiles, SubmitLimit, TrackLimit int
	RateWindow                        time.Duration
	RequestTimeout                    time.Duration
	UploadConcurrency                 int
}

func parseReports(get func(string) string) (ReportsConfig, error) {
	value := func(k, d string) string {
		if v := get(k); v != "" {
			return v
		}
		return d
	}
	r := ReportsConfig{Timezone: value("APP_TIMEZONE", "Asia/Jakarta"), UploadDir: value("UPLOAD_DIR", "./var/private/attachments")}
	if _, err := time.LoadLocation(r.Timezone); err != nil {
		return r, fmt.Errorf("APP_TIMEZONE must be a valid IANA timezone")
	}
	abs, err := filepath.Abs(r.UploadDir)
	if err != nil {
		return r, fmt.Errorf("UPLOAD_DIR is invalid")
	}
	r.UploadDir = abs
	for _, component := range strings.Split(filepath.ToSlash(abs), "/") {
		if component == "public" || component == ".next" || component == "static" {
			return r, fmt.Errorf("UPLOAD_DIR must be outside public web roots")
		}
	}
	for _, f := range []struct {
		k, d string
		p    *int64
	}{{"UPLOAD_MAX_FILE_BYTES", "5242880", &r.MaxFileBytes}, {"UPLOAD_MAX_REQUEST_BYTES", "6291456", &r.MaxRequestBytes}} {
		v, e := strconv.ParseInt(value(f.k, f.d), 10, 64)
		if e != nil || v < 1 || v > 32<<20 {
			return r, fmt.Errorf("%s must be 1..33554432", f.k)
		}
		*f.p = v
	}
	if r.MaxRequestBytes < r.MaxFileBytes+65536 {
		return r, fmt.Errorf("UPLOAD_MAX_REQUEST_BYTES must allow file plus at least 64 KiB overhead")
	}
	for _, f := range []struct {
		k, d string
		p    *int
	}{{"UPLOAD_MAX_FILES", "1", &r.MaxFiles}, {"REPORT_SUBMIT_RATE_LIMIT", "10", &r.SubmitLimit}, {"REPORT_TRACK_RATE_LIMIT", "30", &r.TrackLimit}} {
		v, e := strconv.Atoi(value(f.k, f.d))
		if e != nil || v < 0 || v > 10000 {
			return r, fmt.Errorf("%s is invalid", f.k)
		}
		*f.p = v
	}
	if r.MaxFiles > 1 {
		return r, fmt.Errorf("UPLOAD_MAX_FILES must be 0 or 1")
	}
	if r.SubmitLimit < 1 || r.TrackLimit < 1 {
		return r, fmt.Errorf("report rate limits must be positive")
	}
	r.UploadConcurrency, err = strconv.Atoi(value("REPORT_UPLOAD_CONCURRENCY", "2"))
	if err != nil || r.UploadConcurrency < 1 || r.UploadConcurrency > 16 {
		return r, fmt.Errorf("REPORT_UPLOAD_CONCURRENCY must be 1..16")
	}
	r.RequestTimeout, err = time.ParseDuration(value("REPORT_REQUEST_TIMEOUT", "10s"))
	if err != nil || r.RequestTimeout < time.Second {
		return r, fmt.Errorf("REPORT_REQUEST_TIMEOUT must be >=1s")
	}
	r.RateWindow, err = time.ParseDuration(value("REPORT_RATE_WINDOW", "15m"))
	if err != nil || r.RateWindow < time.Second || r.RateWindow > 24*time.Hour {
		return r, fmt.Errorf("REPORT_RATE_WINDOW must be 1s..24h")
	}
	seen := map[string]bool{}
	for _, kind := range strings.Split(value("UPLOAD_ALLOWED_TYPES", "image/jpeg,image/png,application/pdf"), ",") {
		kind = strings.TrimSpace(kind)
		if kind != "image/jpeg" && kind != "image/png" && kind != "application/pdf" {
			return r, fmt.Errorf("UPLOAD_ALLOWED_TYPES supports JPEG, PNG and PDF only")
		}
		if !seen[kind] {
			r.AllowedTypes = append(r.AllowedTypes, kind)
			seen[kind] = true
		}
	}
	return r, nil
}
