package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"
	"wbs-golang/backend/internal/config"
)

func TestMappingRenderingAndPrivacy(t *testing.T) {
	for status, key := range map[string]string{"Open": "", "On Progress": "REPORT_ON_PROGRESS", "Done": "REPORT_DONE", "Rejected": "REPORT_REJECTED"} {
		if EventKey(status) != key {
			t.Fatal(status)
		}
	}
	p := Payload{Version: 1, ReportCode: "WBS-141126-X9001", ReportType: "Ethics", Status: "Rejected", ExternalNote: `<script>alert(1)</script> & reason`, Timestamp: time.Now()}
	subject, html, err := Render("REPORT_REJECTED", "{{report_code}}", `<p onclick="evil()">{{user_name}}: {{external_note}}</p><script>evil()</script>`, "<b>A</b>", p)
	if err != nil || subject != p.ReportCode || strings.Contains(html, "<script>") || strings.Contains(html, "onclick") || !strings.Contains(html, "&lt;script&gt;") || !strings.Contains(html, "&lt;b&gt;") {
		t.Fatalf("unsafe template: %q %v", html, err)
	}
	for _, body := range []string{`<p>{{unknown}}</p>`, `<p>{{report_code}}</p>`, `<p>{{report_code}}{{external_note}} {{if .Secret}}</p>`} {
		if _, _, err := Render("REPORT_REJECTED", "subject", body, "user", p); err == nil {
			t.Fatal("invalid template accepted")
		}
	}
	if _, _, err := Render("REPORT_CREATED", "{{user_name}} {{report_code}}", `<p>{{report_code}}</p>`, "x\r\nBcc: attack", p); err == nil {
		t.Fatal("header injection")
	}
	text := TelegramText(p)
	if strings.Contains(text, "reason") || strings.Contains(text, "script") || !strings.Contains(text, p.ReportCode) {
		t.Fatal("telegram privacy")
	}
	e := RealtimeEvent{Sequence: 12, ID: "id", Name: "report.created", Code: p.ReportCode, Status: "Open", Timestamp: p.Timestamp}
	frame := SSEFrame(e)
	if !strings.Contains(frame, "id: 12\nevent: report.created\ndata: ") {
		t.Fatal(frame)
	}
	data, _ := json.Marshal(p)
	for _, secret := range []string{"contact", "internal_note", "attachment", "detail"} {
		if strings.Contains(string(data), secret) {
			t.Fatal("payload leak")
		}
	}
}
func TestRetryAndValidation(t *testing.T) {
	intervals := []time.Duration{30 * time.Second, 2 * time.Minute, 10 * time.Minute, 30 * time.Minute, 2 * time.Hour}
	for attempt := 1; attempt <= 10; attempt++ {
		base := intervals[min(attempt-1, 4)]
		for _, u := range []float64{-1, 0, .5, 1, 2} {
			v := Backoff(attempt, intervals, u)
			if v < base || v > base*12/10 {
				t.Fatal(v)
			}
		}
	}
	if !Classify(context.DeadlineExceeded).Retryable || !Classify(errors.New("secret URL")).Retryable {
		t.Fatal("network classification")
	}
	f := &Failure{Code: "INVALID_RECIPIENT"}
	if Classify(f).Retryable || Classify(f).Error() != "INVALID_RECIPIENT" {
		t.Fatal("permanent classification")
	}
	for _, v := range []string{"-1", "secret", "1 OR 1=1", "9223372036854775808"} {
		if _, err := parseCursor(v); err == nil {
			t.Fatal(v)
		}
	}
	if p, s, e := ParsePage("", "100"); e != nil || p != 1 || s != 100 {
		t.Fatal("pagination")
	}
	if _, _, e := ParsePage("1", "101"); e == nil {
		t.Fatal("bound")
	}
	c, _ := config.Parse(func(k string) string {
		if k == "DATABASE_URL" {
			return "postgres://test@localhost/test"
		}
		return ""
	})
	c.Notifications.WorkerEnabled = false
	w := NewWorker(nil, nil, nil, c.Notifications, time.Second, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err := w.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c.Notifications.WorkerEnabled = true
	w.config = c.Notifications
	if err := w.Run(ctx); err != nil {
		t.Fatal(err)
	}
}
func TestPayloadValidation(t *testing.T) {
	valid := Payload{Version: 1, ReportCode: "WBS-141126-X9001", ReportType: "Type", Status: "Open", Timestamp: time.Now()}
	if !ValidPayload("REPORT_CREATED", valid) {
		t.Fatal("valid event")
	}
	for _, mutate := range []func(*Payload){func(p *Payload) { p.Version = 2 }, func(p *Payload) { p.ReportCode = "" }, func(p *Payload) { p.Timestamp = time.Time{} }, func(p *Payload) { p.Status = "Done" }} {
		p := valid
		mutate(&p)
		if ValidPayload("REPORT_CREATED", p) {
			t.Fatal("invalid payload accepted")
		}
	}
	valid.Status = "Rejected"
	if ValidPayload("REPORT_REJECTED", valid) {
		t.Fatal("missing rejection reason")
	}
	valid.ExternalNote = "reason"
	if !ValidPayload("REPORT_REJECTED", valid) {
		t.Fatal("rejected snapshot")
	}
}
