package handlers

import (
	"crypto/rand"
	"strconv"
	"strings"
)

// Username generation (user direction 2026-11-14): the system chooses
// each user's username from their name at registration time. Users do
// NOT pick a custom username. The generated username is lowercase
// [a-z0-9_], 3-32 characters, and unique: when the name is already
// taken, a number is appended (ahmad_khan, ahmad_khan2, ...).
// Persian/Arabic names are transliterated to Latin letters, so a user
// named «لیورا احمدی» becomes lyvra_ahmady.

// translit maps non-ASCII letters to their Latin reading. Letters not
// in the map are dropped by sanitizeUsernameBase.
var translit = map[rune]string{
	// Persian / Arabic letters.
	'ا': "a", 'آ': "a", 'ب': "b", 'پ': "p", 'ت': "t", 'ث': "s",
	'ج': "j", 'چ': "ch", 'ح': "h", 'خ': "kh", 'د': "d", 'ذ': "z",
	'ر': "r", 'ز': "z", 'ژ': "zh", 'س': "s", 'ش': "sh", 'ص': "s",
	'ض': "z", 'ط': "t", 'ظ': "z", 'ع': "a", 'غ': "gh", 'ف': "f",
	'ق': "q", 'ک': "k", 'ك': "k", 'گ': "g", 'ل': "l", 'م': "m",
	'ن': "n", 'و': "v", 'ه': "h", 'ی': "y", 'ي': "y", 'ى': "y",
	'ؤ': "v", 'ئ': "y", 'ة': "a", 'ء': "", 'ـ': "",
	// Common Latin accents.
	'é': "e", 'è': "e", 'ê': "e", 'ë': "e", 'á': "a", 'à': "a",
	'â': "a", 'ä': "a", 'í': "i", 'ì': "i", 'î': "i", 'ó': "o",
	'ò': "o", 'ô': "o", 'ö': "o", 'ú': "u", 'ù': "u", 'û': "u",
	'ü': "u", 'ç': "c", 'ñ': "n", 'ş': "s", 'ğ': "g", 'ı': "i",
}

// sanitizeUsernameBase turns a display name (or an email local part)
// into a lowercase [a-z0-9_] base: words joined with a single
// underscore, leading/trailing underscores removed, capped at 32
// characters. It returns "" when nothing usable remains.
func sanitizeUsernameBase(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	lastSep := true // avoids a leading underscore
	for _, r := range s {
		if rep, ok := translit[r]; ok {
			b.WriteString(rep)
			if rep != "" {
				lastSep = false
			}
			continue
		}
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			lastSep = false
		case r == ' ' || r == '-' || r == '.' || r == '_':
			if !lastSep {
				b.WriteByte('_')
				lastSep = true
			}
		default:
			// Diacritics, symbols and anything else: dropped.
		}
	}
	out := strings.Trim(b.String(), "_")
	if len(out) > 32 {
		out = strings.Trim(out[:32], "_")
	}
	return out
}

func randomDigits(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand essentially never fails; fall back to zeros
		// rather than failing registration over a username suffix.
		for i := range b {
			b[i] = '0'
		}
		return string(b)
	}
	for i := range b {
		b[i] = '0' + b[i]%10
	}
	return string(b)
}

// generateUsername picks a unique username from the user's display
// name. taken reports whether a candidate is already in use. When the
// name yields nothing usable (too short, or only symbols), the email
// local part is tried next, and finally "user" plus random digits.
func generateUsername(displayName, email string, taken func(string) bool) string {
	base := sanitizeUsernameBase(displayName)
	if len(base) < 3 {
		local := email
		if i := strings.IndexByte(email, '@'); i >= 0 {
			local = email[:i]
		}
		base = sanitizeUsernameBase(local)
	}
	if len(base) < 3 {
		base = "user" + randomDigits(4)
	}
	if len(base) > 32 {
		base = strings.Trim(base[:32], "_")
	}
	if !taken(base) {
		return base
	}
	for i := 2; i < 100; i++ {
		suffix := strconv.Itoa(i)
		cand := base
		if len(cand)+len(suffix) > 32 {
			cand = strings.Trim(cand[:32-len(suffix)], "_")
		}
		cand += suffix
		if !taken(cand) {
			return cand
		}
	}
	// Extremely unlikely (100 collisions): random suffix.
	for i := 0; i < 100; i++ {
		suffix := randomDigits(4)
		cand := base
		if len(cand)+len(suffix) > 32 {
			cand = strings.Trim(cand[:32-len(suffix)], "_")
		}
		cand += suffix
		if !taken(cand) {
			return cand
		}
	}
	return base + randomDigits(4)
}
