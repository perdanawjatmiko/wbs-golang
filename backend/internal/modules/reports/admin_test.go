package reports

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestAdminQueryValidation(t *testing.T) {
	zone, _ := time.LoadLocation("Asia/Jakarta")
	q, err := ParseAdminQuery(url.Values{}, zone)
	if err != nil || q.Page != 1 || q.PerPage != 20 || q.Sort != "created_at" || q.Direction != "desc" {
		t.Fatal(q, err)
	}
	for _, query := range []string{"per_page=101", "page=0", "status=Approved", "report_type_id=bad", "date_from=2026-02-30", "date_from=2026-11-14&date_to=2026-11-13", "sort=password_hash", "direction=DESC%3BDROP", "search=" + strings.Repeat("x", 201)} {
		v, _ := url.ParseQuery(query)
		if _, err := ParseAdminQuery(v, zone); err == nil {
			t.Fatal("invalid accepted", query)
		}
	}
	v, _ := url.ParseQuery("date_from=2026-11-13&date_to=2026-11-13")
	q, err = ParseAdminQuery(v, zone)
	if err != nil || q.From.UTC().Hour() != 17 || q.To.Sub(*q.From) != 24*time.Hour {
		t.Fatal("date range", q, err)
	}
}
func TestWorkflowValidation(t *testing.T) {
	now := time.Now()
	for _, status := range []string{"Open", "On Progress", "Done", "Rejected"} {
		input, err := ValidateMutation(Mutation{Status: status, ExternalNote: " Reason ", ExpectedUpdatedAt: &now}, false)
		if err != nil || input.ExternalNote != "Reason" {
			t.Fatal(status, err)
		}
	}
	for _, input := range []Mutation{{Status: "Approved", ExpectedUpdatedAt: &now}, {Status: "Rejected", ExternalNote: " ", ExpectedUpdatedAt: &now}, {Status: "Open"}, {Status: "Done", InternalNote: strings.Repeat("x", 10001), ExpectedUpdatedAt: &now}} {
		if _, err := ValidateMutation(input, false); err == nil {
			t.Fatal("invalid workflow accepted")
		}
	}
	if _, err := ValidateMutation(Mutation{ExpectedUpdatedAt: &now}, true); err == nil {
		t.Fatal("empty notes accepted")
	}
	if _, err := ValidateMutation(Mutation{InternalNote: " Only internal ", ExpectedUpdatedAt: &now}, true); err != nil {
		t.Fatal(err)
	}
	if IsActive("Done") || IsActive("Rejected") || !IsActive("Open") || !IsActive("On Progress") {
		t.Fatal("active state mismatch")
	}
}
