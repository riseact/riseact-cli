// Package tunnel exposes a partner's local dev server on a public HTTPS URL,
// so it can be loaded inside the Riseact iframe. See TUNNEL.md in this package
// for how the pieces fit together.
package tunnel

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fatedier/frp/client"
	"github.com/fatedier/frp/pkg/config"
	"github.com/fatedier/frp/pkg/config/source"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	frplog "github.com/fatedier/frp/pkg/util/log"
	"github.com/samber/lo"
)

// Config describes the tunnel to open. Everything here is known before the
// tunnel exists: the hostname is derived, not assigned by the server.
type Config struct {
	// ControlHost is the frps control channel, e.g. tunnel.riseact.org. It is
	// reached over WSS on 443 through the shared Caddy.
	ControlHost string

	// Zone is the DNS zone tunnels are published under, e.g. tun.riseact.org.
	Zone string

	// Token is the coarse frp gate. It ships inside this binary, so treat it as
	// public — riseact-core does the real authorization.
	Token string

	// ClientID and ClientSecret are the app's OAuth credentials. They are sent
	// to frps as metadata and forwarded to riseact-core, which verifies them and
	// checks that Subdomain is the one this app may bind.
	ClientID     string
	ClientSecret string

	// LocalPort is where the CLI's reverse proxy is listening.
	LocalPort int

	// LogPath is where frp's own diagnostics are written. frp logs through a
	// package-level logger that would otherwise print to the terminal in its own
	// format, on top of ours. Sending it to a file keeps the output clean while
	// leaving a trail to ask a partner for. Defaults to the temp directory.
	LogPath string
}

// frp configures logging through a package-level singleton, so this happens once
// per process regardless of how many tunnels are opened.
var initFrpLogging sync.Once

func setupLogging(logPath string) {
	initFrpLogging.Do(func() {
		if logPath == "" {
			logPath = filepath.Join(os.TempDir(), "riseact-tunnel.log")
		}

		frplog.InitLogger(logPath, "info", 3, true)
	})
}

// Tunnel is a running tunnel. Close stops it.
type Tunnel struct {
	url    string
	cancel context.CancelFunc
	done   chan error
}

// URL is the public HTTPS address the app is reachable at.
func (t *Tunnel) URL() string {
	return t.url
}

// Close tears the tunnel down. The public URL stops working immediately and
// starts serving frps' "no dev tunnel here" page.
func (t *Tunnel) Close() {
	t.cancel()

	select {
	case <-t.done:
	case <-time.After(3 * time.Second):
	}
}

// Start opens the tunnel and blocks until it is bound and authorized, or fails.
//
// It does not wait for the TLS certificate — call Warm for that.
func Start(cfg Config) (*Tunnel, error) {
	if cfg.ControlHost == "" || cfg.Zone == "" {
		return nil, fmt.Errorf("tunnel is not configured for this environment")
	}

	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, fmt.Errorf("app credentials are missing, cannot open a tunnel")
	}

	setupLogging(cfg.LogPath)

	subdomain := Subdomain(cfg.ClientID, cfg.ClientSecret)
	url := fmt.Sprintf("https://%s.%s", subdomain, cfg.Zone)

	common, proxies, err := buildConfig(cfg, subdomain)
	if err != nil {
		return nil, err
	}

	configSource := source.NewConfigSource()
	if err := configSource.ReplaceAll(proxies, nil); err != nil {
		return nil, fmt.Errorf("cannot set tunnel configuration: %w", err)
	}

	svc, err := client.NewService(client.ServiceOptions{
		Common:                 common,
		ConfigSourceAggregator: source.NewAggregator(configSource),
		ConnectorCreator: func(ctx context.Context, c *v1.ClientCommonConfig) client.Connector {
			return newConnector(ctx, c)
		},
	})
	if err != nil {
		return nil, fmt.Errorf("cannot create tunnel client: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)

	go func() {
		done <- svc.Run(ctx)
		close(done)
	}()

	t := &Tunnel{url: url, cancel: cancel, done: done}

	if err := waitUntilBound(t, cfg, subdomain); err != nil {
		t.Close()
		return nil, err
	}

	return t, nil
}

// waitUntilBound blocks until the proxy is registered on the server, so callers
// never advertise a URL that was refused.
//
// frp keeps retrying a rejected proxy forever rather than surfacing an error, so
// this polls the client's own view of its proxies instead of waiting on Run.
func waitUntilBound(t *Tunnel, cfg Config, subdomain string) error {
	deadline := time.After(30 * time.Second)
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case err := <-t.done:
			if err != nil {
				return fmt.Errorf("tunnel stopped: %w", err)
			}
			return fmt.Errorf("tunnel stopped before it was established")

		case <-deadline:
			return fmt.Errorf(
				"tunnel was not authorized within 30s: check that app %s is registered and its credentials are current",
				cfg.ClientID,
			)

		case <-ticker.C:
			if reachable(t.url) {
				return nil
			}
		}
	}
}

// reachable reports whether the public URL is being served by our tunnel rather
// than frps' not-found page. A TLS error means the certificate does not exist
// yet, which is expected on a first run and handled by Warm.
func reachable(url string) bool {
	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// frps answers 404 for a hostname with no client attached.
	return resp.StatusCode != http.StatusNotFound
}

func buildConfig(cfg Config, subdomain string) (*v1.ClientCommonConfig, []v1.ProxyConfigurer, error) {
	common := &v1.ClientCommonConfig{
		ServerAddr: cfg.ControlHost,
		ServerPort: 443,
		Auth: v1.AuthClientConfig{
			Method: v1.AuthMethodToken,
			Token:  cfg.Token,
		},
		// Forwarded to riseact-core by the authorization plugin as
		// content.user.metas. The secret never leaves the TLS connection.
		Metadatas: map[string]string{
			"client_id":     cfg.ClientID,
			"client_secret": cfg.ClientSecret,
		},
	}

	// "wss" would make frp build its own TLS config, which skips verification
	// without a CA file. Our connector does TLS and the WebSocket handshake
	// itself, so the transport is plain as far as frp is concerned.
	common.Transport.Protocol = "tcp"
	common.Transport.TLS.Enable = lo.ToPtr(false)
	common.Transport.TLS.ServerName = cfg.ControlHost
	common.Transport.TCPMux = lo.ToPtr(true)
	common.Transport.TCPMuxKeepaliveInterval = 30

	common.Complete()

	proxy := &v1.HTTPProxyConfig{
		ProxyBaseConfig: v1.ProxyBaseConfig{
			Name: "riseact-dev",
			Type: string(v1.ProxyTypeHTTP),
			ProxyBackend: v1.ProxyBackend{
				LocalIP:   "127.0.0.1",
				LocalPort: cfg.LocalPort,
			},
		},
		DomainConfig: v1.DomainConfig{
			SubDomain: subdomain,
		},
	}

	proxies := config.CompleteProxyConfigurers([]v1.ProxyConfigurer{proxy})

	return common, proxies, nil
}
