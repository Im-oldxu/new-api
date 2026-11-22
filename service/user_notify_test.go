package service

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Im-oldxu/new-api/common"
	"github.com/Im-oldxu/new-api/dto"
	"github.com/Im-oldxu/new-api/setting/system_setting"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneralOutboundHTTP(t *testing.T) {
	fetch := system_setting.GetFetchSetting()
	previousFetch := *fetch
	previousWorker, previousWorkerKey := system_setting.WorkerUrl, system_setting.WorkerValidKey
	previousWorkerHTTP := system_setting.WorkerAllowHttpImageRequestEnabled
	t.Cleanup(func() {
		*fetch = previousFetch
		system_setting.WorkerUrl, system_setting.WorkerValidKey = previousWorker, previousWorkerKey
		system_setting.WorkerAllowHttpImageRequestEnabled = previousWorkerHTTP
	})
	*fetch = system_setting.FetchSetting{EnableSSRFProtection: true, ApplyIPFilterForDomain: true}
	system_setting.WorkerUrl = ""
	notification := dto.Notify{Type: "test", Title: "notification", Content: "Changed {{value}}", Values: []any{"language"}}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/webhook":
			payload, err := io.ReadAll(request.Body)
			assert.NoError(t, err)
			assert.Equal(t, http.MethodPost, request.Method)
			assert.Equal(t, generateSignature("test-signing-key", payload), request.Header.Get("X-Webhook-Signature"))
			var notice WebhookPayload
			if assert.NoError(t, common.Unmarshal(payload, &notice)) {
				assert.Equal(t, "test", notice.Type)
				assert.Equal(t, "notification", notice.Title)
				assert.Equal(t, "Changed language", notice.Content)
				assert.Equal(t, []any{"language"}, notice.Values)
				assert.Positive(t, notice.Timestamp)
			}
			writer.WriteHeader(http.StatusNoContent)
		case "/":
			var workerRequest WorkerRequest
			assert.NoError(t, common.DecodeJson(request.Body, &workerRequest))
			assert.Equal(t, "http://93.184.216.34/download", workerRequest.URL)
			assert.Equal(t, "test-worker-key", workerRequest.Key)
			_, _ = io.WriteString(writer, "download contents")
		case "/download":
			_, _ = io.WriteString(writer, "download contents")
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	assert.ErrorContains(t, SendWebhookNotify(server.URL+"/webhook", "test-signing-key", notification), "private IP address not allowed")
	_, err := DoDownloadRequest(server.URL + "/download?token=test-download-secret")
	require.ErrorContains(t, err, "private IP address not allowed")
	assert.NotContains(t, err.Error(), "test-download-secret")

	fetch.EnableSSRFProtection = false
	require.NoError(t, SendWebhookNotify(server.URL+"/webhook", "test-signing-key", notification))
	for _, viaWorker := range []bool{false, true} {
		origin := server.URL + "/download"
		if viaWorker {
			fetch.EnableSSRFProtection = true
			system_setting.WorkerUrl = server.URL
			system_setting.WorkerValidKey = "test-worker-key"
			system_setting.WorkerAllowHttpImageRequestEnabled = true
			origin = "http://93.184.216.34/download"
		}
		response, err := DoDownloadRequest(origin)
		require.NoError(t, err)
		defer response.Body.Close()
		body, err := io.ReadAll(response.Body)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Equal(t, "download contents", string(body))
	}
}
