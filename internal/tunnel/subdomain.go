package tunnel

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base32"
	"strings"
)

const (
	// Base32 is used because its alphabet (a-z, 2-7) is a valid DNS label as-is,
	// unlike base64url which allows underscores. 26 characters is 130 bits.
	subdomainLength = 26

	// Guarantees the label starts with a letter and makes tunnels obvious in logs.
	subdomainPrefix = "d"
)

// Subdomain derives the one subdomain an application is entitled to.
//
// riseact-core computes the same value with the same algorithm and refuses any
// other, so the two implementations must agree byte for byte — see
// riseact/controlplane/tunnel/selectors.py. HMAC rather than a cipher: the value
// never needs to be reversed, and a deterministic cipher over variable-length
// input would mean either a fixed nonce or ECB.
func Subdomain(clientID, clientSecret string) string {
	mac := hmac.New(sha256.New, []byte(clientSecret))
	mac.Write([]byte(clientID))

	encoded := strings.ToLower(
		base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(mac.Sum(nil)),
	)

	return subdomainPrefix + encoded[:subdomainLength]
}
