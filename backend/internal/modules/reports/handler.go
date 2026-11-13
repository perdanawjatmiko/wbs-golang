package reports

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strconv"
	"time"
	"wbs-golang/backend/internal/config"
	"wbs-golang/backend/internal/modules/auth"
	"wbs-golang/backend/internal/platform/database"
	"wbs-golang/backend/internal/platform/filestore"
	"wbs-golang/backend/internal/platform/problem"
	"wbs-golang/backend/internal/platform/ratelimit"
	"wbs-golang/backend/internal/platform/response"
)

type Handler struct {
	s             *Service
	files         filestore.Store
	config        config.ReportsConfig
	log           *slog.Logger
	submit, track *ratelimit.Limiter
	slots         chan struct{}
}

func NewHandler(s *Service, files filestore.Store, c config.ReportsConfig, log *slog.Logger) *Handler {
	return &Handler{s: s, files: files, config: c, log: log, submit: ratelimit.NewLimiter(c.SubmitLimit, c.RateWindow, 10000), track: ratelimit.NewLimiter(c.TrackLimit, c.RateWindow, 10000), slots: make(chan struct{}, c.UploadConcurrency)}
}
func (h *Handler) fail(c *gin.Context, err error) {
	var oversized *http.MaxBytesError
	if errors.As(err, &oversized) {
		response.Error(c, 413, "BODY_TOO_LARGE", "Ukuran request terlalu besar.")
		return
	}
	p := problem.Map(err)
	if p.Status == 500 {
		h.log.Error("public report request failed", "request_id", c.GetString("request_id"), "error_class", database.FailureClass(err))
	}
	response.Error(c, p.Status, p.Code, p.Message)
}
func (h *Handler) limit(l *ratelimit.Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.Allow(c.ClientIP(), time.Now()) {
			c.Header("Retry-After", strconv.Itoa(int(h.config.RateWindow.Seconds())))
			response.Error(c, 429, "RATE_LIMITED", "Terlalu banyak request. Coba kembali nanti.")
			return
		}
		c.Next()
	}
}
func (h *Handler) Register(r *gin.Engine) {
	g := r.Group("/api/v1/public")
	g.POST("/reports", h.limit(h.submit), h.Submit)
	g.POST("/reports/track", h.limit(h.track), h.Track)
	// Non-sensitive technical policy keeps client feedback in sync with configurable upload limits.
	g.GET("/reports/options", func(c *gin.Context) {
		response.Success(c, 200, gin.H{"max_file_bytes": h.config.MaxFileBytes, "max_request_bytes": h.config.MaxRequestBytes, "allowed_types": h.config.AllowedTypes, "max_files": h.config.MaxFiles})
	})
}
func (h *Handler) Submit(c *gin.Context) {
	select {
	case h.slots <- struct{}{}:
		defer func() { <-h.slots }()
	default:
		c.Header("Retry-After", "5")
		response.Error(c, 429, "UPLOAD_BUSY", "Upload sedang sibuk. Coba kembali nanti.")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.config.RequestTimeout)
	defer cancel()
	c.Request = c.Request.WithContext(ctx)
	input, file, err := h.multipart(c)
	if err != nil {
		h.fail(c, err)
		return
	}
	result, err := h.s.Submit(c.Request.Context(), input, file)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, 201, result)
}
func (h *Handler) multipart(c *gin.Context) (Submission, *filestore.File, error) {
	var input Submission
	var file *filestore.File
	success := false
	defer func() {
		if !success {
			h.s.Discard(file)
		}
	}()
	if c.Request.ContentLength > h.config.MaxRequestBytes {
		return input, nil, problem.New(413, "BODY_TOO_LARGE", "Ukuran request terlalu besar.")
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.config.MaxRequestBytes)
	reader, err := c.Request.MultipartReader()
	if err != nil {
		return input, nil, problem.New(415, "MULTIPART_REQUIRED", "Gunakan multipart/form-data.")
	}
	seen := map[string]bool{}
	fields := map[string]*string{"name": &input.Name, "contact": &input.Contact, "contact_type": &input.ContactType, "report_type_id": &input.ReportTypeID, "detail": &input.Detail}
	for count := 0; ; count++ {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			var max *http.MaxBytesError
			if errors.As(err, &max) {
				return input, file, err
			}
			return input, file, problem.New(400, "INVALID_MULTIPART", "Multipart request tidak valid.")
		}
		if count >= 6 {
			return input, file, validation("request", "Terlalu banyak field atau attachment.")
		}
		name := part.FormName()
		if seen[name] {
			return input, file, validation(name, "Field tidak boleh berulang.")
		}
		seen[name] = true
		_, params, err := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
		if err != nil {
			return input, file, validation("request", "Header multipart tidak valid.")
		}
		if name == "attachment" {
			if h.config.MaxFiles == 0 {
				return input, file, validation("attachment", "Upload dinonaktifkan.")
			}
			if err = filestore.ValidateFilename(params["filename"]); err != nil {
				return input, file, err
			}
			staged, err := h.files.Stage(c.Request.Context(), params["filename"], part.Header.Get("Content-Type"), part)
			if err != nil {
				return input, file, err
			}
			file = &staged
		} else {
			field, ok := fields[name]
			if !ok || params["filename"] != "" {
				return input, file, validation("request", "Field tidak dikenal.")
			}
			max := int64(1024)
			if name == "detail" {
				max = 80000
			}
			value, err := io.ReadAll(io.LimitReader(part, max+1))
			if err != nil {
				return input, file, err
			}
			if int64(len(value)) > max {
				return input, file, validation(name, "Field terlalu panjang.")
			}
			*field = string(value)
		}
		if err = part.Close(); err != nil {
			return input, file, err
		}
	}
	success = true
	return input, file, nil
}
func (h *Handler) Track(c *gin.Context) {
	var input TrackRequest
	if err := auth.Decode(c, &input); err != nil {
		h.fail(c, err)
		return
	}
	result, err := h.s.Track(c.Request.Context(), input)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, 200, result)
}
