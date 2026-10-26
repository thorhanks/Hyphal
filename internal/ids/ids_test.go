package ids

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
)

var idPattern = regexp.MustCompile(`^[0-9a-hjkmnp-tv-z]{12}$`)

func TestNewNode_Format(t *testing.T) {
	for range 10_000 {
		if id := string(NewNode()); !idPattern.MatchString(id) || !Valid(id) {
			t.Fatalf("NewNode() = %q, not a valid ID", id)
		}
		if id := string(NewChangeSet()); !idPattern.MatchString(id) || !Valid(id) {
			t.Fatalf("NewChangeSet() = %q, not a valid ID", id)
		}
	}
}

func TestNewNodeFrom_Deterministic(t *testing.T) {
	src := bytes.Repeat([]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef}, 2)
	r := bytes.NewReader(src)
	a, err := NewNodeFrom(r)
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewNodeFrom(r)
	if err != nil {
		t.Fatal(err)
	}
	// 0x0123456789abcdef: the low 60 bits are 0x123456789abcdef.
	if want := "28t5cy4tqkff"; a != b || string(a) != want {
		t.Fatalf("NewNodeFrom = %q, %q; want %q twice", a, b, want)
	}
	if _, err := NewNodeFrom(bytes.NewReader([]byte{1, 2, 3})); err == nil {
		t.Fatal("NewNodeFrom on a short reader: want error")
	}
}

func TestValid(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"k3j9x2m4q7tb", true},
		{"000000000000", true},
		{"zzzzzzzzzzzz", true},
		{"", false},
		{"k3j9x2m4q7t", false},
		{"k3j9x2m4q7tbb", false},
		{"K3j9x2m4q7tb", false},
		{"k3j9x2m4q7tI", false},
		{"k3j9x2m4q7ti", false},
		{"k3j9x2m4q7tl", false},
		{"k3j9x2m4q7to", false},
		{"k3j9x2m4q7tu", false},
		{"k3j9x2m4q7t-", false},
		{"k3j9x2m4q7t ", false},
		{"k3j9x2m4q7té", false},
	}
	for _, tt := range tests {
		if got := Valid(tt.in); got != tt.want {
			t.Errorf("Valid(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
	for _, c := range alphabet {
		if !Valid(strings.Repeat(string(c), Length)) {
			t.Errorf("Valid rejects alphabet character %q", c)
		}
	}
}

func TestNewNode_NoCollisions(t *testing.T) {
	if testing.Short() {
		t.Skip("1,000,000 IDs; skipped with -short")
	}
	const n = 1_000_000
	seen := make(map[string]struct{}, n)
	for range n {
		id := string(NewNode())
		if _, dup := seen[id]; dup {
			t.Fatalf("collision on %q", id)
		}
		seen[id] = struct{}{}
	}
}
