package app

import (
	"bytes"
	"testing"
)

func TestClearFilterDropsScreenErasingSequences(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "erase screen",
			in:   "before\x1b[2Jafter",
			want: "beforeafter",
		},
		{
			name: "erase scrollback",
			in:   "before\x1b[3Jafter",
			want: "beforeafter",
		},
		{
			name: "full reset",
			in:   "before\x1bcafter",
			want: "beforeafter",
		},
		{
			name: "cursor home then erase, as console.clear emits it",
			in:   "before\x1b[H\x1b[2Jafter",
			want: "before\x1b[Hafter",
		},
		{
			name: "colours survive",
			in:   "\x1b[34m[SRV]\x1b[0m ready",
			want: "\x1b[34m[SRV]\x1b[0m ready",
		},
		{
			name: "erase to end of screen is left alone",
			in:   "keep\x1b[0Jthis",
			want: "keep\x1b[0Jthis",
		},
		{
			name: "erase in line is left alone",
			in:   "keep\x1b[2Kthis",
			want: "keep\x1b[2Kthis",
		},
		{
			name: "nothing to do",
			in:   "plain output\n",
			want: "plain output\n",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var out bytes.Buffer

			f := newClearFilter(&out)
			n, err := f.Write([]byte(c.in))

			if err != nil {
				t.Fatalf("Write() error: %v", err)
			}

			if n != len(c.in) {
				t.Errorf("Write() = %d, want %d — a short count reads as an error", n, len(c.in))
			}

			if out.String() != c.want {
				t.Errorf("got %q, want %q", out.String(), c.want)
			}
		})
	}
}

// A sequence split across writes must still be recognised, which is the case a
// naive per-write scan gets wrong.
func TestClearFilterHandlesSequenceSplitAcrossWrites(t *testing.T) {
	var out bytes.Buffer

	f := newClearFilter(&out)

	for _, chunk := range []string{"before\x1b", "[2", "Jafter"} {
		if _, err := f.Write([]byte(chunk)); err != nil {
			t.Fatalf("Write(%q) error: %v", chunk, err)
		}
	}

	if got := out.String(); got != "beforeafter" {
		t.Errorf("got %q, want %q", got, "beforeafter")
	}
}

func TestClearFilterFlushesAnUnterminatedEscape(t *testing.T) {
	var out bytes.Buffer

	f := newClearFilter(&out)

	// Longer than the lookahead cap, so it must be released rather than buffered
	// for ever.
	junk := "\x1b[" + string(bytes.Repeat([]byte("1"), maxEscapeLookahead+8))

	if _, err := f.Write([]byte(junk)); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	if out.Len() == 0 {
		t.Error("an unterminated escape longer than the cap was swallowed")
	}
}
