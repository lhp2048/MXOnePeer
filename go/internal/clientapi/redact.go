package clientapi

import "strings"

func Redact(text string) string {
	const marker = "tskey-auth-"
	const redacted = "tskey-auth-***"
	var b strings.Builder
	for {
		start := strings.Index(text, marker)
		if start < 0 {
			b.WriteString(text)
			return b.String()
		}
		b.WriteString(text[:start])
		b.WriteString(redacted)
		end := start + len(marker)
		for end < len(text) && text[end] != ' ' && text[end] != '\n' && text[end] != '\r' && text[end] != '\t' {
			end++
		}
		text = text[end:]
	}
}
