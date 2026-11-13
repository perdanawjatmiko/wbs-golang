package master

import (
	"encoding/json"
	"testing"
)

func TestValidation(t *testing.T) {
	for _, tt := range []struct {
		resource, payload string
		create            bool
	}{{"permissions", `{"slug":"bad","module_name":"x"}`, true}, {"roles", `{"name":""}`, true}, {"users", `{"password_hash":"plaintext"}`, false}, {"faqs", `{"sort_order":-1}`, false}, {"faqs", `{"report_type_id":"not-uuid"}`, false}, {"report-types", `{"is_active":"true"}`, false}, {"roles", `{"permission_ids":["00000000-0000-0000-0000-000000000000","00000000-0000-0000-0000-000000000000"]}`, false}} {
		e, _ := Find(tt.resource)
		var input Input
		_ = json.Unmarshal([]byte(tt.payload), &input)
		if _, err := Validate(e, input, tt.create); err == nil {
			t.Fatal("invalid accepted", tt.resource, tt.payload)
		}
	}
	e, _ := Find("faqs")
	values, err := Validate(e, Input{"report_type_id": json.RawMessage(`null`)}, false)
	if err != nil || values["report_type_id"] != nil {
		t.Fatal("nullable FK rejected")
	}
}
