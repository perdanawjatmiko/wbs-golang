package master

import (
	"github.com/gin-gonic/gin"
	"strconv"
	"strings"
	"time"
	"wbs-golang/backend/internal/modules/auth"
	"wbs-golang/backend/internal/platform/problem"
	"wbs-golang/backend/internal/platform/ratelimit"
	"wbs-golang/backend/internal/platform/response"
	"wbs-golang/backend/internal/platform/security"
)

type Handler struct {
	s    *Service
	auth *auth.Handler
	gate ratelimit.Gate
}

func NewHandler(s *Service, a *auth.Handler) *Handler    { return &Handler{s: s, auth: a} }
func (h *Handler) UseRateGate(g ratelimit.Gate) *Handler { h.gate = g; return h }
func parseQuery(c *gin.Context, public bool) (Query, error) {
	q := Query{Page: 1, Size: 20, Search: strings.TrimSpace(c.Query("search")), Active: c.Query("is_active"), ReportType: c.Query("report_type_id"), Public: public}
	var err error
	if v := c.Query("page"); v != "" {
		q.Page, err = strconv.Atoi(v)
		if err != nil {
			return q, invalid("page tidak valid.")
		}
	}
	if v := c.Query("page_size"); v != "" {
		q.Size, err = strconv.Atoi(v)
		if err != nil {
			return q, invalid("page_size tidak valid.")
		}
	}
	if q.Page < 1 || q.Page > 100000 || q.Size < 1 || q.Size > 100 || len(q.Search) > 200 || (q.Active != "" && q.Active != "true" && q.Active != "false") || (q.ReportType != "" && q.ReportType != "general" && !security.UUID.MatchString(q.ReportType)) {
		return q, invalid("Filter atau pagination tidak valid.")
	}
	return q, nil
}
func (h *Handler) Register(r *gin.Engine) {
	for _, e := range entities {
		g := r.Group("/api/v1/admin/"+e.Path, h.auth.Authenticate(), h.auth.CSRF(), func(c *gin.Context) {
			if err := authorize(auth.Current(c), e); err != nil {
				h.auth.Fail(c, err)
				return
			}
			c.Next()
		})
		importEntity := e
		if importEntity.Path == "users" {
			importEntity.Path = "admin-users"
		}
		if _, supported := importColumns[importEntity.Path]; supported {
			g.GET("/template", func(c *gin.Context) { h.export(c, importEntity, true) })
			g.GET("/export", func(c *gin.Context) { h.export(c, importEntity, false) })
			g.POST("/import/preview", h.limitImport("master_import_preview"), func(c *gin.Context) { h.previewImport(c, importEntity) })
			g.POST("/import/commit", h.limitImport("master_import_commit"), func(c *gin.Context) { h.commitImport(c, importEntity) })
		}
		g.GET("", func(c *gin.Context) { h.list(c, e, false) })
		g.GET("/:id", func(c *gin.Context) {
			v, err := h.s.Get(c.Request.Context(), e, c.Param("id"))
			if err != nil {
				h.auth.Fail(c, err)
				return
			}
			response.Success(c, 200, v)
		})
		write := gin.HandlerFunc(func(c *gin.Context) { h.save(c, e, false) })
		remove := gin.HandlerFunc(func(c *gin.Context) { h.save(c, e, true) })
		if e.Path == "users" {
			g.POST("", h.limitImport("master_user_admin"), write)
			g.PATCH("/:id", h.limitImport("master_user_admin"), write)
			g.DELETE("/:id", h.limitImport("master_user_admin"), remove)
		} else {
			g.POST("", write)
			g.PATCH("/:id", write)
			g.DELETE("/:id", remove)
		}
	}
	for _, path := range []string{"report-types", "faqs"} {
		e, _ := Find(path)
		r.GET("/api/v1/public/"+path, func(c *gin.Context) { h.list(c, e, true) })
	}
}

func (h *Handler) limitImport(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.gate == nil {
			c.Next()
			return
		}
		decision, err := h.gate.Check(c.Request.Context(), scope, c.ClientIP(), 10, 15*time.Minute)
		if err != nil {
			h.auth.Fail(c, problem.New(503, "RATE_LIMIT_UNAVAILABLE", "Layanan pembatasan request belum tersedia."))
			return
		}
		if !decision.Allowed {
			c.Header("Retry-After", strconv.Itoa(decision.RetryAfter))
			response.Error(c, 429, "RATE_LIMITED", "Terlalu banyak request administratif. Coba kembali nanti.")
			return
		}
		c.Next()
	}
}
func (h *Handler) list(c *gin.Context, e Entity, public bool) {
	q, err := parseQuery(c, public)
	if err != nil {
		h.auth.Fail(c, err)
		return
	}
	v, err := h.s.List(c.Request.Context(), e, q)
	if err != nil {
		h.auth.Fail(c, err)
		return
	}
	response.Success(c, 200, v)
}
func (h *Handler) save(c *gin.Context, e Entity, remove bool) {
	var input Input
	if !remove {
		if err := auth.Decode(c, &input); err != nil {
			h.auth.Fail(c, err)
			return
		}
	}
	v, err := h.s.Mutate(c.Request.Context(), auth.Current(c), e, c.Param("id"), input, remove)
	if err != nil {
		h.auth.Fail(c, err)
		return
	}
	status := 200
	if c.Request.Method == "POST" {
		status = 201
	}
	response.Success(c, status, v)
}
