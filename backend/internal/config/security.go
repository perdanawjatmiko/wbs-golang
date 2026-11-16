package config

import (
	"fmt"
	"net/netip"
	"strconv"
	"strings"
	"time"
)

type SecurityConfig struct {
	CookieSecure             bool
	TrustedProxyCIDRs        []string
	LoginLimit, RateSlots    int
	LoginWindow, RateTimeout time.Duration
}

func parseSecurity(get func(string) string, env string) (SecurityConfig, error) {
	c := SecurityConfig{CookieSecure: env == "production", LoginLimit: 10, RateSlots: 16384, LoginWindow: 15 * time.Minute, RateTimeout: time.Second}
	if v := get("COOKIE_SECURE"); v != "" {
		value, err := strconv.ParseBool(v)
		if err != nil {
			return c, fmt.Errorf("COOKIE_SECURE must be boolean")
		}
		c.CookieSecure = value
	}
	if env == "production" && !c.CookieSecure {
		return c, fmt.Errorf("production requires COOKIE_SECURE=true")
	}
	for _, f := range []struct {
		k        string
		target   *int
		min, max int
	}{{"LOGIN_RATE_LIMIT", &c.LoginLimit, 1, 10000}, {"RATE_LIMIT_BUCKETS", &c.RateSlots, 1024, 65536}} {
		if v := get(f.k); v != "" {
			n, e := strconv.Atoi(v)
			if e != nil || n < f.min || n > f.max {
				return c, fmt.Errorf("%s outside allowed range", f.k)
			}
			*f.target = n
		}
	}
	for _, f := range []struct {
		k string
		p *time.Duration
	}{{"LOGIN_RATE_WINDOW", &c.LoginWindow}, {"RATE_LIMIT_TIMEOUT", &c.RateTimeout}} {
		if v := get(f.k); v != "" {
			n, e := time.ParseDuration(v)
			if e != nil || n < time.Second || n > 24*time.Hour {
				return c, fmt.Errorf("%s must be 1s..24h", f.k)
			}
			*f.p = n
		}
	}
	if v := strings.TrimSpace(get("HTTP_TRUSTED_PROXY_CIDRS")); v != "" {
		for _, raw := range strings.Split(v, ",") {
			p, e := netip.ParsePrefix(strings.TrimSpace(raw))
			if e != nil || !privateProxy(p) || p.Addr().Is4In6() {
				return c, fmt.Errorf("HTTP_TRUSTED_PROXY_CIDRS requires explicit non-global IPv4/IPv6 CIDRs")
			}
			if len(c.TrustedProxyCIDRs) >= 16 {
				return c, fmt.Errorf("HTTP_TRUSTED_PROXY_CIDRS supports at most 16 prefixes")
			}
			c.TrustedProxyCIDRs = append(c.TrustedProxyCIDRs, p.Masked().String())
		}
	}
	return c, nil
}

// Only controlled private/loopback proxy ranges may establish forwarded identity.
func privateProxy(p netip.Prefix) bool {
	if !p.IsValid() {
		return false
	}
	for _, raw := range []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "127.0.0.0/8", "fc00::/7", "::1/128"} {
		allowed := netip.MustParsePrefix(raw)
		if p.Bits() >= allowed.Bits() && allowed.Contains(p.Masked().Addr()) {
			return true
		}
	}
	return false
}
