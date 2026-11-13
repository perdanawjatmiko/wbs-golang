package master

import (
	"github.com/gin-gonic/gin"
	"strconv"
	"strings"
	"wbs-golang/backend/internal/modules/auth"
	"wbs-golang/backend/internal/platform/response"
	"wbs-golang/backend/internal/platform/security"
)

type Handler struct {
	s    *Service
	auth *auth.Handler
}

func NewHandler(s *Service, a *auth.Handler) *Handler { return &Handler{s, a} }
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
		g.GET("", func(c *gin.Context) { h.list(c, e, false) })
		g.GET("/:id", func(c *gin.Context) {
			v, err := h.s.Get(c.Request.Context(), e, c.Param("id"))
			if err != nil {
				h.auth.Fail(c, err)
				return
			}
			response.Success(c, 200, v)
		})
		g.POST("", func(c *gin.Context) { h.save(c, e, false) })
		g.PATCH("/:id", func(c *gin.Context) { h.save(c, e, false) })
		g.DELETE("/:id", func(c *gin.Context) { h.save(c, e, true) })
	}
	for _, path := range []string{"report-types", "faqs"} {
		e, _ := Find(path)
		r.GET("/api/v1/public/"+path, func(c *gin.Context) { h.list(c, e, true) })
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
