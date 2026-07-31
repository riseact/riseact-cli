package app

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// ReverseProxy sits between the tunnel and the partner's dev server, on a local
// port. Everything goes to the app server, websocket upgrades included: the SDK
// installs its own upgrade handler and forwards HMR traffic to Vite itself, so
// there is nothing to route separately here.
type ReverseProxy struct {
	listener net.Listener
	proxy    *httputil.ReverseProxy
}

// NewReverseProxy binds a listener on an ephemeral local port. Ephemeral rather
// than fixed so two projects can run side by side without colliding.
func NewReverseProxy(targetURL string) (*ReverseProxy, error) {
	target, err := url.Parse(targetURL)

	if err != nil {
		return nil, fmt.Errorf("invalid app url %q: %w", targetURL, err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")

	if err != nil {
		return nil, fmt.Errorf("cannot open a local port: %w", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	// The dev server is usually still booting when the first request arrives, and
	// the default behaviour is to log a Go stack-flavoured error to stderr on
	// every one. Say something a partner can act on instead, once per request.
	proxy.ErrorLog = log.New(io.Discard, "", 0)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(w, "The dev server at %s is not answering yet.\n", target)
	}

	return &ReverseProxy{listener: listener, proxy: proxy}, nil
}

// Port is the local port the proxy listens on.
func (rp *ReverseProxy) Port() int {
	return rp.listener.Addr().(*net.TCPAddr).Port
}

// Serve blocks until the listener is closed.
func (rp *ReverseProxy) Serve() error {
	return http.Serve(rp.listener, rp.proxy)
}

// Close stops accepting connections.
func (rp *ReverseProxy) Close() error {
	return rp.listener.Close()
}
