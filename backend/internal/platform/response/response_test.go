package response

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"testing"
)

func TestEnvelope(t *testing.T) {
	for _, success := range []bool{true, false} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("request_id", "test-id")
		if success {
			Success(c, 200, map[string]string{"status": "ok"})
		} else {
			Error(c, 503, "NOT_READY", "Layanan belum siap.")
		}
		var body Envelope
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.Success != success || body.Meta.RequestID != "test-id" {
			t.Fatal(body)
		}
		if !success && (w.Code != 503 || body.Error.Code != "NOT_READY" || !c.IsAborted()) {
			t.Fatal(body)
		}
	}
}
