package notifications

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"wbs-golang/backend/internal/platform/security"
)

var slots = regexp.MustCompile(`\{\{\s*([a-z_]+)\s*\}\}`)

func Render(eventKey, subject, body, userName string, p Payload) (string, string, error) {
	clean, err := security.Template(subject, body)
	if err != nil {
		return "", "", fmt.Errorf("TEMPLATE_INVALID")
	}
	combined := subject + clean
	found := map[string]bool{}
	for _, m := range slots.FindAllStringSubmatch(combined, -1) {
		found[m[1]] = true
	}
	if !found["report_code"] || (eventKey == "REPORT_REJECTED" && !found["external_note"]) {
		return "", "", fmt.Errorf("TEMPLATE_REQUIRED_PLACEHOLDER")
	}
	values := map[string]string{"report_code": p.ReportCode, "user_name": userName, "status": p.Status, "external_note": p.ExternalNote}
	replace := func(s string, escape bool) string {
		return slots.ReplaceAllStringFunc(s, func(slot string) string {
			v := values[slots.FindStringSubmatch(slot)[1]]
			if escape {
				return html.EscapeString(v)
			}
			return v
		})
	}
	subject = replace(subject, false)
	if strings.ContainsAny(subject, "\r\n\x00") || len(subject) > 998 {
		return "", "", fmt.Errorf("TEMPLATE_SUBJECT_INVALID")
	}
	return subject, replace(clean, true), nil
}
func TelegramText(p Payload) string {
	return fmt.Sprintf("Laporan baru\nKode: %s\nJenis: %s\nStatus: Open\nDibuat: %s", p.ReportCode, p.ReportType, p.Timestamp.UTC().Format("2006-01-02 15:04:05 UTC"))
}
