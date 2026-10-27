package fracindex

import (
	"errors"
	"fmt"
	"strings"
)

// digits are the base-62 digits in byte order (CONTRACTS §3).
const digits = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

const base = len(digits)

// smallestInteger is the lowest integer part. A key that is only this integer
// is invalid, so there is always room to insert before any valid key.
var smallestInteger = "A" + strings.Repeat("0", 26)

// ErrInvalidKey reports a key that is not a well-formed fractional index key.
var ErrInvalidKey = errors.New("fracindex: invalid key")

// ErrOrder reports bounds where a is not less than b.
var ErrOrder = errors.New("fracindex: a must be less than b")

// Valid reports whether key is a well-formed fractional index key: an integer
// part whose head character encodes its length, then an optional fractional
// part with no trailing '0' (CONTRACTS §3).
func Valid(key string) bool {
	if key == "" || key == smallestInteger {
		return false
	}
	n := integerLength(key[0])
	if n == 0 || n > len(key) {
		return false
	}
	for i := 1; i < len(key); i++ {
		if digitValue(key[i]) < 0 {
			return false
		}
	}
	return len(key) == n || key[len(key)-1] != '0'
}

// Between returns a key k with a < k < b (CONTRACTS §3). An empty a or b is an
// open end; Between("", "") returns "a0".
func Between(a, b string) (string, error) {
	if a != "" && !Valid(a) {
		return "", fmt.Errorf("%w: %q", ErrInvalidKey, a)
	}
	if b != "" && !Valid(b) {
		return "", fmt.Errorf("%w: %q", ErrInvalidKey, b)
	}
	if a != "" && b != "" && a >= b {
		return "", fmt.Errorf("%w: %q >= %q", ErrOrder, a, b)
	}

	switch {
	case a == "" && b == "":
		return "a0", nil
	case a == "":
		ib := b[:integerLength(b[0])]
		if ib == smallestInteger {
			return ib + midpoint("", b[len(ib):]), nil
		}
		if ib < b {
			return ib, nil
		}
		// b is a bare integer above the smallest, so it can be decremented.
		// The reference algorithm returns the bare smallest integer when b is
		// the one above it, but that key is invalid; add a fraction instead.
		if i := decrementInteger(ib); i != smallestInteger {
			return i, nil
		}
		return smallestInteger + midpoint("", ""), nil
	case b == "":
		ia := a[:integerLength(a[0])]
		if i, ok := incrementInteger(ia); ok {
			return i, nil
		}
		return ia + midpoint(a[len(ia):], ""), nil
	}

	ia := a[:integerLength(a[0])]
	ib := b[:integerLength(b[0])]
	if ia == ib {
		return ia + midpoint(a[len(ia):], b[len(ib):]), nil
	}
	// ia < ib, so ia is not the largest integer and the increment succeeds.
	i, _ := incrementInteger(ia)
	if i < b {
		return i, nil
	}
	return ia + midpoint(a[len(ia):], ""), nil
}

// NBetween returns n strictly increasing keys between a and b, spread evenly
// by bisection (CONTRACTS §3). Empty bounds are open ends as in Between.
func NBetween(a, b string, n int) ([]string, error) {
	if n < 0 {
		return nil, fmt.Errorf("fracindex: negative count %d", n)
	}
	keys := make([]string, 0, n)
	switch {
	case n == 0:
		return keys, nil
	case b == "":
		// Appending one after another keeps keys short at the open end.
		c := a
		for range n {
			k, err := Between(c, b)
			if err != nil {
				return nil, err
			}
			keys = append(keys, k)
			c = k
		}
		return keys, nil
	case a == "":
		c := b
		for range n {
			k, err := Between(a, c)
			if err != nil {
				return nil, err
			}
			keys = append(keys, k)
			c = k
		}
		for i, j := 0, len(keys)-1; i < j; i, j = i+1, j-1 {
			keys[i], keys[j] = keys[j], keys[i]
		}
		return keys, nil
	}
	return bisect(keys, a, b, n)
}

// bisect appends n keys in (a, b) to keys, placing the middle one first so the
// keys spread evenly.
func bisect(keys []string, a, b string, n int) ([]string, error) {
	if n == 0 {
		return keys, nil
	}
	c, err := Between(a, b)
	if err != nil {
		return nil, err
	}
	mid := n / 2
	if keys, err = bisect(keys, a, c, mid); err != nil {
		return nil, err
	}
	keys = append(keys, c)
	return bisect(keys, c, b, n-mid-1)
}

// midpoint returns a fractional part strictly between fractional parts a and
// b, where b == "" means 1. Both have no trailing '0' and a < b.
func midpoint(a, b string) string {
	if b != "" {
		// Skip the common prefix, reading a as padded with '0'.
		n := 0
		for n < len(b) && digitAt(a, n) == b[n] {
			n++
		}
		if n > 0 {
			return b[:n] + midpoint(tail(a, n), b[n:])
		}
	}
	da := 0
	if a != "" {
		da = digitValue(a[0])
	}
	db := base
	if b != "" {
		db = digitValue(b[0])
	}
	if db-da > 1 {
		return string(digits[(da+db+1)/2])
	}
	// The first digits are consecutive.
	if len(b) > 1 {
		return b[:1]
	}
	return string(digits[da]) + midpoint(tail(a, 1), "")
}

// integerLength returns the length of the integer part that starts with head,
// or 0 if head is not a valid head character.
func integerLength(head byte) int {
	switch {
	case head >= 'a' && head <= 'z':
		return int(head-'a') + 2
	case head >= 'A' && head <= 'Z':
		return int('Z'-head) + 2
	}
	return 0
}

// incrementInteger returns the next integer part, or false if x is the largest.
func incrementInteger(x string) (string, bool) {
	head, digs := x[0], []byte(x[1:])
	for i := len(digs) - 1; i >= 0; i-- {
		d := digitValue(digs[i]) + 1
		if d < base {
			digs[i] = digits[d]
			return string(head) + string(digs), true
		}
		digs[i] = '0'
	}
	// Carry out of the digits: move to the next head and length.
	switch head {
	case 'Z':
		return "a0", true
	case 'z':
		return "", false
	}
	head++
	if head > 'a' {
		digs = append(digs, '0')
	} else {
		digs = digs[:len(digs)-1]
	}
	return string(head) + string(digs), true
}

// decrementInteger returns the previous integer part. x must be greater than
// smallestInteger.
func decrementInteger(x string) string {
	head, digs := x[0], []byte(x[1:])
	for i := len(digs) - 1; i >= 0; i-- {
		d := digitValue(digs[i]) - 1
		if d >= 0 {
			digs[i] = digits[d]
			return string(head) + string(digs)
		}
		digs[i] = digits[base-1]
	}
	// Borrow out of the digits: move to the previous head and length.
	if head == 'a' {
		return "Z" + string(digits[base-1])
	}
	head--
	if head < 'Z' {
		digs = append(digs, digits[base-1])
	} else {
		digs = digs[:len(digs)-1]
	}
	return string(head) + string(digs)
}

// digitValue returns the value of base-62 digit c, or -1.
func digitValue(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'A' && c <= 'Z':
		return int(c-'A') + 10
	case c >= 'a' && c <= 'z':
		return int(c-'a') + 36
	}
	return -1
}

// digitAt returns s[i], or '0' past the end of s.
func digitAt(s string, i int) byte {
	if i < len(s) {
		return s[i]
	}
	return '0'
}

// tail returns s[n:], or "" if s is shorter than n.
func tail(s string, n int) string {
	if n < len(s) {
		return s[n:]
	}
	return ""
}
