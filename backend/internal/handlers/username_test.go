package handlers

import (
	"strings"
	"testing"
)

func TestSanitizeUsernameBase(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Najmuddin Safari", "najmuddin_safari"},
		{"Ahmad-Khan!", "ahmad_khan"},
		{"  Mary  Jane  ", "mary_jane"},
		{"لیورا", "lyvra"},
		{"لیورا احمدی", "lyvra_ahmdy"},
		{"José García", "jose_garcia"},
		{"a.b_c d", "a_b_c_d"},
		{"!!!", ""},
		{"AB", "ab"},
		{"a very long display name that goes on and on forever", "a_very_long_display_name_that_go"},
	}
	for _, c := range cases {
		if got := sanitizeUsernameBase(c.in); got != c.want {
			t.Errorf("sanitizeUsernameBase(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestGenerateUsername(t *testing.T) {
	// Free name: used as-is.
	if got := generateUsername("Najmuddin Safari", "n@example.com", func(string) bool { return false }); got != "najmuddin_safari" {
		t.Errorf("free name: got %q", got)
	}
	// Taken name: numeric suffix.
	taken := map[string]bool{"ahmad_khan": true, "ahmad_khan2": true}
	if got := generateUsername("Ahmad Khan", "a@example.com", func(s string) bool { return taken[s] }); got != "ahmad_khan3" {
		t.Errorf("taken name: got %q, want ahmad_khan3", got)
	}
	// Persian name transliterates.
	if got := generateUsername("لیورا احمدی", "l@example.com", func(string) bool { return false }); got != "lyvra_ahmdy" {
		t.Errorf("persian name: got %q", got)
	}
	// Name too short: fall back to the email local part.
	if got := generateUsername("Al", "alex.khan@example.com", func(string) bool { return false }); got != "alex_khan" {
		t.Errorf("short name: got %q, want alex_khan", got)
	}
	// Nothing usable anywhere: user + 4 digits.
	got := generateUsername("!!", "x@example.com", func(string) bool { return false })
	if !strings.HasPrefix(got, "user") || len(got) != 8 {
		t.Errorf("fallback: got %q, want user + 4 digits", got)
	}
	for _, r := range got[4:] {
		if r < '0' || r > '9' {
			t.Errorf("fallback: got %q, suffix is not digits", got)
		}
	}
}
