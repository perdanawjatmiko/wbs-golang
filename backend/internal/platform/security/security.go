package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/microcosm-cc/bluemonday"
	"golang.org/x/crypto/bcrypt"
	"regexp"
	"strings"
)

const PasswordCost = 12

func HashPassword(password string) (string, error) {
	if len(password) < 12 || len(password) > 72 {
		return "", fmt.Errorf("password must contain 12..72 bytes")
	}
	b, err := bcrypt.GenerateFromPassword([]byte(password), PasswordCost)
	return string(b), err
}
func Verify(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
func Token() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("secure randomness unavailable")
	}
	return hex.EncodeToString(b)
}
func HashToken(token string) []byte { s := sha256.Sum256([]byte(token)); return s[:] }
func CSRF(token string) string {
	s := sha256.Sum256([]byte("wbs-csrf-v1:" + token))
	return hex.EncodeToString(s[:])
}

var UUID = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
var placeholder = regexp.MustCompile(`\{\{\s*([a-z_]+)\s*\}\}`)
var allowed = map[string]bool{"report_code": true, "user_name": true, "status": true, "external_note": true}

func Template(subject, body string) (string, error) {
	for _, s := range []string{subject, body} {
		for _, match := range placeholder.FindAllStringSubmatch(s, -1) {
			if !allowed[match[1]] {
				return "", fmt.Errorf("unsupported template placeholder")
			}
		}
		rest := placeholder.ReplaceAllString(s, "")
		if strings.Contains(rest, "{{") || strings.Contains(rest, "}}") {
			return "", fmt.Errorf("invalid template placeholder")
		}
	}
	// Deliberately small formatting-only allowlist: no URL, style, image, script or event attributes.
	policy := bluemonday.NewPolicy()
	policy.AllowElements("p", "br", "strong", "em", "ul", "ol", "li", "h1", "h2", "h3", "blockquote")
	cleaned := policy.Sanitize(body)
	if strings.TrimSpace(cleaned) == "" {
		return "", fmt.Errorf("HTML body is empty after sanitization")
	}
	return cleaned, nil
}
