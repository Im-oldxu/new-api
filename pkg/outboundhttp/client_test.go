package outboundhttp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type staticSSRFResolver map[string][]net.IPAddr

func (r staticSSRFResolver) LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error) {
	if ips, ok := r[host]; ok {
		return ips, nil
	}
	return nil, fmt.Errorf("unexpected lookup for %s", host)
}

func staticProtection(protection *SSRFProtection) func() (*SSRFProtection, bool, error) {
	return func() (*SSRFProtection, bool, error) {
		return protection, true, nil
	}
}

func testConn(t *testing.T) net.Conn {
	t.Helper()
	clientConn, serverConn := net.Pipe()
	t.Cleanup(func() {
		clientConn.Close()
		serverConn.Close()
	})
	return clientConn
}

func mustParseURL(t *testing.T, rawURL string) *url.URL {
	t.Helper()
	parsedURL, err := url.Parse(rawURL)
	require.NoError(t, err)
	return parsedURL
}

func TestProtectedFetchDialerRejectsPrivateReboundAddress(t *testing.T) {
	dialer := &protectedFetchDialer{
		resolver: staticSSRFResolver{
			"safe.example": {{IP: net.ParseIP("127.0.0.1")}},
		},
		dialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			t.Fatalf("dialContext should not be called for blocked address %s", address)
			return nil, nil
		},
		getProtection: staticProtection(&SSRFProtection{
			AllowPrivateIp:         false,
			DomainFilterMode:       false,
			IpFilterMode:           false,
			ApplyIPFilterForDomain: true,
		}),
	}

	conn, err := dialer.DialContext(context.Background(), "tcp", "safe.example:80")

	require.Error(t, err)
	require.Nil(t, conn)
	require.Contains(t, err.Error(), "private IP address not allowed")
}

func TestProtectedFetchDialerRejectsMixedResolvedIPs(t *testing.T) {
	var dialed []string
	dialer := &protectedFetchDialer{
		resolver: staticSSRFResolver{
			"safe.example": {
				{IP: net.ParseIP("10.0.0.1")},
				{IP: net.ParseIP("8.8.8.8")},
			},
		},
		dialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			dialed = append(dialed, address)
			return testConn(t), nil
		},
		getProtection: staticProtection(&SSRFProtection{
			AllowPrivateIp:         false,
			DomainFilterMode:       false,
			IpFilterMode:           false,
			ApplyIPFilterForDomain: true,
		}),
	}

	conn, err := dialer.DialContext(context.Background(), "tcp", "safe.example:443")
	require.Error(t, err)
	require.Nil(t, conn)

	require.Empty(t, dialed)
	require.Contains(t, err.Error(), "private IP address not allowed")
}

func TestProtectedFetchDialerDialsWhenAllResolvedIPsAllowed(t *testing.T) {
	var dialed []string
	dialer := &protectedFetchDialer{
		resolver: staticSSRFResolver{
			"safe.example": {
				{IP: net.ParseIP("8.8.8.8")},
				{IP: net.ParseIP("1.1.1.1")},
			},
		},
		dialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			dialed = append(dialed, address)
			return testConn(t), nil
		},
		getProtection: staticProtection(&SSRFProtection{
			AllowPrivateIp:         false,
			DomainFilterMode:       false,
			IpFilterMode:           false,
			ApplyIPFilterForDomain: true,
		}),
	}

	conn, err := dialer.DialContext(context.Background(), "tcp", "safe.example:443")
	require.NoError(t, err)
	require.NotNil(t, conn)

	require.Equal(t, []string{"8.8.8.8:443"}, dialed)
}

func TestProtectedFetchDialerAllowsPrivateIPWhenWhitelisted(t *testing.T) {
	var dialed []string
	dialer := &protectedFetchDialer{
		resolver: staticSSRFResolver{
			"internal.example": {{IP: net.ParseIP("10.1.2.3")}},
		},
		dialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			dialed = append(dialed, address)
			return testConn(t), nil
		},
		getProtection: staticProtection(&SSRFProtection{
			AllowPrivateIp:         true,
			DomainFilterMode:       false,
			IpFilterMode:           true,
			IpList:                 []string{"10.0.0.0/8"},
			ApplyIPFilterForDomain: true,
		}),
	}

	conn, err := dialer.DialContext(context.Background(), "tcp", "internal.example:80")
	require.NoError(t, err)
	require.NotNil(t, conn)

	require.Equal(t, []string{"10.1.2.3:80"}, dialed)
}

func TestProtectedFetchDialerSkipsResolutionWhenDomainIPFilterIsDisabled(t *testing.T) {
	var dialed []string
	dialer := &protectedFetchDialer{
		resolver: staticSSRFResolver{},
		dialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			dialed = append(dialed, address)
			return testConn(t), nil
		},
		getProtection: staticProtection(&SSRFProtection{
			AllowPrivateIp:         false,
			DomainFilterMode:       false,
			IpFilterMode:           false,
			ApplyIPFilterForDomain: false,
		}),
	}

	conn, err := dialer.DialContext(context.Background(), "tcp", "safe.example:80")
	require.NoError(t, err)
	require.NotNil(t, conn)

	require.Equal(t, []string{"safe.example:80"}, dialed)
}

