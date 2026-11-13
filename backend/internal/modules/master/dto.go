package master

import (
	"encoding/json"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"unicode/utf8"
	"wbs-golang/backend/internal/platform/problem"
	"wbs-golang/backend/internal/platform/security"
)

type Field struct {
	Name, Kind string
	Required   bool
	Max        int
}
type Entity struct {
	Path, Table, Permission string
	SuperOnly               bool
	Fields                  []Field
	Search                  []string
	Updated                 bool
}

var entities = []Entity{
	{"users", "admin_users", "users.manage", true, []Field{{"role_id", "uuid", true, 0}, {"name", "string", true, 150}, {"email", "email", true, 254}, {"password", "password", true, 72}, {"is_active", "bool", false, 0}}, []string{"name", "email"}, true},
	{"roles", "roles", "roles.manage", true, []Field{{"name", "string", true, 100}, {"description", "string", false, 10000}, {"permission_ids", "ids", false, 0}}, []string{"name"}, true},
	{"permissions", "permissions", "permissions.manage", true, []Field{{"slug", "slug", true, 100}, {"module_name", "string", true, 100}, {"description", "string", false, 10000}}, []string{"slug", "module_name"}, false},
	{"report-types", "report_types", "report-types.manage", false, []Field{{"name", "string", true, 150}, {"description", "string", false, 10000}, {"is_active", "bool", false, 0}}, []string{"name"}, true},
	{"faqs", "faqs", "faqs.manage", false, []Field{{"report_type_id", "nullable_uuid", false, 0}, {"question", "string", true, 5000}, {"answer", "string", true, 20000}, {"sort_order", "int", false, 0}, {"is_active", "bool", false, 0}}, []string{"question", "answer"}, false},
	{"email-templates", "email_templates", "email-templates.manage", false, []Field{{"event_key", "event", true, 100}, {"subject", "string", true, 255}, {"body_html", "string", true, 100000}}, []string{"event_key", "subject"}, true},
}

func Entities() []Entity { return entities }
func Find(path string) (Entity, bool) {
	for _, e := range entities {
		if e.Path == path {
			return e, true
		}
	}
	return Entity{}, false
}

type Input map[string]json.RawMessage
type Values map[string]any
type Page struct {
	Items    []json.RawMessage `json:"items"`
	Total    int               `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
}
type Query struct {
	Page, Size                 int
	Search, Active, ReportType string
	Public                     bool
}

var slugPattern = regexp.MustCompile(`^[a-z][a-z-]*\.[a-z][a-z-]*$`)
var eventPattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

func invalid(message string) error { return problem.New(422, "VALIDATION_ERROR", message) }
func Validate(e Entity, input Input, create bool) (Values, error) {
	if len(input) == 0 {
		return nil, invalid("Isi perubahan wajib diisi.")
	}
	known := map[string]Field{}
	for _, f := range e.Fields {
		known[f.Name] = f
		if create && f.Required {
			if _, ok := input[f.Name]; !ok {
				return nil, invalid(f.Name + " wajib diisi.")
			}
		}
	}
	out := Values{}
	for name, raw := range input {
		f, ok := known[name]
		if !ok {
			return nil, invalid("Field tidak dikenal: " + name)
		}
		var value any
		if string(raw) == "null" {
			if f.Kind == "nullable_uuid" {
				out[name] = nil
				continue
			}
			return nil, invalid(name + " tidak boleh null.")
		}
		switch f.Kind {
		case "bool":
			var v bool
			if json.Unmarshal(raw, &v) != nil {
				return nil, invalid(name + " harus boolean.")
			}
			value = v
		case "int":
			var v int32
			if json.Unmarshal(raw, &v) != nil || v < 0 {
				return nil, invalid(name + " harus integer nonnegatif.")
			}
			value = v
		case "ids":
			var ids []string
			if json.Unmarshal(raw, &ids) != nil || len(ids) > 200 {
				return nil, invalid("permission_ids tidak valid.")
			}
			seen := map[string]bool{}
			for _, id := range ids {
				if !security.UUID.MatchString(id) || seen[id] {
					return nil, invalid("permission_ids harus UUID unik.")
				}
				seen[id] = true
			}
			value = ids
		default:
			var v string
			if json.Unmarshal(raw, &v) != nil || !utf8.ValidString(v) {
				return nil, invalid(name + " harus string.")
			}
			if f.Kind != "password" {
				v = strings.TrimSpace(v)
			}
			if (f.Required && v == "") || (f.Max > 0 && utf8.RuneCountInString(v) > f.Max) || strings.ContainsRune(v, 0) {
				return nil, invalid(name + " kosong atau terlalu panjang.")
			}
			switch f.Kind {
			case "uuid", "nullable_uuid":
				if !security.UUID.MatchString(v) {
					return nil, invalid(name + " harus UUID.")
				}
			case "email":
				address, err := mail.ParseAddress(v)
				if err != nil || address.Address != v {
					return nil, invalid("Email tidak valid.")
				}
				v = strings.ToLower(v)
			case "password":
				hash, err := security.HashPassword(v)
				if err != nil {
					return nil, invalid("Password harus 12–72 byte.")
				}
				out["password_hash"] = hash
				continue
			case "slug":
				if !slugPattern.MatchString(v) {
					return nil, invalid("Slug harus module.action.")
				}
			case "event":
				if !eventPattern.MatchString(v) {
					return nil, invalid("Event key tidak valid.")
				}
			}
			value = v
		}
		out[name] = value
	}
	return out, nil
}
func ValidateTemplate(values Values, existing map[string]any) error {
	subject, _ := values["subject"].(string)
	body, _ := values["body_html"].(string)
	if _, ok := values["subject"]; !ok {
		subject, _ = existing["subject"].(string)
	}
	if _, ok := values["body_html"]; !ok {
		body, _ = existing["body_html"].(string)
	}
	cleaned, err := security.Template(subject, body)
	if err != nil {
		return invalid(fmt.Sprint(err))
	}
	values["body_html"] = cleaned
	return nil
}
