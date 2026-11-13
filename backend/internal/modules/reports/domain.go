package reports

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/big"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
	"wbs-golang/backend/internal/platform/problem"
	"wbs-golang/backend/internal/platform/security"
)

const ActiveLimit = 5

var phonePattern = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)
var codePattern = regexp.MustCompile(`^WBS-[0-9]{6}-[A-Z0-9]{2}[0-9]{3,19}$`)

func NormalizeContact(kind, raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if len(value) > 254 || !utf8.ValidString(value) {
		return "", validation("contact", "Contact tidak valid.")
	}
	switch kind {
	case "EMAIL":
		a, err := mail.ParseAddress(value)
		if err != nil || a.Address != value || !strings.Contains(value, "@") || strings.ContainsAny(value, "\r\n\x00") {
			return "", validation("contact", "Email tidak valid.")
		}
		return strings.ToLower(value), nil
	case "PHONE":
		value = strings.Map(func(r rune) rune {
			if r == ' ' || r == '-' || r == '(' || r == ')' || r == '.' {
				return -1
			}
			return r
		}, value)
		if !phonePattern.MatchString(value) {
			return "", validation("contact", "Gunakan nomor internasional + diikuti 8–15 digit.")
		}
		return value, nil
	default:
		return "", validation("contact_type", "Pilih EMAIL atau PHONE.")
	}
}
func ContactForTracking(raw string) (kind, contact string, err error) {
	kind = "PHONE"
	if strings.Contains(raw, "@") {
		kind = "EMAIL"
	}
	contact, err = NormalizeContact(kind, raw)
	return
}

// A 64-bit hash collision only serializes unrelated contacts; the count still uses exact fields.
func QuotaLockKey(kind, contact string) int64 {
	sum := sha256.Sum256([]byte("wbs/report-quota/v1\x00" + kind + "\x00" + contact))
	return int64(binary.BigEndian.Uint64(sum[:8]))
}
func RandomPrefix() (string, error) {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	out := make([]byte, 2)
	for i := range out {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", err
		}
		out[i] = alphabet[n.Int64()]
	}
	return string(out), nil
}
func FormatCode(created time.Time, zone *time.Location, prefix string, seq int64) (string, error) {
	if seq < 1 || len(prefix) != 2 || !regexp.MustCompile(`^[A-Z0-9]{2}$`).MatchString(prefix) {
		return "", fmt.Errorf("invalid report code input")
	}
	return fmt.Sprintf("WBS-%s-%s%03d", created.In(zone).Format("020106"), prefix, seq), nil
}

type Submission struct {
	Name         string `json:"name"`
	Contact      string `json:"contact"`
	ContactType  string `json:"contact_type"`
	ReportTypeID string `json:"report_type_id"`
	Detail       string `json:"detail"`
}
type Created struct {
	ReportCode           string    `json:"report_code"`
	Status               string    `json:"status"`
	CreatedAt            time.Time `json:"created_at"`
	TrackingInstructions string    `json:"tracking_instructions"`
}
type TrackRequest struct {
	ReportCode string `json:"report_code"`
	Contact    string `json:"contact"`
}
type Event struct {
	NewStatus    string    `json:"new_status"`
	ExternalNote string    `json:"external_note"`
	CreatedAt    time.Time `json:"created_at"`
}
type Tracked struct {
	ReportCode string    `json:"report_code"`
	Name       string    `json:"name"`
	ReportType string    `json:"report_type"`
	Detail     string    `json:"detail"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Timeline   []Event   `json:"timeline"`
}

func validation(field, message string) error {
	return problem.New(422, "VALIDATION_ERROR", field+": "+message)
}
func Validate(input Submission) (Submission, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Detail = strings.TrimSpace(input.Detail)
	for _, v := range []struct {
		field, value string
		max          int
	}{{"name", input.Name, 150}, {"detail", input.Detail, 20000}} {
		if !utf8.ValidString(v.value) || strings.ContainsRune(v.value, 0) || utf8.RuneCountInString(v.value) < 1 || utf8.RuneCountInString(v.value) > v.max {
			return input, validation(v.field, "Wajib diisi dan panjang tidak valid.")
		}
	}
	if !security.UUID.MatchString(input.ReportTypeID) {
		return input, validation("report_type_id", "Jenis laporan tidak valid.")
	}
	contact, err := NormalizeContact(input.ContactType, input.Contact)
	input.Contact = contact
	return input, err
}
func NotFound() error {
	return problem.New(404, "REPORT_NOT_FOUND", "Laporan tidak ditemukan. Periksa kode dan contact.")
}
