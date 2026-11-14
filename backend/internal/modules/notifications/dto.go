package notifications

import "time"

type InboxItem struct {
	ID        string     `json:"id"`
	EventID   string     `json:"event_id"`
	ReportID  string     `json:"report_id"`
	Title     string     `json:"title"`
	Message   string     `json:"message"`
	ReadAt    *time.Time `json:"read_at"`
	CreatedAt time.Time  `json:"created_at"`
}
type Page struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}
type Inbox struct {
	Items      []InboxItem `json:"items"`
	Pagination Page        `json:"pagination"`
}
type Delivery struct {
	ID              string     `json:"id"`
	EventKey        string     `json:"event_key"`
	Channel         string     `json:"channel"`
	Status          string     `json:"status"`
	Attempts        int        `json:"attempts"`
	MaxAttempts     int        `json:"max_attempts"`
	NextAttempt     time.Time  `json:"next_attempt_at"`
	LastError       *string    `json:"last_error"`
	CreatedAt       time.Time  `json:"created_at"`
	SentAt          *time.Time `json:"sent_at"`
	ManualRetryUsed bool       `json:"manual_retry_used"`
}
type Deliveries struct {
	Items      []Delivery     `json:"items"`
	Pagination Page           `json:"pagination"`
	Metrics    map[string]int `json:"metrics"`
}
type RealtimeEvent struct {
	Sequence  int64     `json:"-"`
	Name      string    `json:"-"`
	ID        string    `json:"event_id"`
	ReportID  string    `json:"report_id"`
	Code      string    `json:"report_code"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

func page(p, size, total int) Page { return Page{p, size, total, (total + size - 1) / size} }
