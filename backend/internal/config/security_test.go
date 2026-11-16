package config

import "testing"

func TestSecurityConfig(t *testing.T) {
	for _, v := range []map[string]string{{"COOKIE_SECURE": "invalid"}, {"LOGIN_RATE_LIMIT": "0"}, {"RATE_LIMIT_BUCKETS": "1000000"}, {"RATE_LIMIT_TIMEOUT": "20s"}, {"HTTP_TRUSTED_PROXY_CIDRS": "0.0.0.0/0"}, {"HTTP_TRUSTED_PROXY_CIDRS": "::/0"}, {"HTTP_TRUSTED_PROXY_CIDRS": "192.0.2.0/24"}, {"HTTP_TRUSTED_PROXY_CIDRS": "10.0.0.0/7"}} {
		v["DATABASE_URL"] = "postgres://qa@localhost/test"
		if _, e := Parse(func(k string) string { return v[k] }); e == nil {
			t.Fatalf("invalid policy accepted: %v", v)
		}
	}
	v := map[string]string{"DATABASE_URL": "postgres://qa@localhost/test", "COOKIE_SECURE": "true", "HTTP_TRUSTED_PROXY_CIDRS": "10.88.0.0/24"}
	c, e := Parse(func(k string) string { return v[k] })
	if e != nil || !c.Security.CookieSecure || len(c.Security.TrustedProxyCIDRs) != 1 {
		t.Fatal(e)
	}
}
