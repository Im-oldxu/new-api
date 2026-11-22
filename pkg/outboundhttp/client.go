package outboundhttp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

const (
	networkTimeout = 30 * time.Second
	requestTimeout = 2 * time.Minute
)

type PolicySource func() (*SSRFProtection, bool, error)

type ssrfResolver interface {
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

type protectedFetchDialer struct {
	resolver      ssrfResolver
	dialContext   func(ctx context.Context, network, address string) (net.Conn, error)
	getProtection PolicySource
}

type ssrfProtectedRoundTripper struct {
	resolver      ssrfResolver
	dialContext   func(ctx context.Context, network, address string) (net.Conn, error)
	getProtection PolicySource
	proxy         func(*http.Request) (*url.URL, error)
	tlsConfig     *tls.Config

	mutex      sync.Mutex
	transports map[transportKey]*http.Transport
}

type transportKey struct {
	proxy     string
	protected bool
}

func NewClient(checkRedirect func(*http.Request, []*http.Request) error, tlsConfig *tls.Config) *http.Client {
	transport := newTransport(tlsConfig)
	return &http.Client{Transport: transport, CheckRedirect: checkRedirect, Timeout: requestTimeout}
}

func NewProtectedClient(getProtection PolicySource, tlsConfig *tls.Config) *http.Client {
	return newProtectedFetchHTTPClient(nil, nil, getProtection, http.ProxyFromEnvironment, tlsConfig)
}

func newProtectedFetchHTTPClient(resolver ssrfResolver, dialContext func(ctx context.Context, network, address string) (net.Conn, error), getProtection PolicySource, proxy func(*http.Request) (*url.URL, error), tlsConfig *tls.Config) *http.Client {
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	if dialContext == nil {
		netDialer := &net.Dialer{
			Timeout:   networkTimeout,
			KeepAlive: 30 * time.Second,
		}
		dialContext = netDialer.DialContext
	}
	transport := &ssrfProtectedRoundTripper{
		resolver:      resolver,
		dialContext:   dialContext,
		getProtection: getProtection,
		proxy:         proxy,
		tlsConfig:     tlsConfig,
		transports:    make(map[transportKey]*http.Transport),
	}
	return &http.Client{Transport: transport, CheckRedirect: transport.checkRedirect, Timeout: requestTimeout}
}

func (t *ssrfProtectedRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req == nil {
		return nil, fmt.Errorf("invalid request")
	}
	protected, err := t.validateRequest(req)
	if err != nil {
		if req.Body != nil {
			req.Body.Close()
		}
		return nil, err
	}

	proxyURL, err := t.proxy(req)
	if err != nil {
		if req.Body != nil {
			req.Body.Close()
		}
		return nil, err
	}
	return t.transportFor(proxyURL, protected).RoundTrip(req)
}

func (t *ssrfProtectedRoundTripper) validateRequest(req *http.Request) (bool, error) {
	if req.URL == nil {
		return false, fmt.Errorf("invalid request URL")
	}
	protection, enabled, err := t.getProtection()
	if err != nil || !enabled {
		return enabled, err
	}
	return true, protection.validateURL(req.Context(), req.URL, t.resolver)
}

func (t *ssrfProtectedRoundTripper) checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return fmt.Errorf("stopped after 10 redirects")
	}
	if _, err := t.validateRequest(req); err != nil {
		return fmt.Errorf("redirect blocked: %w", err)
	}
	return nil
}

func (t *ssrfProtectedRoundTripper) CloseIdleConnections() {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	for _, transport := range t.transports {
		transport.CloseIdleConnections()
	}
}

func (t *ssrfProtectedRoundTripper) transportFor(proxyURL *url.URL, protected bool) *http.Transport {
	// 只按代理地址分组：代理来自环境变量，取值有限，map 有界；
	// 目标 origin 是用户可控输入，不能作为缓存 key。
	key := transportKey{protected: protected}
	if proxyURL != nil {
		key.proxy = proxyURL.String()
	}
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if transport, ok := t.transports[key]; ok {
		return transport
	}

	transport := t.newTransport(proxyURL, protected)
	t.transports[key] = transport
	return transport
}

func (t *ssrfProtectedRoundTripper) newTransport(proxyURL *url.URL, protected bool) *http.Transport {
	transport := newTransport(t.tlsConfig)
	transport.DialContext = t.dialContext
	transport.Proxy = nil
	if proxyURL != nil {
		transport.Proxy = http.ProxyURL(proxyURL)
	} else if protected {
		protectedDialer := &protectedFetchDialer{
			resolver:      t.resolver,
			dialContext:   t.dialContext,
			getProtection: t.getProtection,
		}
		transport.DialContext = protectedDialer.DialContext
	}
	return transport
}

func newTransport(tlsConfig *tls.Config) *http.Transport {
	dialer := &net.Dialer{Timeout: networkTimeout, KeepAlive: 30 * time.Second}
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		TLSClientConfig:       tlsConfig.Clone(),
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: networkTimeout,
		ExpectContinueTimeout: time.Second,
		ForceAttemptHTTP2:     true,
	}
}

func (d *protectedFetchDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	protection, enabled, err := d.getProtection()
	if err != nil {
		return nil, err
	}
	if !enabled {
		return nil, fmt.Errorf("fetch protection changed before dialing")
	}

	host, portText, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("invalid dial address %s: %w", addr, err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		return nil, fmt.Errorf("invalid port: %s", portText)
	}
	if err := protection.ValidateNetworkTarget(host, port); err != nil {
		return nil, err
	}

	if ip := net.ParseIP(host); ip != nil {
		return d.dialContext(ctx, network, net.JoinHostPort(ip.String(), portText))
	}
	if !protection.ApplyIPFilterForDomain {
		return d.dialContext(ctx, network, addr)
	}

	lookupContext, cancel := context.WithTimeout(ctx, networkTimeout)
	resolved, err := d.resolver.LookupIPAddr(lookupContext, host)
	cancel()
	if err != nil {
		return nil, fmt.Errorf("DNS resolution failed for %s: %w", host, err)
	}

	var candidateIPs []net.IP
	for _, ipAddr := range resolved {
		ip := ipAddr.IP
		if ip == nil || !networkAllowsIP(network, ip) {
			continue
		}
		if err := protection.ValidateResolvedIP(host, ip); err != nil {
			return nil, err
		}
		candidateIPs = append(candidateIPs, ip)
	}

	var lastDialErr error
	for _, ip := range candidateIPs {
		conn, err := d.dialContext(ctx, network, net.JoinHostPort(ip.String(), portText))
		if err == nil {
			return conn, nil
		}
		lastDialErr = err
	}

	if lastDialErr != nil {
		return nil, lastDialErr
	}
	return nil, fmt.Errorf("DNS resolution for %s returned no usable IP addresses", host)
}

func networkAllowsIP(network string, ip net.IP) bool {
	switch network {
	case "tcp4":
		return ip.To4() != nil
	case "tcp6":
		return ip.To4() == nil
	default:
		return true
	}
}
