package reports

import (
	"strings"
	"testing"
	"time"
)

func TestContactNormalization(t *testing.T) {
	for _, tt := range []struct{ kind, in, want string }{{"EMAIL", " User.Name+Alias@Example.COM ", "user.name+alias@example.com"}, {"PHONE", " +44 (20) 7946-0958 ", "+442079460958"}, {"PHONE", "+62 812.3456.7890", "+6281234567890"}} {
		got, err := NormalizeContact(tt.kind, tt.in)
		if err != nil || got != tt.want {
			t.Fatal(tt, got, err)
		}
		kind, contact, err := ContactForTracking(tt.in)
		if err != nil || kind != tt.kind || contact != got {
			t.Fatal("tracking normalization mismatch")
		}
	}
	for _, tt := range []struct{ kind, in string }{{"EMAIL", "Person <user@example.test>"}, {"EMAIL", "bad"}, {"PHONE", "08123456789"}, {"PHONE", "+01234567890"}, {"PHONE", "+1<script>"}, {"PHONE", "+123"}, {"PHONE", strings.Repeat("1", 300)}, {"OTHER", "value"}} {
		if _, err := NormalizeContact(tt.kind, tt.in); err == nil {
			t.Fatal("invalid accepted", tt)
		}
	}
	if QuotaLockKey("EMAIL", "a@example.test") != QuotaLockKey("EMAIL", "a@example.test") || QuotaLockKey("EMAIL", "a@example.test") == QuotaLockKey("PHONE", "a@example.test") {
		t.Fatal("unstable/non-namespaced lock")
	}
}
func TestCode(t *testing.T) {
	zone, _ := time.LoadLocation("Asia/Jakarta")
	instant := time.Date(2026, 11, 13, 17, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		seq  int64
		want string
	}{{1, "WBS-141126-X9001"}, {999, "WBS-141126-X9999"}, {1000, "WBS-141126-X91000"}} {
		got, err := FormatCode(instant, zone, "X9", tt.seq)
		if err != nil || got != tt.want {
			t.Fatal(got, err)
		}
	}
	for i := 0; i < 100; i++ {
		prefix, err := RandomPrefix()
		if err != nil || len(prefix) != 2 {
			t.Fatal(prefix, err)
		}
		if _, err = FormatCode(instant, zone, prefix, 1); err != nil {
			t.Fatal(err)
		}
	}
}
func TestReportValidation(t *testing.T) {
	valid := Submission{Name: " Name ", Contact: " User@example.test ", ContactType: "EMAIL", ReportTypeID: "00000000-0000-0000-0000-000000000001", Detail: " Details "}
	out, err := Validate(valid)
	if err != nil || out.Name != "Name" || out.Detail != "Details" {
		t.Fatal(out, err)
	}
	for _, field := range []string{"name", "contact", "type", "report-type", "detail"} {
		copy := valid
		switch field {
		case "name":
			copy.Name = " "
		case "contact":
			copy.Contact = "bad"
		case "type":
			copy.ContactType = "invalid"
		case "report-type":
			copy.ReportTypeID = "bad"
		case "detail":
			copy.Detail = ""
		}
		if _, err := Validate(copy); err == nil {
			t.Fatal(field)
		}
	}
}
