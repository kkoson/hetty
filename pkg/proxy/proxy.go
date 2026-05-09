// Package proxy provides an HTTP/HTTPS proxy server for intercepting
// and inspecting HTTP traffic.
package proxy

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"time"
)

// Config holds configuration options for the proxy server.
type Config struct {
	// Addr is the address the proxy listens on (e.g., ":8080").
	Addr string
	// CAKeyFile is the path to the CA private key file used for TLS interception.
	CAKeyFile string
	// CACertFile is the path to the CA certificate file used for TLS interception.
	CACertFile string
	// OnRequest is an optional hook called when a request is intercepted.
	OnRequest func(req *http.Request)
	// OnResponse is an optional hook called when a response is received.
	OnResponse func(req *http.Request, resp *http.Response)
}

// Proxy is an HTTP/HTTPS intercepting proxy.
type Proxy struct {
	config Config
	server *http.Server
	transport *http.Transport
}

// New creates a new Proxy with the given configuration.
func New(cfg Config) (*Proxy, error) {
	if cfg.Addr == "" {
		cfg.Addr = ":8080"
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   60 * time.Second, // increased from 30s for slower networks
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     false,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false, //nolint:gosec
		},
	}

	p := &Proxy{
		config:    cfg,
		transport: transport,
	}

	p.server = &http.Server{
		Addr:         cfg.Addr,
		Handler:      p,
		ReadTimeout:  60 * time.Second, // increased from 30s; some large requests need more time
		WriteTimeout: 60 * time.Second, // increased from 30s
		IdleTimeout:  60 * time.Second,
	}

	return p, nil
}

// Start begins listening and serving proxy requests.
func (p *Proxy) Start() error {
	log.Printf("[INFO] Proxy server listening on %s", p.config.Addr)
	return p.server.ListenAndServe()
}

// Close gracefully shuts down the proxy server.
func (p *Proxy) Close() error {
	return p.server.Close()
}

// ServeHTTP implements http.Handler and routes requests to the appropriate handler.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		p.handleConnect(w, r)
		return
	}
	p.handleHTTP(w, r)
}

// handleHTTP proxies a plain HTTP request to the target server.
func (p *Proxy) handleHTTP(w http.ResponseWriter, r *http.Request) {
	if p.config.OnRequest != nil {
		p.config.OnRequest(r)
	}

	rp := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			// Ensure the request URL has a scheme.
			if req.URL.Scheme == "" {
				req.URL.Scheme = "http"
			}
			if req.URL.Host == "" {
				req.URL.Host = req.Host
			}
			req.Header.Del("Proxy-Connection")
		},
		Transport: p.transport,
		ModifyResponse: func(resp *http.Response) error {
			if p.config.OnResponse != nil {
				p.config.OnResponse(r, resp)
			}
			return nil
		},
		ErrorHandler: func(w http.Resp