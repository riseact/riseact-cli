package app

import (
	"io"
	"strings"
)

// maxEscapeLookahead caps how much of a partial escape sequence is held back
// waiting for the rest of it. Well past any real sequence, so malformed output
// cannot make this buffer without bound.
const maxEscapeLookahead = 32

// clearFilter passes a child process's output through, minus the escape
// sequences that erase the terminal.
//
// Dev servers commonly clear the screen on every restart — ts-node-dev does it
// with --clear, and vite has its own — which wipes what the CLI printed before
// handing over, including the tunnel URL. That URL is derived and not shown
// anywhere else in the terminal, so losing it means the partner has to restart
// to see it again.
//
// Only the destructive sequences are dropped: a full reset, and erasing the
// screen or the scrollback. Colours, cursor movement and everything else pass
// through untouched.
type clearFilter struct {
	w io.Writer

	// pending holds a sequence that was cut in half by a write boundary.
	pending []byte
}

func newClearFilter(w io.Writer) *clearFilter {
	return &clearFilter{w: w}
}

func (f *clearFilter) Write(p []byte) (int, error) {
	// Report the caller's whole slice as written: what gets dropped is dropped on
	// purpose, and a short count would be read as an error.
	written := len(p)

	buf := p

	if len(f.pending) > 0 {
		buf = append(f.pending, p...)
		f.pending = nil
	}

	out := make([]byte, 0, len(buf))

	for i := 0; i < len(buf); {
		if buf[i] != 0x1b {
			out = append(out, buf[i])
			i++
			continue
		}

		length, drop, complete := escapeAt(buf[i:])

		if !complete {
			// Hold the tail back until the rest arrives, unless it has grown past
			// anything plausible.
			if len(buf)-i <= maxEscapeLookahead {
				f.pending = append([]byte(nil), buf[i:]...)
				break
			}

			out = append(out, buf[i])
			i++

			continue
		}

		if !drop {
			out = append(out, buf[i:i+length]...)
		}

		i += length
	}

	if len(out) > 0 {
		if _, err := f.w.Write(out); err != nil {
			return 0, err
		}
	}

	return written, nil
}

// escapeAt inspects the escape sequence starting at b[0], reporting its length,
// whether it erases the terminal, and whether all of it is present yet.
func escapeAt(b []byte) (length int, drop bool, complete bool) {
	if len(b) < 2 {
		return 0, false, false
	}

	// ESC c — full reset.
	if b[1] == 'c' {
		return 2, true, true
	}

	if b[1] != '[' {
		// Some other escape; hand over the two bytes and carry on.
		return 2, false, true
	}

	// CSI: parameter bytes, then a final byte that identifies the command.
	erasesScreen := false

	for i := 2; i < len(b); i++ {
		c := b[i]

		if c >= '0' && c <= '9' || c == ';' {
			// ED takes 2 to erase the screen and 3 to erase the scrollback. 0 and 1
			// only clear around the cursor and are left alone.
			if c == '2' || c == '3' {
				erasesScreen = true
			}

			continue
		}

		if c >= 0x40 && c <= 0x7e {
			return i + 1, c == 'J' && erasesScreen, true
		}

		// Not a valid CSI after all.
		return i + 1, false, true
	}

	return 0, false, false
}

// forceColor tells the child to keep its colours even though its output is a
// pipe rather than a terminal. Without this, filtering the output would cost the
// coloured prefixes the dev server uses to say which process a line came from.
//
// Existing values are left as they are: a partner who deliberately turned colour
// off should stay that way.
func forceColor(env []string) []string {
	wanted := map[string]string{
		"FORCE_COLOR":    "1",
		"CLICOLOR_FORCE": "1",
	}

	for _, entry := range env {
		name, _, found := strings.Cut(entry, "=")

		if !found {
			continue
		}

		delete(wanted, name)
	}

	out := append([]string(nil), env...)

	for name, value := range wanted {
		out = append(out, name+"="+value)
	}

	return out
}
