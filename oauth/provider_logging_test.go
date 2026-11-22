package oauth

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Im-oldxu/new-api/common"
	"github.com/Im-oldxu/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenericOAuthExchangeTokenLogsOnlyDiagnosticMetadata(t *testing.T) {
	var output bytes.Buffer
	previousWriter, previousDebug := gin.DefaultErrorWriter, common.DebugEnabled
	gin.DefaultErrorWriter, common.DebugEnabled = &output, true
	t.Cleanup(func() { gin.DefaultErrorWriter, common.DebugEnabled = previousWriter, previousDebug })
	providerServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.NoError(t, request.ParseForm())
		assert.Equal(t, "synthetic-client-secret", request.Form.Get("client_secret"))
		assert.Equal(t, "synthetic-auth-code", request.Form.Get("code"))
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path == "/reject" {
			writer.WriteHeader(http.StatusBadRequest)
			_, _ = writer.Write([]byte(`{"error":"synthetic-provider-error","error_description":"synthetic-provider-description"}`))
			return
		}
		_, _ = writer.Write([]byte(`{"access_token":"synthetic-access-token","scope":"synthetic-provider-scope"}`))
	}))
	t.Cleanup(providerServer.Close)
	config := &model.CustomOAuthProvider{
		Slug:          "test-provider",
		Name:          "Test Provider",
		ClientSecret:  "synthetic-client-secret",
		TokenEndpoint: providerServer.URL + "/token?client-secret=synthetic-endpoint-secret",
	}
	provider := NewGenericOAuthProvider(config)
	ctx := context.WithValue(t.Context(), common.RequestIdKey, "test-request-id")

	token, err := provider.ExchangeToken(ctx, "synthetic-auth-code", nil)
	require.NoError(t, err)
	assert.Equal(t, "synthetic-access-token", token.AccessToken)

	config.TokenEndpoint = providerServer.URL + "/reject"
	token, err = provider.ExchangeToken(ctx, "synthetic-auth-code", nil)
	assert.Nil(t, token)
	var oauthErr *OAuthError
	require.ErrorAs(t, err, &oauthErr)
	assert.NotContains(t, err.Error(), "synthetic-")

	config.TokenEndpoint = providerServer.URL + "/token?client-secret=synthetic-transport-secret"
	cancelledCtx, cancel := context.WithCancel(ctx)
	cancel()
	_, err = provider.ExchangeToken(cancelledCtx, "synthetic-auth-code", nil)
	require.ErrorAs(t, err, &oauthErr)
	assert.NotContains(t, err.Error(), "synthetic-")

	logged := output.String()
	assert.Contains(t, logged, "test-request-id")
	assert.Contains(t, logged, "[OAuth-Generic-test-provider] ExchangeToken success")
	assert.Contains(t, logged, "ExchangeToken provider rejected request: status=400")
	assert.Contains(t, logged, "ExchangeToken request failed")
	assert.NotContains(t, logged, "synthetic-")
}
