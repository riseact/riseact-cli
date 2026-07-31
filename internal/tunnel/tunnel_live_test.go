package tunnel

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// TestLiveTunnel opens a real tunnel against the production frps and checks that
// traffic comes back. It exercises the custom connector — our own TLS, the
// WebSocket handshake and the yamux session — which nothing else covers.
//
// Skipped unless credentials are supplied:
//
//	RISEACT_TUNNEL_LIVE=1 \
//	RISEACT_TUNNEL_CLIENT_ID=... \
//	RISEACT_TUNNEL_CLIENT_SECRET=... \
//	RISEACT_TUNNEL_TOKEN=... \
//	go test ./internal/tunnel/ -run TestLiveTunnel -v
func TestLiveTunnel(t *testing.T) {
	if os.Getenv("RISEACT_TUNNEL_LIVE") == "" {
		t.Skip("set RISEACT_TUNNEL_LIVE=1 and the credential env vars to run this")
	}

	clientID := os.Getenv("RISEACT_TUNNEL_CLIENT_ID")
	clientSecret := os.Getenv("RISEACT_TUNNEL_CLIENT_SECRET")
	token := os.Getenv("RISEACT_TUNNEL_TOKEN")

	if clientID == "" || clientSecret == "" || token == "" {
		t.Fatal("RISEACT_TUNNEL_CLIENT_ID, RISEACT_TUNNEL_CLIENT_SECRET and RISEACT_TUNNEL_TOKEN are required")
	}

	const body = "RISEACT-CLI-TUNNEL-OK"

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("cannot open a local port: %v", err)
	}
	defer listener.Close()

	go http.Serve(listener, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, body)
	}))

	localPort := listener.Addr().(*net.TCPAddr).Port

	start := time.Now()

	tun, err := Start(Config{
		ControlHost:  "tunnel.riseact.org",
		Zone:         "tun.riseact.org",
		Token:        token,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		LocalPort:    localPort,
	})
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer tun.Close()

	t.Logf("tunnel bound in %s: %s", time.Since(start).Round(time.Millisecond), tun.URL())

	if want := Subdomain(clientID, clientSecret); !strings.Contains(tun.URL(), want) {
		t.Errorf("URL %q does not contain the derived subdomain %q", tun.URL(), want)
	}

	warmStart := time.Now()
	if err := Warm(tun.URL(), func() { t.Log("certificate issuance in progress...") }); err != nil {
		t.Fatalf("Warm() failed: %v", err)
	}
	t.Logf("certificate ready in %s", time.Since(warmStart).Round(time.Millisecond))

	resp, err := http.Get(tun.URL())
	if err != nil {
		t.Fatalf("GET %s failed: %v", tun.URL(), err)
	}
	defer resp.Body.Close()

	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("cannot read the response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", resp.StatusCode, got)
	}

	if string(got) != body {
		t.Errorf("body = %q, want %q", got, body)
	}
}
