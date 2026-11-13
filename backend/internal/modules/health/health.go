package health

import (
	"context"
	"github.com/gin-gonic/gin"
	"log/slog"
	"time"
	"wbs-golang/backend/internal/platform/database"
	"wbs-golang/backend/internal/platform/response"
)

// Pinger is the actual dependency used by readiness, implemented by pgxpool.Pool.
type Pinger interface{ Ping(context.Context) error }
type Service struct {
	db      Pinger
	timeout time.Duration
}

func NewService(db Pinger, timeout time.Duration) *Service { return &Service{db, timeout} }
func (s *Service) Ready(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	return s.db.Ping(ctx)
}

type Handler struct {
	service            *Service
	log                *slog.Logger
	name, env, version string
}

func NewHandler(s *Service, l *slog.Logger, name, env, version string) *Handler {
	return &Handler{s, l, name, env, version}
}
func (h *Handler) Live(c *gin.Context) {
	response.Success(c, 200, gin.H{"status": "ok", "service": h.name})
}
func (h *Handler) Ready(c *gin.Context) {
	if err := h.service.Ready(c.Request.Context()); err != nil {
		h.log.Error("database readiness failed", "request_id", c.GetString("request_id"), "error_class", database.FailureClass(err))
		response.Error(c, 503, "NOT_READY", "Layanan belum siap.")
		return
	}
	response.Success(c, 200, gin.H{"status": "ready"})
}
func (h *Handler) Info(c *gin.Context) {
	response.Success(c, 200, gin.H{"app_name": h.name, "environment": h.env, "version": h.version})
}
