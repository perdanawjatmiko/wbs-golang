package httpserver

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"log/slog"
	"net"
	"net/http"
	"time"
	"wbs-golang/backend/internal/config"
	"wbs-golang/backend/internal/middleware"
	"wbs-golang/backend/internal/modules/health"
	"wbs-golang/backend/internal/platform/response"
)

func Router(c config.Config, log *slog.Logger, h *health.Handler) *gin.Engine {
	r := gin.New()
	_ = r.SetTrustedProxies(nil)
	r.HandleMethodNotAllowed = true
	r.Use(middleware.RequestID(), middleware.Logging(log), middleware.Recovery(log), middleware.Security(c.Env == "production"), middleware.CORS(c.Origins), middleware.BodyLimit(c.BodyLimit))
	r.GET("/health/live", h.Live)
	r.GET("/health/ready", h.Ready)
	// Preserve the original /health payload for existing consumers.
	r.GET("/health", func(ctx *gin.Context) { ctx.JSON(200, gin.H{"status": "ok", "service": c.AppName}) })
	r.Group("/api/v1").GET("/system/info", h.Info)
	r.NoRoute(func(c *gin.Context) { response.Error(c, 404, "NOT_FOUND", "Endpoint tidak ditemukan.") })
	r.NoMethod(func(c *gin.Context) { response.Error(c, 405, "METHOD_NOT_ALLOWED", "Metode tidak diizinkan.") })
	return r
}
func New(c config.Config, h http.Handler) *http.Server {
	return &http.Server{Addr: net.JoinHostPort(c.Host, c.Port), Handler: h, ReadTimeout: c.ReadTimeout, ReadHeaderTimeout: c.ReadTimeout, WriteTimeout: c.WriteTimeout, IdleTimeout: c.IdleTimeout, MaxHeaderBytes: 1 << 20}
}
func Run(ctx context.Context, s *http.Server, timeout time.Duration) error {
	result := make(chan error, 1)
	go func() { result <- s.ListenAndServe() }()
	select {
	case err := <-result:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		err := s.Shutdown(shutdown)
		if err != nil {
			_ = s.Close()
		}
		serveErr := <-result
		if err != nil {
			return err
		}
		if !errors.Is(serveErr, http.ErrServerClosed) {
			return serveErr
		}
		return nil
	}
}
