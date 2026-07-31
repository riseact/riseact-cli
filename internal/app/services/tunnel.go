package services

import (
	"fmt"
	"os"
	"path/filepath"
	"riseact/internal/app"
	"riseact/internal/config"
	"riseact/internal/tunnel"
	"riseact/internal/utils/logger"
)

// devAccess is how a browser reaches the local dev server. In production that
// means a tunnel; against a local riseact-core it is just the local address,
// because both ends are plain http on the same machine and the iframe can load
// the app directly.
type devAccess struct {
	URL string

	tunnel *tunnel.Tunnel
}

func (d *devAccess) Close() {
	if d.tunnel != nil {
		d.tunnel.Close()
	}
}

// startDevAccess resolves the app, makes localPort reachable and points the app
// at the resulting URL.
//
// The order matters when a tunnel is involved. The hostname is derived from the
// app's OAuth credentials, so the app has to be resolved first; and the TLS
// certificate is warmed before app_url is updated, so nothing can request the
// URL before Caddy is able to serve it. See internal/tunnel/TUNNEL.md.
func startDevAccess(localPort int) (*app.Application, *devAccess, error) {
	settings := config.GetAppSettings()

	a, appEnv, err := initApp()

	if err != nil {
		logger.Debugf("Error initializing app: %s", err.Error())
		return nil, nil, err
	}

	access, err := openAccess(settings, a, appEnv, localPort)

	if err != nil {
		return nil, nil, err
	}

	// Deriving the hostname from credentials means it never changes, so this is
	// idempotent after the first run.
	if err := a.UpdateAppUris(access.URL); err != nil {
		access.Close()
		return nil, nil, fmt.Errorf("cannot update the app urls: %w", err)
	}

	appEnv.RiseactAppUrl = access.URL
	appEnv.Store()

	return a, access, nil
}

func openAccess(
	settings *config.AppSettings,
	a *app.Application,
	appEnv *app.AppEnv,
	localPort int,
) (*devAccess, error) {
	if settings.TunnelControlHost == "" {
		url := fmt.Sprintf("http://localhost:%d", localPort)
		logger.Debugf("No tunnel for this environment, serving directly on %s", url)

		return &devAccess{URL: url}, nil
	}

	clientID, clientSecret := credentials(a, appEnv)

	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("app credentials are missing, cannot open a tunnel")
	}

	tun, err := tunnel.Start(tunnel.Config{
		ControlHost:  settings.TunnelControlHost,
		Zone:         settings.TunnelZone,
		Token:        settings.TunnelToken,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		LocalPort:    localPort,
		LogPath:      tunnelLogPath(),
	})

	if err != nil {
		return nil, err
	}

	// On a first run Caddy issues the certificate during this call, which takes
	// a few seconds. Warming it here keeps that wait in the terminal, where it
	// can be explained, instead of in a blank iframe.
	err = tunnel.Warm(tun.URL(), func() {
		logger.Info("Preparing the HTTPS certificate, this happens once per app...")
	})

	if err != nil {
		tun.Close()
		return nil, err
	}

	return &devAccess{URL: tun.URL(), tunnel: tun}, nil
}

// tunnelLogPath keeps frp's own diagnostics next to the CLI configuration, so
// they survive across runs and can be asked for when a partner reports trouble.
func tunnelLogPath() string {
	home, err := os.UserHomeDir()

	if err != nil {
		return ""
	}

	return filepath.Join(home, ".config", "riseact-tunnel.log")
}

// credentials prefers what the project stores locally, falling back to whatever
// the API returned. riseact-core derives the subdomain from the same pair, so a
// mismatch here shows up as a refused tunnel.
func credentials(a *app.Application, appEnv *app.AppEnv) (string, string) {
	clientID := appEnv.ClientId
	clientSecret := appEnv.ClientSecret

	if clientID == "" {
		clientID = a.ClientId
	}

	if clientSecret == "" {
		clientSecret = a.ClientSecret
	}

	return clientID, clientSecret
}
