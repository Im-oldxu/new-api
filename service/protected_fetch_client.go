package service

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/Im-oldxu/new-api/common"
	"github.com/Im-oldxu/new-api/pkg/outboundhttp"
	"github.com/Im-oldxu/new-api/setting/system_setting"
)

var ssrfProtectedHTTPClient = sync.OnceValue(func() *http.Client {
	return outboundhttp.NewProtectedClient(currentFetchProtection, outboundTLSConfig())
})

var workerHTTPClient = sync.OnceValue(func() *http.Client {
	return outboundhttp.NewClient(checkRedirect, outboundTLSConfig())
})

func outboundTLSConfig() *tls.Config {
	if common.TLSInsecureSkipVerify {
		return common.InsecureTLSConfig.Clone()
	}
	return nil
}

func currentFetchProtection() (*outboundhttp.SSRFProtection, bool, error) {
	fetchSetting := system_setting.GetFetchSetting()
	if !fetchSetting.EnableSSRFProtection {
		return nil, false, nil
	}
	protection, err := outboundhttp.NewSSRFProtectionFromFetchSetting(
		fetchSetting.AllowPrivateIp,
		fetchSetting.DomainFilterMode,
		fetchSetting.IpFilterMode,
		fetchSetting.DomainList,
		fetchSetting.IpList,
		fetchSetting.AllowedPorts,
		fetchSetting.ApplyIPFilterForDomain,
	)
	return protection, true, err
}

func checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return fmt.Errorf("stopped after 10 redirects")
	}
	protection, enabled, err := currentFetchProtection()
	if err != nil || !enabled {
		return err
	}
	if err := protection.ValidateURL(req.Context(), req.URL.String()); err != nil {
		return fmt.Errorf("redirect blocked: %w", err)
	}
	return nil
}

// DoProtectedFetch applies the deployment's outbound request policy and omits
// the target URL from transport errors.
func DoProtectedFetch(req *http.Request) (*http.Response, error) {
	resp, err := ssrfProtectedHTTPClient().Do(req)
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return resp, fmt.Errorf("%s: %w", urlErr.Op, urlErr.Err)
	}
	return resp, err
}
