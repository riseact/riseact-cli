// Package tunnel exposes a partner's local dev server on a public HTTPS URL,
// so it can be loaded inside the Riseact iframe. See TUNNEL.md in this package
// for how the pieces fit together.
package tunnel

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fatedier/frp/client"
	"github.com/fatedier/frp/client/proxy"
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

	// OnProgress, when set, receives a short line for each step of bringing the
	// tunnel up. Establishing one takes a couple of network round trips, and on a
	// first run also waits on certificate issuance — without this the CLI looks
	// stalled for several seconds with nothing on screen.
	OnProgress func(message string)

	// LogPath is where frp's own diagnostics are written. frp logs through a
	// package-level logger that would otherwise print to the terminal in its own
	// format, on top of ours. Sending it to a file keeps the output clean while
	// leaving a trail to ask a partner for. Defaults to the temp directory.
	LogPath string
}

const (
	// proxyName identifies our single proxy to frp, and is what the status lookup
	// below asks about.
	proxyName = "riseact-dev"

	// bindTimeout bounds the wait for the server to accept the tunnel.
	bindTimeout = 30 * time.Second
)

func (c Config) progress(message string) {
	if c.OnProgress != nil {
		c.OnProgress(message)
	}
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

	cfg.progress(fmt.Sprintf("Opening the tunnel to %s (subdomain %s)...", cfg.ControlHost, subdomain))

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

	if err := waitUntilBound(svc, t); err != nil {
		t.Close()
		return nil, err
	}

	cfg.progress("Tunnel established")

	return t, nil
}

// waitUntilBound blocks until the server has accepted the proxy, so callers never
// advertise a URL that was refused.
//
// It reads the client's own view of the proxy rather than probing the public URL.
// Probing was wrong in two ways. On a first run the probe itself triggers
// certificate issuance, which takes longer than any sensible probe timeout, so the
// CLI spun silently for tens of seconds. And a refusal was indistinguishable from
// "not ready yet", so a rejected tunnel surfaced as a generic timeout instead of
// the reason the server actually gave.
func waitUntilBound(svc *client.Service, t *Tunnel) error {
	deadline := time.After(bindTimeout)
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	status := svc.StatusExporter()

	for {
		select {
		case err := <-t.done:
			if err != nil {
				return fmt.Errorf("tunnel stopped: %w", err)
			}
			return fmt.Errorf("tunnel stopped before it was established")

		case <-deadline:
			return fmt.Errorf("the tunnel was not established within %s", bindTimeout)

		case <-ticker.C:
			s, ok := status.GetProxyStatus(proxyName)

			if !ok {
				continue
			}

			switch s.Phase {
			case proxy.ProxyPhaseRunning:
				return nil

			case proxy.ProxyPhaseStartErr, proxy.ProxyPhaseClosed:
				// Err carries what the server said, which for a refusal is the
				// reason riseact-core gave.
				if s.Err != "" {
					return fmt.Errorf("the tunnel was refused: %s", s.Err)
				}

				return fmt.Errorf("the tunnel could not be established (%s)", s.Phase)
			}
		}
	}
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
			Name: proxyName,
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
