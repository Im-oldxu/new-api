package common

import (
	"net/url"
	"strings"
)

var logQueryKeySeparators = strings.NewReplacer("-", "", "_", "", ".", "")

func SanitizeURLForLog(rawURL string) string {
	if rawURL == "" {
		return rawURL
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil || parsedURL.Opaque != "" {
		return "***masked***"
	}

	query, queryErr := url.ParseQuery(parsedURL.RawQuery)
	changed := parsedURL.User != nil || strings.Contains(rawURL, "#") || queryErr != nil
	parsedURL.User = nil
	parsedURL.Fragment = ""
	parsedURL.RawFragment = ""
	for key := range query {
		if isSensitiveURLQueryKey(key) {
			query.Set(key, "***masked***")
			changed = true
		}
	}
	if !changed {
		return rawURL
	}

	parsedURL.RawQuery = query.Encode()
	return parsedURL.String()
}

func isSensitiveURLQueryKey(key string) bool {
	normalized := logQueryKeySeparators.Replace(strings.ToLower(strings.TrimSpace(key)))
	switch normalized {
	case "key",
		"access",
		"authorization",
		"auth",
		"code",
		"codeverifier",
		"codechallenge",
		"clientassertion",
		"assertion",
		"state",
		"nonce",
		"otp",
		"totp",
		"turnstile",
		"cfturnstileresponse",
		"recoverycode",
		"verificationcode",
		"cookie",
		"session",
		"sessionid",
		"sig",
		"awsaccesskeyid":
		return true
	}
	return strings.Contains(normalized, "token") ||
		strings.Contains(normalized, "secret") ||
		strings.Contains(normalized, "signature") ||
		strings.Contains(normalized, "credential") ||
		strings.Contains(normalized, "password") ||
		strings.Contains(normalized, "passwd") ||
		strings.Contains(normalized, "apikey") ||
		strings.Contains(normalized, "privatekey")
}
