package notifications

import (
	"context"
	"github.com/gin-gonic/gin"
	"net/url"
	"strconv"
	"sync/atomic"
	"wbs-golang/backend/internal/config"
	"wbs-golang/backend/internal/modules/auth"
	"wbs-golang/backend/internal/platform/response"
	"wbs-golang/backend/internal/platform/security"
)

type Handler struct {
	s           *Service
	a           *auth.Handler
	config      config.NotificationsConfig
	shutdown    context.Context
	connections atomic.Int64
}

func NewHandler(s *Service, a *auth.Handler, c config.NotificationsConfig, shutdown context.Context) *Handler {
	return &Handler{s: s, a: a, config: c, shutdown: shutdown}
}
func (h *Handler) Register(r *gin.Engine) {
	g := r.Group("/api/v1/admin", h.a.Authenticate(), h.a.CSRF())
	g.GET("/notifications", h.a.Require("report.view", false), h.inbox)
	g.GET("/notifications/unread-count", h.a.Require("report.view", false), h.unread)
	g.POST("/notifications/read-all", h.a.Require("report.view", false), h.read)
	g.POST("/notifications/:id/read", h.a.Require("report.view", false), h.read)
	g.GET("/notifications/stream", h.a.Require("report.view", false), h.stream)
	g.GET("/notification-deliveries", h.a.Require("notifications.manage", false), h.deliveries)
	g.GET("/notification-deliveries/:id", h.a.Require("notifications.manage", false), h.deliveries)
	g.POST("/notification-deliveries/:id/retry", h.a.Require("notifications.manage", false), h.retry)
}
func (h *Handler) query(c *gin.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.Request.Context(), h.s.timeout)
}
func (h *Handler) pagination(c *gin.Context) (int, int, bool) {
	values, e := url.ParseQuery(c.Request.URL.RawQuery)
	if e != nil {
		e = invalid()
	}
	if len(c.Request.URL.RawQuery) > 1024 {
		e = invalid()
	}
	for k, v := range values {
		if len(v) != 1 || (k != "page" && k != "per_page" && k != "channel" && k != "status") {
			e = invalid()
		}
	}
	p, size, err2 := ParsePage(c.Query("page"), c.Query("per_page"))
	if err2 != nil {
		e = err2
	}
	if e != nil {
		h.a.Fail(c, e)
		return 0, 0, false
	}
	return p, size, true
}
func (h *Handler) inbox(c *gin.Context) {
	p, size, ok := h.pagination(c)
	if !ok {
		return
	}
	ctx, cancel := h.query(c)
	defer cancel()
	v, err := h.s.Inbox(ctx, auth.Current(c).ID, p, size)
	if err != nil {
		h.a.Fail(c, err)
		return
	}
	response.Success(c, 200, v)
}
func (h *Handler) unread(c *gin.Context) {
	ctx, cancel := h.query(c)
	defer cancel()
	v, err := h.s.Unread(ctx, auth.Current(c).ID)
	if err != nil {
		h.a.Fail(c, err)
		return
	}
	response.Success(c, 200, gin.H{"unread_count": v})
}
func (h *Handler) read(c *gin.Context) {
	id := c.Param("id")
	if id != "" && !security.UUID.MatchString(id) {
		h.a.Fail(c, invalid())
		return
	}
	ctx, cancel := h.query(c)
	defer cancel()
	if err := h.s.MarkRead(ctx, auth.Current(c).ID, id); err != nil {
		h.a.Fail(c, err)
		return
	}
	response.Success(c, 200, gin.H{"status": "ok"})
}
func (h *Handler) deliveries(c *gin.Context) {
	p, size, ok := h.pagination(c)
	if !ok {
		return
	}
	channel, status, id := c.Query("channel"), c.Query("status"), c.Param("id")
	if (id != "" && !security.UUID.MatchString(id)) || (channel != "" && channel != "EMAIL" && channel != "TELEGRAM") || (status != "" && status != "pending" && status != "processing" && status != "sent" && status != "retrying" && status != "failed") {
		h.a.Fail(c, invalid())
		return
	}
	ctx, cancel := h.query(c)
	defer cancel()
	v, err := h.s.Deliveries(ctx, p, size, channel, status, id)
	if err != nil {
		h.a.Fail(c, err)
		return
	}
	response.Success(c, 200, v)
}
func (h *Handler) retry(c *gin.Context) {
	id := c.Param("id")
	if !security.UUID.MatchString(id) {
		h.a.Fail(c, invalid())
		return
	}
	ctx, cancel := h.query(c)
	defer cancel()
	token, _ := c.Cookie(auth.CookieName)
	// Re-resolve authorization immediately before administrative mutation.
	err := h.s.Retry(ctx, id, token)
	if err != nil {
		h.a.Fail(c, err)
		return
	}
	response.Success(c, 200, gin.H{"status": "retrying"})
}
func parseCursor(raw string) (int64, error) {
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n < 0 || len(raw) > 19 {
		return 0, invalid()
	}
	return n, nil
}
