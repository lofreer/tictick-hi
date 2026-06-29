package errtext

import (
	"net/url"
	"regexp"
	"strings"
)

const maxExternalErrorRunes = 500

var (
	externalErrorTransportPattern = regexp.MustCompile(`\b(?:Get|Post|Put|Patch|Delete|Head|Options) "(https?://[^"]+)": ([^;]+)`)
	externalErrorURLPattern       = regexp.MustCompile(`https?://[^\s"'<>]+`)
)

func ExternalError(value string) string {
	normalized := strings.Join(strings.Fields(value), " ")
	if normalized == "" {
		return ""
	}
	sanitized := externalErrorTransportPattern.ReplaceAllStringFunc(normalized, func(raw string) string {
		matches := externalErrorTransportPattern.FindStringSubmatch(raw)
		if len(matches) != 3 {
			return "[external-url]"
		}
		return externalErrorHost(matches[1]) + ": " + strings.TrimSpace(matches[2])
	})
	sanitized = externalErrorURLPattern.ReplaceAllStringFunc(sanitized, externalErrorHost)
	runes := []rune(sanitized)
	if len(runes) <= maxExternalErrorRunes {
		return sanitized
	}
	return string(runes[:maxExternalErrorRunes-3]) + "..."
}

func externalErrorHost(raw string) string {
	parsed, err := url.Parse(raw)
	if err == nil && parsed.Host != "" {
		return parsed.Host
	}
	return "[external-url]"
}
