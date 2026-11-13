package reports

import (
	"context"
	"github.com/gin-gonic/gin"
	"io"
	"log/slog"
	"mime"
	"net/url"
	"strconv"
	"time"
	"wbs-golang/backend/internal/modules/auth"
	"wbs-golang/backend/internal/platform/database"
	"wbs-golang/backend/internal/platform/filestore"
	"wbs-golang/backend/internal/platform/response"
)

type AdminHandler struct {
	s       *AdminService
	auth    *auth.Handler
	zone    *time.Location
	timeout time.Duration
	log     *slog.Logger
}

func NewAdminHandler(s *AdminService, a *auth.Handler, zone *time.Location, timeout time.Duration, log *slog.Logger) *AdminHandler {
	return &AdminHandler{s: s, auth: a, zone: zone, timeout: timeout, log: log}
}
func (h *AdminHandler) Register(r *gin.Engine) {
	g := r.Group("/api/v1/admin", h.auth.Authenticate(), h.auth.CSRF(), func(c *gin.Context) {
		if len(c.Request.URL.RawQuery) > 4096 {
			h.auth.Fail(c, validation("query", "Query terlalu panjang."))
			return
		}
		q, err := url.ParseQuery(c.Request.URL.RawQuery)
		if err != nil {
			h.auth.Fail(c, validation("query", "Query tidak valid."))
			return
		}
		for _, values := range q {
			if len(values) > 1 {
				h.auth.Fail(c, validation("query", "Parameter tidak boleh berulang."))
				return
			}
		}
		c.Next()
	})
	view := h.auth.Require("report.view", false)
	update := h.auth.Require("report.update", false)
	g.GET("/dashboard", view, func(c *gin.Context) {
		limit, err := integer(c.Request.URL.Query(), "recent_limit", 10, 50)
		if err != nil {
			h.auth.Fail(c, err)
			return
		}
		data, err := h.s.Dashboard(c.Request.Context(), limit)
		h.reply(c, data, err)
	})
	g.GET("/reports", view, func(c *gin.Context) {
		q, err := ParseAdminQuery(c.Request.URL.Query(), h.zone)
		if err != nil {
			h.auth.Fail(c, err)
			return
		}
		data, err := h.s.List(c.Request.Context(), q)
		h.reply(c, data, err)
	})
	g.GET("/reports/options", view, func(c *gin.Context) {
		page, err := integer(c.Request.URL.Query(), "page", 1, 100000)
		if err != nil {
			h.auth.Fail(c, err)
			return
		}
		size, err := integer(c.Request.URL.Query(), "per_page", 100, 100)
		if err != nil {
			h.auth.Fail(c, err)
			return
		}
		data, err := h.s.Options(c.Request.Context(), page, size)
		h.reply(c, data, err)
	})
	g.GET("/reports/:id", view, func(c *gin.Context) {
		page, err := integer(c.Request.URL.Query(), "history_page", 1, 100000)
		if err != nil {
			h.auth.Fail(c, err)
			return
		}
		size, err := integer(c.Request.URL.Query(), "history_per_page", 50, 100)
		if err != nil {
			h.auth.Fail(c, err)
			return
		}
		data, err := h.s.Detail(c.Request.Context(), c.Param("id"), page, size)
		h.reply(c, data, err)
	})
	g.GET("/reports/:id/attachment", view, h.Attachment)
	g.PATCH("/reports/:id/status", update, func(c *gin.Context) {
		var input Mutation
		if err := auth.Decode(c, &input); err != nil {
			h.auth.Fail(c, err)
			return
		}
		data, err := h.s.Mutate(c.Request.Context(), auth.Current(c), c.Param("id"), input, false)
		h.reply(c, data, err)
	})
	g.POST("/reports/:id/notes", update, func(c *gin.Context) {
		var input NotesRequest
		if err := auth.Decode(c, &input); err != nil {
			h.auth.Fail(c, err)
			return
		}
		data, err := h.s.Mutate(c.Request.Context(), auth.Current(c), c.Param("id"), Mutation{InternalNote: input.InternalNote, ExternalNote: input.ExternalNote, ExpectedUpdatedAt: input.ExpectedUpdatedAt}, true)
		h.reply(c, data, err)
	})
}
func (h *AdminHandler) reply(c *gin.Context, data any, err error) {
	if err != nil {
		h.auth.Fail(c, err)
		return
	}
	response.Success(c, 200, data)
}
func (h *AdminHandler) Attachment(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()
	file, err := h.s.Attachment(ctx, c.Param("id"))
	if err != nil {
		h.auth.Fail(c, err)
		return
	}
	defer file.Body.Close()
	c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": file.Name}))
	c.Header("Content-Type", file.ContentType)
	c.Header("Content-Length", strconv.FormatInt(file.Size, 10))
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Cache-Control", "no-store, private")
	c.Status(200)
	n, err := filestore.Copy(ctx, c.Writer, file)
	if err == nil && n != file.Size {
		err = io.ErrUnexpectedEOF
	}
	if err != nil {
		h.log.Warn("attachment stream interrupted", "request_id", c.GetString("request_id"), "error_class", database.FailureClass(err))
	}
}
