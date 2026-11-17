package auth

import (
	"crypto/subtle"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
	"wbs-golang/backend/internal/platform/database"
	"wbs-golang/backend/internal/platform/problem"
	"wbs-golang/backend/internal/platform/ratelimit"
	"wbs-golang/backend/internal/platform/response"
	"wbs-golang/backend/internal/platform/security"
)

const CookieName = "wbs_session"

type Handler struct {
	s           *Service
	log         *slog.Logger
	secure      bool
	ttl         time.Duration
	origins     map[string]bool
	limiter     *Limiter
	gate        ratelimit.Gate
	loginLimit  int
	loginWindow time.Duration
}

func NewHandler(s *Service, log *slog.Logger, secure bool, ttl time.Duration, origins []string) *Handler {
	allowed := map[string]bool{}
	for _, o := range origins {
		allowed[o] = true
	}
	return &Handler{s: s, log: log, secure: secure, ttl: ttl, origins: allowed, limiter: NewLimiter(10, 15*time.Minute, 10000), loginLimit: 10, loginWindow: 15 * time.Minute}
}

// UseRateGate is composition-time injection; API always supplies the PostgreSQL gate.
func (h *Handler) UseRateGate(g ratelimit.Gate, limit int, window time.Duration) *Handler {
	h.gate = g
	h.loginLimit = limit
	h.loginWindow = window
	return h
}
func (h *Handler) Fail(c *gin.Context, err error) {
	p := problem.Map(err)
	if p.Status == 500 {
		h.log.Error("request operation failed", "request_id", c.GetString("request_id"), "error_class", database.FailureClass(err))
	}
	response.Error(c, p.Status, p.Code, p.Message)
}
func Decode(c *gin.Context, dst any) error {
	if strings.Split(c.GetHeader("Content-Type"), ";")[0] != "application/json" {
		return problem.New(415, "JSON_REQUIRED", "Gunakan application/json.")
	}
	d := json.NewDecoder(c.Request.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(dst); err != nil {
		return problem.New(422, "VALIDATION_ERROR", "JSON tidak valid.")
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return problem.New(422, "VALIDATION_ERROR", "JSON tidak valid.")
	}
	return nil
}
func (h *Handler) Origin(c *gin.Context) bool {
	if !h.origins[c.GetHeader("Origin")] {
		h.Fail(c, problem.New(403, "CSRF_REJECTED", "Origin request tidak valid."))
		return false
	}
	return true
}
func (h *Handler) cookie(c *gin.Context, value string, maxAge int) {
	http.SetCookie(c.Writer, &http.Cookie{Name: CookieName, Value: value, Path: "/", MaxAge: maxAge, HttpOnly: true, Secure: h.secure, SameSite: http.SameSiteLaxMode})
}
func (h *Handler) Login(c *gin.Context) {
	if !h.Origin(c) {
		return
	}
	allowed, retry := true, 900
	if h.gate != nil {
		d, err := h.gate.Check(c.Request.Context(), "admin_login", c.ClientIP(), h.loginLimit, h.loginWindow)
		if err != nil {
			h.Fail(c, problem.New(503, "RATE_LIMIT_UNAVAILABLE", "Layanan pembatasan request belum tersedia."))
			return
		}
		allowed, retry = d.Allowed, d.RetryAfter
	} else {
		allowed = h.limiter.Allow(c.ClientIP(), time.Now())
	}
	if !allowed {
		c.Header("Retry-After", strconv.Itoa(retry))
		response.Error(c, 429, "RATE_LIMITED", "Terlalu banyak percobaan login. Coba lagi nanti.")
		return
	}
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := Decode(c, &req); err != nil {
		h.Fail(c, err)
		return
	}
	if len(req.Email) > 254 || len(req.Password) > 72 || req.Email == "" || req.Password == "" {
		h.Fail(c, problem.New(401, "INVALID_CREDENTIALS", "Email atau password tidak valid."))
		return
	}
	old, _ := c.Cookie(CookieName)
	token, p, err := h.s.Login(c.Request.Context(), req.Email, req.Password, old)
	if err != nil {
		h.Fail(c, err)
		return
	}
	h.cookie(c, token, int(h.ttl.Seconds()))
	response.Success(c, 200, gin.H{"user": p, "csrf_token": security.CSRF(token)})
}
func (h *Handler) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, _ := c.Cookie(CookieName)
		p, err := h.s.Authenticate(c.Request.Context(), token)
		if err != nil {
			h.Fail(c, err)
			return
		}
		c.Set("principal", p)
		if p.MustChangePassword && c.Request.URL.Path != "/api/v1/auth/change-password" && c.Request.URL.Path != "/api/v1/auth/me" && c.Request.URL.Path != "/api/v1/auth/logout" {
			h.Fail(c, problem.New(403, "PASSWORD_CHANGE_REQUIRED", "Ubah password sebelum menggunakan aplikasi."))
			return
		}
		c.Next()
	}
}
func Current(c *gin.Context) Principal {
	p, _ := c.Get("principal")
	value, _ := p.(Principal)
	return value
}
func (h *Handler) CSRF() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}
		if !h.Origin(c) {
			return
		}
		token, _ := c.Cookie(CookieName)
		if token == "" || subtle.ConstantTimeCompare([]byte(security.CSRF(token)), []byte(c.GetHeader("X-CSRF-Token"))) != 1 {
			h.Fail(c, problem.New(403, "CSRF_REJECTED", "Token CSRF tidak valid."))
			return
		}
		c.Next()
	}
}
func (h *Handler) Require(permission string, superOnly bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		p := Current(c)
		if (superOnly && !p.Super()) || !p.Can(permission) {
			h.Fail(c, problem.New(403, "FORBIDDEN", "Akses tidak diizinkan."))
			return
		}
		c.Next()
	}
}
func (h *Handler) Me(c *gin.Context) {
	token, _ := c.Cookie(CookieName)
	response.Success(c, 200, gin.H{"user": Current(c), "csrf_token": security.CSRF(token)})
}
func (h *Handler) Logout(c *gin.Context) {
	token, _ := c.Cookie(CookieName)
	if err := h.s.Logout(c.Request.Context(), token); err != nil {
		h.Fail(c, err)
		return
	}
	h.cookie(c, "", -1)
	response.Success(c, 200, gin.H{"status": "logged_out"})
}
func (h *Handler) ChangePassword(c *gin.Context) {
	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := Decode(c, &req); err != nil {
		h.Fail(c, err)
		return
	}
	if len(req.CurrentPassword) > 72 || len(req.NewPassword) > 72 {
		h.Fail(c, problem.New(422, "VALIDATION_ERROR", "Password tidak valid."))
		return
	}
	if err := h.s.ChangePassword(c.Request.Context(), h.s.repo.(PasswordChanger), Current(c).ID, req.CurrentPassword, req.NewPassword); err != nil {
		h.Fail(c, err)
		return
	}
	h.cookie(c, "", -1)
	response.Success(c, 200, gin.H{"status": "password_changed", "login_required": true})
}
func (h *Handler) Register(r *gin.Engine) {
	g := r.Group("/api/v1/auth")
	g.POST("/login", h.Login)
	g.GET("/me", h.Authenticate(), h.Me)
	g.POST("/logout", h.Authenticate(), h.CSRF(), h.Logout)
	g.POST("/change-password", h.Authenticate(), h.CSRF(), h.ChangePassword)
}

// Compatibility aliases retained for existing auth tests/callers.
type Limiter = ratelimit.Limiter

func NewLimiter(max int, window time.Duration, capacity int) *Limiter {
	return ratelimit.NewLimiter(max, window, capacity)
}
