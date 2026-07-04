package sqlcheck

import (
	"strings"
	"unicode"
)

func IsSelect(sql string) bool {
	normalized := strings.TrimSpace(stripLeadingComments(sql))
	normalized = strings.TrimLeftFunc(normalized, unicode.IsSpace)
	if normalized == "" {
		return false
	}
	upper := strings.ToUpper(normalized)
	return strings.HasPrefix(upper, "SELECT") || strings.HasPrefix(upper, "WITH")
}

func stripLeadingComments(sql string) string {
	s := strings.TrimSpace(sql)
	for {
		switch {
		case strings.HasPrefix(s, "--"):
			if idx := strings.IndexByte(s, '\n'); idx >= 0 {
				s = strings.TrimSpace(s[idx+1:])
				continue
			}
			return ""
		case strings.HasPrefix(s, "/*"):
			if idx := strings.Index(s, "*/"); idx >= 0 {
				s = strings.TrimSpace(s[idx+2:])
				continue
			}
			return s
		default:
			return s
		}
	}
}
