package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
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
	return c, nil
}
