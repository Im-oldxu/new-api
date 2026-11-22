package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Im-oldxu/new-api/common"
	"github.com/Im-oldxu/new-api/setting/system_setting"
)

// WorkerRequest Worker请求的数据结构
type WorkerRequest struct {
	URL     string            `json:"url"`
	Key     string            `json:"key"`
	Method  string            `json:"method,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    json.RawMessage   `json:"body,omitempty"`
}

// DoWorkerRequest 通过Worker发送请求
func DoWorkerRequest(req *WorkerRequest) (*http.Response, error) {
	if !system_setting.EnableWorker() {
		return nil, fmt.Errorf("worker not enabled")
	}
	if !system_setting.WorkerAllowHttpImageRequestEnabled && !strings.HasPrefix(req.URL, "https") {
		return nil, fmt.Errorf("only support https url")
	}

	// SSRF防护：验证请求URL
	protection, enabled, err := currentFetchProtection()
	if err != nil {
		return nil, fmt.Errorf("request reject: %w", err)
	}
	if enabled {
		if err := protection.ValidateURL(context.Background(), req.URL); err != nil {
			return nil, fmt.Errorf("request reject: %w", err)
		}
	}

	workerUrl := system_setting.WorkerUrl
	if !strings.HasSuffix(workerUrl, "/") {
		workerUrl += "/"
	}

	// 序列化worker请求数据
	workerPayload, err := common.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal worker payload: %v", err)
	}

	return workerHTTPClient().Post(workerUrl, "application/json", bytes.NewReader(workerPayload))
}

func DoDownloadRequest(originUrl string, reason ...string) (resp *http.Response, err error) {
	if system_setting.EnableWorker() {
		common.SysLog(common.LogText("downloading file from worker, reason: %s", strings.Join(reason, ", ")))
		req := &WorkerRequest{
			URL: originUrl,
			Key: system_setting.WorkerValidKey,
		}
		return DoWorkerRequest(req)
	} else {
		common.SysLog(common.LogText("downloading from origin, reason: %s", strings.Join(reason, ", ")))
		request, err := http.NewRequest(http.MethodGet, originUrl, nil)
		if err != nil {
			return nil, fmt.Errorf("invalid download URL")
		}
		return DoProtectedFetch(request)
	}
}
