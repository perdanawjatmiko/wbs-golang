package reports

import (
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
	"wbs-golang/backend/internal/platform/security"
)

const MaxNoteLength = 10000

func IsStatus(s string) bool {
	return s == "Open" || s == "On Progress" || s == "Done" || s == "Rejected"
}
func IsActive(s string) bool { return s == "Open" || s == "On Progress" }

type Pagination struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

func pagination(page, size, total int) Pagination {
	return Pagination{page, size, total, (total + size - 1) / size}
}

type AdminQuery struct {
	Page, PerPage                           int
	Search, Status, TypeID, Sort, Direction string
	From, To                                *time.Time
}

func integer(q url.Values, key string, defaultValue, max int) (int, error) {
	v := q.Get(key)
	if v == "" {
		return defaultValue, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 || n > max {
		return 0, validation(key, "Nilai tidak valid.")
	}
	return n, nil
}
func ParseAdminQuery(q url.Values, zone *time.Location) (AdminQuery, error) {
	r := AdminQuery{Search: strings.TrimSpace(q.Get("search")), Status: q.Get("status"), TypeID: q.Get("report_type_id"), Sort: q.Get("sort"), Direction: q.Get("direction")}
	var err error
	r.Page, err = integer(q, "page", 1, 100000)
	if err != nil {
		return r, err
	}
	r.PerPage, err = integer(q, "per_page", 20, 100)
	if err != nil {
		return r, err
	}
	if !utf8.ValidString(r.Search) || strings.ContainsRune(r.Search, 0) || utf8.RuneCountInString(r.Search) > 200 {
		return r, validation("search", "Maksimum 200 karakter.")
	}
	if r.Status != "" && !IsStatus(r.Status) {
		return r, validation("status", "Status tidak valid.")
	}
	if r.TypeID != "" && !security.UUID.MatchString(r.TypeID) {
		return r, validation("report_type_id", "UUID tidak valid.")
	}
	if r.Sort == "" {
		r.Sort = "created_at"
	}
	if r.Sort != "created_at" && r.Sort != "updated_at" && r.Sort != "status" {
		return r, validation("sort", "Kolom tidak valid.")
	}
	if r.Direction == "" {
		r.Direction = "desc"
	}
	if r.Direction != "asc" && r.Direction != "desc" {
		return r, validation("direction", "Gunakan asc atau desc.")
	}
	for _, f := range []struct {
		key    string
		target **time.Time
	}{{"date_from", &r.From}, {"date_to", &r.To}} {
		if raw := q.Get(f.key); raw != "" {
			date, e := time.ParseInLocation("2006-01-02", raw, zone)
			if e != nil {
				return r, validation(f.key, "Gunakan tanggal YYYY-MM-DD yang valid.")
			}
			*f.target = &date
		}
	}
	if r.From != nil && r.To != nil && r.From.After(*r.To) {
		return r, validation("date_to", "Tidak boleh sebelum date_from.")
	}
	if r.To != nil {
		end := r.To.AddDate(0, 0, 1)
		r.To = &end
	}
	return r, nil
}

type AdminItem struct {
	ID           string    `json:"id"`
	Code         string    `json:"code"`
	ReporterName string    `json:"reporter_name"`
	Contact      string    `json:"contact"`
	ReportTypeID string    `json:"report_type_id"`
	ReportType   string    `json:"report_type"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
type AdminList struct {
	Items      []AdminItem `json:"items"`
	Pagination Pagination  `json:"pagination"`
}
type HistoryAdmin struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type AdminHistory struct {
	ID           string       `json:"id"`
	OldStatus    string       `json:"old_status"`
	NewStatus    string       `json:"new_status"`
	InternalNote string       `json:"internal_note"`
	ExternalNote string       `json:"external_note"`
	CreatedAt    time.Time    `json:"created_at"`
	Admin        HistoryAdmin `json:"admin"`
}
type AttachmentMeta struct {
	Available   bool   `json:"available"`
	Name        string `json:"name,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	Size        int64  `json:"size,omitempty"`
}
type AdminDetail struct {
	AdminItem
	ContactType       string         `json:"contact_type"`
	Detail            string         `json:"detail"`
	Attachment        AttachmentMeta `json:"attachment"`
	Histories         []AdminHistory `json:"histories"`
	HistoryPagination Pagination     `json:"history_pagination"`
}
type Mutation struct {
	Status            string     `json:"status"`
	InternalNote      string     `json:"internal_note"`
	ExternalNote      string     `json:"external_note"`
	ExpectedUpdatedAt *time.Time `json:"expected_updated_at"`
}
type NotesRequest struct {
	InternalNote      string     `json:"internal_note"`
	ExternalNote      string     `json:"external_note"`
	ExpectedUpdatedAt *time.Time `json:"expected_updated_at"`
}
type MutationResult struct {
	ID        string    `json:"id"`
	Status    string    `json:"status,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

func ValidateMutation(input Mutation, notes bool) (Mutation, error) {
	input.InternalNote = strings.TrimSpace(input.InternalNote)
	input.ExternalNote = strings.TrimSpace(input.ExternalNote)
	if input.ExpectedUpdatedAt == nil || input.ExpectedUpdatedAt.IsZero() {
		return input, validation("expected_updated_at", "Versi laporan wajib diisi.")
	}
	for _, v := range []string{input.InternalNote, input.ExternalNote} {
		if !utf8.ValidString(v) || strings.ContainsRune(v, 0) || utf8.RuneCountInString(v) > MaxNoteLength {
			return input, validation("note", "Maksimum 10.000 karakter per catatan.")
		}
	}
	if notes {
		if input.InternalNote == "" && input.ExternalNote == "" {
			return input, validation("note", "Isi minimal satu catatan.")
		}
	} else {
		if !IsStatus(input.Status) {
			return input, validation("status", "Status tidak valid.")
		}
		if input.Status == "Rejected" && input.ExternalNote == "" {
			return input, validation("external_note", "Alasan penolakan yang terlihat pelapor wajib diisi.")
		}
	}
	return input, nil
}

type StatusCount struct {
	Status string `json:"status"`
	Total  int    `json:"total"`
}
type TypeCount struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Total int    `json:"total"`
}
type Dashboard struct {
	Total      int           `json:"total_reports"`
	Open       int           `json:"open_reports"`
	OnProgress int           `json:"on_progress_reports"`
	Done       int           `json:"done_reports"`
	Rejected   int           `json:"rejected_reports"`
	Recent     []AdminItem   `json:"recent_reports"`
	ByStatus   []StatusCount `json:"by_status"`
	ByType     []TypeCount   `json:"by_report_type"`
}
type TypeOption struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"is_active"`
}
type TypeOptions struct {
	Items      []TypeOption `json:"items"`
	Pagination Pagination   `json:"pagination"`
}
