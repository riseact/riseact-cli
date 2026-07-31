package tunnel

import (
	"fmt"
	"net/http"
	"time"
)

// Warm forces Caddy to issue the TLS certificate for this tunnel's hostname.
//
// The tunnel zone uses on-demand TLS: Caddy obtains a certificate during the
// first TLS handshake for a hostname it has not seen, holding the handshake open
// for roughly eight seconds while it talks to Let's Encrypt. Leaving that to the
// partner's browser means a blank iframe with no explanation, which reads as a
// broken app.
//
// Doing it here also removes a real hazard. Caddy asks riseact-core whether a
// hostname may have a certificate, and the answer is only yes while a tunnel is
// bound. A request arriving before that gets refused, and a failed issuance puts
// Caddy into exponential backoff for that name — up to a day between retries. By
// warming the certificate before the URL is advertised, nothing can arrive first.
//
// Subsequent runs return immediately: the certificate is cached server-side and
// renewed in the background.
func Warm(url string, onSlow func()) error {
	const (
		timeout       = 90 * time.Second
		slowAfter     = 2 * time.Second
		retryInterval = 2 * time.Second
	)

	client := &http.Client{Timeout: timeout}
	deadline := time.Now().Add(timeout)

	notified := false
	notifyIfSlow := time.AfterFunc(slowAfter, func() {
		if onSlow != nil {
			onSlow()
		}
	})
	defer notifyIfSlow.Stop()

	var lastErr error

	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			return nil
		}

		lastErr = err
		notified = true

		time.Sleep(retryInterval)
	}

	if !notified {
		return fmt.Errorf("could not reach %s", url)
	}

	// A TLS error here almost always means Caddy was refused a certificate,
	// which in turn means riseact-core does not consider the tunnel bound.
	return fmt.Errorf(
		"HTTPS certificate for %s was not issued within %s: %w\n"+
			"Check that the tunnel is authorized:\n"+
			"  curl -o /dev/null -w '%%{http_code}' 'https://core.riseact.org/__infra/domain/check/?domain=%s'",
		url, timeout, lastErr, hostOf(url),
	)
}

func hostOf(url string) string {
	host := url

	for _, prefix := range []string{"https://", "http://"} {
		if len(host) > len(prefix) && host[:len(prefix)] == prefix {
			host = host[len(prefix):]
			break
		}
	}

	for i := 0; i < len(host); i++ {
		if host[i] == '/' {
			return host[:i]
		}
	}

	return host
}
