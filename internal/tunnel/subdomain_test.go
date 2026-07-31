package tunnel

import "testing"

// These vectors were produced by riseact-core's tunnel_subdomain() and verified
// against the running production endpoint. If this test fails, the CLI and
// riseact-core no longer agree and every tunnel will be refused.
func TestSubdomainMatchesRiseactCore(t *testing.T) {
	cases := []struct {
		name         string
		clientID     string
		clientSecret string
		want         string
	}{
		{
			name:         "short values",
			clientID:     "abc123",
			clientSecret: "s3cr3t",
			want:         "da2elnq7cd3ubisugdeswazpeei",
		},
		{
			name:         "dev-sdk production app",
			clientID:     "m0a5V1qFzOwW26ft6o6Sgr3Tasgzoe3KV5k7emoe",
			clientSecret: "aCIvRSrUo55UKY7LYNnsKmFeUahT6kDGDq8SImHUnqLvBMdm2zArjirGQTaJ4ISBDBfRPgvxmHqzDvqVsIptqw8shsu4gZGEH6X3Jp1tO6kMIZo8yvxN9VRLR5uM0IQs",
			want:         "dlpzehjmv2nzr6l552z4ccntu2f",
		},
		{
			name:         "empty values",
			clientID:     "",
			clientSecret: "",
			want:         "dwyjwpgqictm6y5zpsxlxrq27yx",
		},
		{
			name:         "utf-8 values",
			clientID:     "unicode-àèì",
			clientSecret: "chiave-con-é",
			want:         "d6uajjdrwnezljnvxojfvpptnrb",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Subdomain(c.clientID, c.clientSecret)
			if got != c.want {
				t.Errorf("Subdomain() = %q, want %q", got, c.want)
			}
		})
	}
}

func TestSubdomainIsAValidDNSLabel(t *testing.T) {
	got := Subdomain("some-client-id", "some-client-secret")

	if len(got) != subdomainLength+len(subdomainPrefix) {
		t.Errorf("length = %d, want %d", len(got), subdomainLength+len(subdomainPrefix))
	}

	if got[0] < 'a' || got[0] > 'z' {
		t.Errorf("label starts with %q, want a letter", got[0])
	}

	for i, r := range got {
		valid := (r >= 'a' && r <= 'z') || (r >= '2' && r <= '7')
		if !valid {
			t.Errorf("character %q at %d is not a valid DNS label character", r, i)
		}
	}
}

func TestSubdomainDependsOnBothInputs(t *testing.T) {
	base := Subdomain("id", "secret")

	if Subdomain("id2", "secret") == base {
		t.Error("changing the client id did not change the subdomain")
	}

	if Subdomain("id", "secret2") == base {
		t.Error("changing the client secret did not change the subdomain")
	}
}
