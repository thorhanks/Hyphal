package ids

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/thorhanks/hyphal/internal/model"
)

// alphabet is lowercase Crockford base32: no i, l, o or u (D-003).
const alphabet = "0123456789abcdefghjkmnpqrstvwxyz"

// Length is the number of characters in an ID (CONTRACTS §2).
const Length = 12

// NewNode returns a new random node ID (§7.2, CONTRACTS §2).
func NewNode() model.NodeID {
	return model.NodeID(newRandom())
}

// NewChangeSet returns a new random change-set ID; it has the node ID format
// (CONTRACTS §2).
func NewChangeSet() model.ChangeSetID {
	return model.ChangeSetID(newRandom())
}

// NewNodeFrom returns a node ID built from the next 8 bytes of r, so tests
// can generate IDs deterministically (CONTRACTS §2).
func NewNodeFrom(r io.Reader) (model.NodeID, error) {
	var b [8]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return "", fmt.Errorf("read id bytes: %w", err)
	}
	return model.NodeID(encode(b)), nil
}

// Valid reports whether s is a well-formed ID: 12 characters of lowercase
// Crockford base32 (CONTRACTS §2). It is the only ID validator.
func Valid(s string) bool {
	if len(s) != Length {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !validChar(s[i]) {
			return false
		}
	}
	return true
}

func validChar(c byte) bool {
	switch {
	case c >= '0' && c <= '9':
		return true
	case c < 'a' || c > 'z':
		return false
	default:
		return c != 'i' && c != 'l' && c != 'o' && c != 'u'
	}
}

func newRandom() string {
	var b [8]byte
	// crypto/rand.Read never returns an error; it aborts the program if the
	// system's randomness source fails.
	_, _ = rand.Read(b[:])
	return encode(b)
}

// encode turns the low 60 bits of b (big-endian) into 12 characters.
func encode(b [8]byte) string {
	v := binary.BigEndian.Uint64(b[:])
	var out [Length]byte
	for i := Length - 1; i >= 0; i-- {
		out[i] = alphabet[v&31]
		v >>= 5
	}
	return string(out[:])
}
