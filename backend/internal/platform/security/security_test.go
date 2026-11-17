package security

import (
	"strings"
	"testing"
)

func TestPassword(t *testing.T) {
	hash, err := HashPassword("correct-password-123")
	if err != nil {
		t.Fatal(err)
	}
	if hash == "correct-password-123" || !Verify(hash, "correct-password-123") || Verify(hash, "incorrect") {
		t.Fatal("hash verification failed")
	}
	for _, password := range []string{"short", strings.Repeat("a", 73)} {
		if _, err := HashPassword(password); err == nil {
			t.Fatal("invalid password accepted")
		}
	}
}
func TestToken(t *testing.T) {
	a, b := Token(), Token()
	if len(a) != 64 || a == b || string(HashToken(a)) == a || CSRF(a) == a || CSRF(a) == CSRF(b) {
		t.Fatal("token generation failed")
	}
}
func TestTemplate(t *testing.T) {
	cleaned, err := Template("{{report_code}}", `<p onclick="alert(1)">{{external_note}}</p><script>alert(1)</script><img src=x onerror=alert(1)><a href="javascript:alert(1)">text</a>`)
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"script", "onclick", "onerror", "href", "<img"} {
		if strings.Contains(cleaned, bad) {
			t.Fatal("unsafe HTML", cleaned)
		}
	}
	for _, bad := range []string{"{{password}}", "{{malformed}", "{{report_code | unsafe}}"} {
		if _, err := Template("Subject", bad); err == nil {
			t.Fatal("unsafe placeholder accepted")
		}
	}
	formatted, err := Template("Subject", `<p><u>Important</u> <a href="https://example.test/help">Help</a> {{report_code}}</p>`)
	if err != nil || !strings.Contains(formatted, "<u>") || !strings.Contains(formatted, `href="https://example.test/help"`) {
		t.Fatal("safe rich formatting was not preserved", formatted, err)
	}
	unsafeLink, err := Template("Subject", `<p><a href="data:text/html,alert(1)">unsafe</a></p>`)
	if err != nil || strings.Contains(unsafeLink, "href=") {
		t.Fatal("unsafe link was preserved", unsafeLink, err)
	}
}
