package agent

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	reScript   = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	reStyle    = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	reNoscript = regexp.MustCompile(`(?is)<noscript[^>]*>.*?</noscript>`)
	reTag      = regexp.MustCompile(`(?is)<[^>]+>`)
	reSpace    = regexp.MustCompile(`[ \t\x0b\f\r]+`)
)

func ReadableHTML(raw string) string {
	s := reScript.ReplaceAllString(raw, " ")
	s = reStyle.ReplaceAllString(s, " ")
	s = reNoscript.ReplaceAllString(s, " ")
	s = reTag.ReplaceAllString(s, " ")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", `"`)
	s = reSpace.ReplaceAllString(s, " ")
	lines := strings.Split(s, "\n")
	var keep []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		keep = append(keep, line)
	}
	out := strings.Join(keep, "\n")
	return strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' {
			return -1
		}
		return r
	}, out))
}

func looksLikeHTML(contentType, body string) bool {
	ct := strings.ToLower(contentType)
	if strings.Contains(ct, "html") {
		return true
	}
	trim := strings.TrimSpace(strings.ToLower(body))
	return strings.HasPrefix(trim, "<!doctype html") || strings.HasPrefix(trim, "<html")
}
