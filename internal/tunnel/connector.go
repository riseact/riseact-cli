package tunnel

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	libnet "github.com/fatedier/golib/net"
	fmux "github.com/hashicorp/yamux"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	netpkg "github.com/fatedier/frp/pkg/util/net"
)

// connector replaces frp's default connector for one reason: frp builds its
// client TLS config with NewClientTLSConfig, which sets InsecureSkipVerify to
// true whenever transport.tls.trustedCaFile is empty, and offers no way to pass
// a *tls.Config in. Since the OAuth client_secret travels over this connection,
// it has to be verified.
//
// The alternative was shipping a CA bundle inside the binary and pointing
// trustedCaFile at it. This is smaller and follows the operating system's trust
// store instead, so it keeps up with revocations without a CLI release.
//
// The dial chain is assembled from frp's own building blocks and mirrors the
// "wss" branch of client/connector.go: TLS at priority 100, the WebSocket
// handshake at 110 so it runs over the already-encrypted connection.
type connector struct {
	ctx        context.Context
	serverAddr string
	serverPort int
	serverName string
	keepalive  time.Duration

	muxSession *fmux.Session
	closeOnce  sync.Once
}

func newConnector(ctx context.Context, cfg *v1.ClientCommonConfig) *connector {
	serverName := cfg.Transport.TLS.ServerName
	if serverName == "" {
		serverName = cfg.ServerAddr
	}

	return &connector{
		ctx:        ctx,
		serverAddr: cfg.ServerAddr,
		serverPort: cfg.ServerPort,
		serverName: serverName,
		keepalive:  time.Duration(cfg.Transport.TCPMuxKeepaliveInterval) * time.Second,
	}
}

func (c *connector) Open() error {
	conn, err := c.dial()
	if err != nil {
		return err
	}

	fmuxCfg := fmux.DefaultConfig()
	fmuxCfg.KeepAliveInterval = c.keepalive
	fmuxCfg.MaxStreamWindowSize = 6 * 1024 * 1024
	// yamux insists on one of Logger or LogOutput; its chatter is not useful here.
	fmuxCfg.LogOutput = io.Discard

	session, err := fmux.Client(conn, fmuxCfg)
	if err != nil {
		conn.Close()
		return err
	}

	c.muxSession = session

	return nil
}

func (c *connector) Connect() (net.Conn, error) {
	if c.muxSession != nil {
		return c.muxSession.OpenStream()
	}

	return c.dial()
}

func (c *connector) Close() error {
	c.closeOnce.Do(func() {
		if c.muxSession != nil {
			_ = c.muxSession.Close()
		}
	})

	return nil
}

func (c *connector) dial() (net.Conn, error) {
	// No RootCAs means the system trust store, and ServerName is what the
	// certificate is checked against.
	tlsConfig := &tls.Config{
		ServerName: c.serverName,
		MinVersion: tls.VersionTLS12,
	}

	// "tcp" rather than "wss": the TLS hook below already encrypts the
	// connection, so the WebSocket handshake itself must use the ws:// scheme.
	// This is what frp's own wss branch does.
	websocketHook := netpkg.DialHookWebsocket("tcp", c.serverName)

	return libnet.DialContext(
		c.ctx,
		net.JoinHostPort(c.serverAddr, strconv.Itoa(c.serverPort)),
		libnet.WithProtocol("tcp"),
		libnet.WithTimeout(15*time.Second),
		libnet.WithTLSConfigAndPriority(100, tlsConfig),
		libnet.WithAfterHook(libnet.AfterHook{Hook: websocketHook, Priority: 110}),
	)
}