func TestProtectedFetchRoundTripperUsesConfiguredProxy(t *testing.T) {
	proxyURL := mustParseURL(t, "http://127.0.0.1:3128")
	var dialed []string
	client := newProtectedFetchHTTPClient(
		staticSSRFResolver{},
		func(ctx context.Context, network, address string) (net.Conn, error) {
			dialed = append(dialed, address)
			return nil, errors.New("stop after proxy dial")
		},
		staticProtection(&SSRFProtection{
			AllowPrivateIp:         false,
			DomainFilterMode:       false,
			IpFilterMode:           false,
			ApplyIPFilterForDomain: true,
		}),
		func(req *http.Request) (*url.URL, error) {
			return proxyURL, nil
		},
		nil,
	)
	req, err := http.NewRequest(http.MethodGet, "http://93.184.216.34/resource", nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.Error(t, err)
	require.Nil(t, resp)
	require.Equal(t, []string{"127.0.0.1:3128"}, dialed)
}

func TestProtectedFetchRoundTripperRejectsPrivateTargetBeforeProxy(t *testing.T) {
	proxyURL := mustParseURL(t, "http://127.0.0.1:3128")
	var dialed []string
	client := newProtectedFetchHTTPClient(
		staticSSRFResolver{"localhost": {{IP: net.ParseIP("127.0.0.1")}}},
		func(ctx context.Context, network, address string) (net.Conn, error) {
			dialed = append(dialed, address)
			return nil, errors.New("proxy should not be dialed")
		},
		staticProtection(&SSRFProtection{
			AllowPrivateIp:         false,
			DomainFilterMode:       false,
			IpFilterMode:           false,
			ApplyIPFilterForDomain: true,
		}),
		func(req *http.Request) (*url.URL, error) {
			return proxyURL, nil
		},
		nil,
	)
	req, err := http.NewRequest(http.MethodGet, "http://localhost/resource", nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.Error(t, err)
	require.Nil(t, resp)
	require.Contains(t, err.Error(), "private IP address not allowed")
	require.Empty(t, dialed)
}

func TestProtectedFetchRoundTripperNoProxyUsesProtectedDialer(t *testing.T) {
	var dialed []string
	client := newProtectedFetchHTTPClient(
		staticSSRFResolver{},
		func(ctx context.Context, network, address string) (net.Conn, error) {
			dialed = append(dialed, address)
			return nil, errors.New("unexpected direct dial")
		},
		staticProtection(&SSRFProtection{
			AllowPrivateIp:         false,
			DomainFilterMode:       false,
			IpFilterMode:           false,
			ApplyIPFilterForDomain: true,
		}),
		func(req *http.Request) (*url.URL, error) {
			return nil, nil
		},
		nil,
	)
	req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1/resource", nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.Error(t, err)
	require.Nil(t, resp)
	require.Contains(t, err.Error(), "private IP address not allowed")
	require.Empty(t, dialed)
}

func TestProtectedFetchRoundTripperReusesTransportPerProxy(t *testing.T) {
	client := NewProtectedClient(staticProtection(&SSRFProtection{}), nil)
	roundTripper, ok := client.Transport.(*ssrfProtectedRoundTripper)
	require.True(t, ok)

	direct := roundTripper.transportFor(nil, true)
	directAgain := roundTripper.transportFor(nil, true)
	proxied := roundTripper.transportFor(mustParseURL(t, "http://127.0.0.1:3128"), true)

	require.Same(t, direct, directAgain)
	require.NotSame(t, direct, proxied)
	require.NotSame(t, direct, roundTripper.transportFor(nil, false))
	require.True(t, direct.ForceAttemptHTTP2)
	require.False(t, direct.DisableKeepAlives)
}

func TestProtectedClientValidatesRedirectTargets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/allowed":
			http.Redirect(writer, request, "/result", http.StatusFound)
		case "/denied":
			http.Redirect(writer, request, "http://blocked.test/result", http.StatusFound)
		case "/loop":
			http.Redirect(writer, request, "/loop", http.StatusFound)
		default:
			_, _ = io.WriteString(writer, "result")
		}
	}))
	t.Cleanup(server.Close)
	var dialer net.Dialer
	client := newProtectedFetchHTTPClient(
		staticSSRFResolver{
			"allowed.test": {{IP: net.ParseIP("93.184.216.34")}},
			"blocked.test": {{IP: net.ParseIP("::1")}},
		},
		func(ctx context.Context, network, address string) (net.Conn, error) {
			if address != "93.184.216.34:80" {
				return nil, fmt.Errorf("unexpected dial destination %s", address)
			}
			return dialer.DialContext(ctx, network, server.Listener.Addr().String())
		},
		staticProtection(&SSRFProtection{ApplyIPFilterForDomain: true}),
		func(*http.Request) (*url.URL, error) { return nil, nil },
		nil,
	)
	t.Cleanup(client.CloseIdleConnections)
	response, err := client.Get("http://allowed.test/allowed")
	require.NoError(t, err)
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	assert.Equal(t, "result", string(body))
	_, err = client.Get("http://allowed.test/denied")
	require.ErrorContains(t, err, "redirect blocked: private IP address not allowed")
	_, err = client.Get("http://allowed.test/loop")
	require.ErrorContains(t, err, "stopped after 10 redirects")
}
