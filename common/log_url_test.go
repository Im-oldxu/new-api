package common

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizeURLForLogMasksSensitiveQueryValues(t *testing.T) {
	rawURL := "https://example.test/path?alt=sse&key=sk-secret&access_token=ya29-secret&api-version=2024-02-01"

	got := SanitizeURLForLog(rawURL)

	assert.NotContains(t, got, "sk-secret")
	assert.NotContains(t, got, "ya29-secret")
	parsedURL, err := url.Parse(got)
	require.NoError(t, err)
	query := parsedURL.Query()
	assert.Equal(t, "***masked***", query.Get("key"))
	assert.Equal(t, "***masked***", query.Get("access_token"))
	assert.Equal(t, "sse", query.Get("alt"))
	assert.Equal(t, "2024-02-01", query.Get("api-version"))
}

func TestSanitizeURLForLogMasksAWSAndSecretLikeQueryKeys(t *testing.T) {
	rawURL := "https://example.test/path?X-Amz-Credential=credential&X-Amz-Signature=signature&session_token=session&client_secret=secret&model=gpt-test"

	got := SanitizeURLForLog(rawURL)

	assert.NotContains(t, got, "X-Amz-Credential=credential")
	assert.NotContains(t, got, "X-Amz-Signature=signature")
	assert.NotContains(t, got, "session_token=session")
	assert.NotContains(t, got, "client_secret=secret")
	parsedURL, err := url.Parse(got)
	require.NoError(t, err)
	query := parsedURL.Query()
	assert.Equal(t, "***masked***", query.Get("X-Amz-Credential"))
	assert.Equal(t, "***masked***", query.Get("X-Amz-Signature"))
	assert.Equal(t, "***masked***", query.Get("session_token"))
	assert.Equal(t, "***masked***", query.Get("client_secret"))
	assert.Equal(t, "gpt-test", query.Get("model"))
}

func TestSanitizeURLForLogKeepsURLWithoutSensitiveQuery(t *testing.T) {
	rawURL := "https://example.test/path?api-version=2024-02-01&alt=sse"

	got := SanitizeURLForLog(rawURL)

	assert.Equal(t, rawURL, got)
}

func TestSanitizeURLForLogOmitsCredentialsFromUserInfoAndMalformedQueries(t *testing.T) {
	assert.Equal(t, "https://example.test/path?keep=yes", SanitizeURLForLog("https://user:secret@example.test/path?access=token%ZZ&keep=yes"))
	assert.Equal(t, "***masked***", SanitizeURLForLog("https://%ZZ:secret@example.test/path"))
}

func TestSanitizeURLForLogFiltersAuthenticationParametersAndFragments(t *testing.T) {
	parameters := []string{"code", "state", "code-verifier", "Client.Assertion", "newPassword", "private-key", "session_id", "otp"}
	query := url.Values{"page": {"2"}}
	for _, parameter := range parameters {
		query.Set(parameter, "synthetic-credential")
	}
	sanitized := SanitizeURLForLog("https://example.test/path?" + query.Encode() + "#access_token=synthetic-fragment")
	parsed, err := url.Parse(sanitized)
	require.NoError(t, err)
	assert.Empty(t, parsed.Fragment)
	assert.NotContains(t, sanitized, "synthetic-")
	loggedQuery := parsed.Query()
	assert.Equal(t, "2", loggedQuery.Get("page"))
	for _, parameter := range parameters {
		assert.Equal(t, "***masked***", loggedQuery.Get(parameter), parameter)
	}
	assert.Equal(t, "***masked***", SanitizeURLForLog("urn:synthetic-credential"))
}
