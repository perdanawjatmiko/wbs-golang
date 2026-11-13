package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"github.com/gin-gonic/gin"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"
	"wbs-golang/backend/internal/platform/response"
)

var validID = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if !validID.MatchString(id) {
			b := make([]byte, 16)
			if _, err := rand.Read(b); err != nil {
				panic("request ID generation failed")
			}
			id = hex.EncodeToString(b)
		}
		c.Set("request_id", id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}
func Logging(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		log.Info("http_request", "request_id", c.GetString("request_id"), "method", c.Request.Method, "route", route, "status", c.Writer.Status(), "duration", time.Since(start).String())
	}
}
func Recovery(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recover() != nil {
				log.Error("request panic recovered", "request_id", c.GetString("request_id"))
				if !c.Writer.Written() {
					response.Error(c, 500, "INTERNAL_ERROR", "Terjadi kesalahan internal.")
				} else {
					c.Abort()
				}
			}
		}()
		c.Next()
	}
}
func Security(production bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("Cache-Control", "no-store")
		if production {
			c.Header("Strict-Transport-Security", "max-age=31536000")
		}
		c.Next()
	}
}
func BodyLimit(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > limit {
			response.Error(c, 413, "BODY_TOO_LARGE", "Ukuran request terlalu besar.")
			return
		}
		// Buffer only up to the limit so chunked bodies are also rejected before handlers run.
		if c.Request.Body != nil {
			body, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, limit))
			_ = c.Request.Body.Close()
			if err != nil {
				if _, ok := err.(*http.MaxBytesError); ok {
					response.Error(c, 413, "BODY_TOO_LARGE", "Ukuran request terlalu besar.")
				} else {
					response.Error(c, 400, "INVALID_BODY", "Request tidak valid.")
				}
				return
			}
			c.Request.Body = io.NopCloser(strings.NewReader(string(body)))
		}
		c.Next()
	}
}
func CORS(origins []string) gin.HandlerFunc {
	allowed := map[string]bool{}
	for _, o := range origins {
		allowed[o] = true
	}
	return func(c *gin.Context) {
		c.Writer.Header().Add("Vary", "Origin")
		origin := c.GetHeader("Origin")
		if origin != "" {
			if !allowed[origin] {
				response.Error(c, 403, "ORIGIN_FORBIDDEN", "Origin tidak diizinkan.")
				return
			}
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Expose-Headers", "X-Request-ID")
		}
		if c.Request.Method == http.MethodOptions && c.GetHeader("Access-Control-Request-Method") != "" {
			c.Writer.Header().Add("Vary", "Access-Control-Request-Method")
			c.Writer.Header().Add("Vary", "Access-Control-Request-Headers")
			method := c.GetHeader("Access-Control-Request-Method")
			if method != "GET" && method != "HEAD" && method != "OPTIONS" {
				response.Error(c, 405, "METHOD_NOT_ALLOWED", "Metode tidak diizinkan.")
				return
			}
			for _, h := range strings.Split(c.GetHeader("Access-Control-Request-Headers"), ",") {
				h = strings.ToLower(strings.TrimSpace(h))
				if h != "" && h != "content-type" && h != "x-request-id" {
					response.Error(c, 403, "HEADER_FORBIDDEN", "Header tidak diizinkan.")
					return
				}
			}
			c.Header("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, X-Request-ID")
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
