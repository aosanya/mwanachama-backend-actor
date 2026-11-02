package main

import (
	"net/url"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	redacted       = "[redacted]"
	shortSecretLen = 8
)

var (
	uriUserinfo   = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*://([^/?#]*)@`)
	uriHost       = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*://(?:[^/?#@]*@)?([^/?#]*)`)
	keywordSecret = regexp.MustCompile(`(?i)\b(?:user|password|host|hostaddr)\s*=\s*('(?:[^'\\]|\\.)*'|[^\s&]+)`)
)

type span struct{ start, end int }

func redact(message, location string) string {
	var spans []span
	for _, secret := range secretsIn(location) {
		spans = append(spans, occurrences(message, secret)...)
	}
	sort.Slice(spans, func(i, j int) bool { return spans[i].start < spans[j].start })
	var out strings.Builder
	at := 0
	for _, s := range spans {
		if s.end <= at {
			continue
		}
		if s.start > at {
			out.WriteString(message[at:s.start])
		}
		out.WriteString(redacted)
		at = s.end
	}
	out.WriteString(message[at:])
	return out.String()
}

func occurrences(message, secret string) []span {
	var found []span
	for from := 0; from <= len(message)-len(secret); {
		i := strings.Index(message[from:], secret)
		if i < 0 {
			break
		}
		start := from + i
		end := start + len(secret)
		if len(secret) >= shortSecretLen || standsAlone(message, start, end) {
			found = append(found, span{start, end})
		}
		from = start + 1
	}
	return found
}

func standsAlone(message string, start, end int) bool {
	return (start == 0 || !isWordByte(message[start-1])) && (end == len(message) || !isWordByte(message[end]))
}

func isWordByte(b byte) bool {
	return b >= '0' && b <= '9' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b == '_'
}

func secretsIn(location string) []string {
	secrets := []string{location}
	if abs, err := filepath.Abs(location); err == nil {
		secrets = append(secrets, abs)
	}
	if m := uriUserinfo.FindStringSubmatch(location); m != nil {
		secrets = append(secrets, m[1])
		user, password, hasPassword := strings.Cut(m[1], ":")
		secrets = append(secrets, user)
		if hasPassword {
			secrets = append(secrets, password)
			if plain, err := url.PathUnescape(password); err == nil {
				secrets = append(secrets, plain)
			}
		}
	}
	if m := uriHost.FindStringSubmatch(location); m != nil {
		secrets = append(secrets, m[1], hostWithoutPort(m[1]))
	}
	for _, m := range keywordSecret.FindAllStringSubmatch(location, -1) {
		secrets = append(secrets, m[1], strings.Trim(m[1], "'"))
	}
	kept := secrets[:0]
	for _, secret := range secrets {
		if secret != "" {
			kept = append(kept, secret)
		}
	}
	return kept
}

func hostWithoutPort(hostPort string) string {
	if strings.HasPrefix(hostPort, "[") {
		if end := strings.Index(hostPort, "]"); end > 0 {
			return hostPort[:end+1]
		}
	}
	host, _, _ := strings.Cut(hostPort, ":")
	return host
}
