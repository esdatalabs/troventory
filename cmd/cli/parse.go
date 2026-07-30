package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/uuid"
)

// newID returns a fresh correlation/idempotency identifier.
func newID() string {
	id, err := uuid.NewV4()
	if err != nil {
		// crypto/rand exhausted — effectively never happens; fall back to
		// a timestamp so the CLI can still proceed.
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	return id.String()
}

// tokenize splits line into shell-like tokens, honoring "double quoted
// substrings" as a single token so multi-word values (descriptions,
// vendors, location names) can be entered naturally.
func tokenize(line string) ([]string, error) {
	var tokens []string
	var cur strings.Builder
	inQuotes := false
	started := false

	flush := func() {
		if started {
			tokens = append(tokens, cur.String())
			cur.Reset()
			started = false
		}
	}

	for i := 0; i < len(line); i++ {
		c := line[i]
		switch {
		case c == '"':
			inQuotes = !inQuotes
			started = true
		case c == ' ' || c == '\t':
			if inQuotes {
				cur.WriteByte(c)
			} else {
				flush()
			}
		default:
			cur.WriteByte(c)
			started = true
		}
	}
	if inQuotes {
		return nil, fmt.Errorf("unterminated quote")
	}
	flush()

	return tokens, nil
}

// parseCents parses a decimal dollar amount ("49.99", "5", "49.9") into
// integer minor-unit cents, per ARCHITECTURE.md §6 (never a float64 for a
// monetary amount — the conversion happens once, here, at the input
// boundary, and never again).
func parseCents(s string) (int64, error) {
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")

	whole, frac, hasFrac := strings.Cut(s, ".")
	if whole == "" {
		whole = "0"
	}
	wholeCents, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount %q", s)
	}

	var fracCents int64
	if hasFrac {
		switch len(frac) {
		case 0:
			fracCents = 0
		case 1:
			d, err := strconv.ParseInt(frac, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid amount %q", s)
			}
			fracCents = d * 10
		default:
			d, err := strconv.ParseInt(frac[:2], 10, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid amount %q", s)
			}
			fracCents = d
		}
	}

	cents := wholeCents*100 + fracCents
	if neg {
		cents = -cents
	}
	return cents, nil
}

// formatCents renders amountCents/currency as a human-readable "12.34 USD"
// string.
func formatCents(amountCents int64, currency string) string {
	neg := ""
	if amountCents < 0 {
		neg = "-"
		amountCents = -amountCents
	}
	return fmt.Sprintf("%s%d.%02d %s", neg, amountCents/100, amountCents%100, currency)
}

// formatMoneyOrDash is formatCents, except an empty currency (no value
// recorded/computed) renders as "—" rather than a misleading "0.00".
func formatMoneyOrDash(amountCents int64, currency string) string {
	if currency == "" {
		return "—"
	}
	return formatCents(amountCents, currency)
}

// today returns the current date as a "YYYY-MM-DD" string.
func today() string {
	return time.Now().UTC().Format("2006-01-02")
}
