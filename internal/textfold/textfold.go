package textfold

import (
	"slices"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/rivo/uniseg"
	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

// Folded is a string with case folded and diacritics removed, plus a map back
// to the original (§6.2).
type Folded struct {
	// S is the folded text.
	S string
	// Map[i] is the byte offset in the original of the rune that produced
	// folded byte i. It has len(S)+1 entries; the last is len(original).
	Map []int

	orig string
}

// Range is a half-open byte range [Start, End) in the original string.
type Range struct {
	Start, End int
}

// Fold NFD-normalizes s, drops nonspacing marks (unicode.Mn) and applies
// Unicode full case folding, rune by rune, so search ignores case and
// diacritics (§6.2). Folding one rune never depends on its neighbours, so the
// fold of a substring cut at rune boundaries is a substring of the fold.
func Fold(s string) Folded {
	var b strings.Builder
	b.Grow(len(s))
	m := make([]int, 0, len(s)+1)
	for i, r := range s {
		if r < utf8.RuneSelf {
			if 'A' <= r && r <= 'Z' {
				r += 'a' - 'A'
			}
			b.WriteByte(byte(r))
			m = append(m, i)
			continue
		}
		f := foldRune(r)
		b.WriteString(f)
		for range len(f) {
			m = append(m, i)
		}
	}
	m = append(m, len(s))
	return Folded{S: b.String(), Map: m, orig: s}
}

// Index returns every match of needle in haystack as ranges of the original
// string, after folding needle the same way (§6.2). Each range is widened to
// the grapheme clusters it touches, so a match inside a ligature or an
// expansion like ß → ss covers the whole character. Ranges are sorted and do
// not overlap; touching or overlapping matches are merged. An empty folded
// needle matches nothing.
func Index(haystack Folded, needle string) []Range {
	n := Fold(needle).S
	if n == "" {
		return nil
	}
	var out []Range
	var bounds []int
	for at := 0; ; {
		i := strings.Index(haystack.S[at:], n)
		if i < 0 {
			break
		}
		fs, fe := at+i, at+i+len(n)
		at = fe
		if bounds == nil {
			bounds = graphemeBounds(haystack.orig)
		}
		r := Range{
			Start: clusterStart(bounds, haystack.Map[fs]),
			End:   clusterEnd(bounds, haystack.Map[fe-1]),
		}
		if k := len(out) - 1; k >= 0 && r.Start <= out[k].End {
			out[k].End = max(out[k].End, r.End)
			continue
		}
		out = append(out, r)
	}
	return out
}

// graphemeBounds returns the byte offsets of every grapheme cluster boundary
// in s, including 0 and len(s).
func graphemeBounds(s string) []int {
	bounds := []int{0}
	state := -1
	for off := 0; off < len(s); {
		var c string
		c, _, _, state = uniseg.StepString(s[off:], state)
		off += len(c)
		bounds = append(bounds, off)
	}
	return bounds
}

// clusterStart returns the start of the cluster containing offset off.
func clusterStart(bounds []int, off int) int {
	i, found := slices.BinarySearch(bounds, off)
	if found {
		return off
	}
	return bounds[i-1]
}

// clusterEnd returns the end of the cluster containing offset off.
func clusterEnd(bounds []int, off int) int {
	i, found := slices.BinarySearch(bounds, off)
	if found {
		i++
	}
	return bounds[i]
}

var (
	// cache maps a non-ASCII rune to its folded form.
	cache sync.Map
	// caserMu guards caser, which is not safe for concurrent use.
	caserMu sync.Mutex
	caser   = cases.Fold()
)

// foldRune returns the fold of one non-ASCII rune: NFD, then nonspacing marks
// dropped, then case folding.
func foldRune(r rune) string {
	if f, ok := cache.Load(r); ok {
		return f.(string)
	}
	var kept strings.Builder
	for _, d := range norm.NFD.String(string(r)) {
		if !unicode.Is(unicode.Mn, d) {
			kept.WriteRune(d)
		}
	}
	caserMu.Lock()
	f := caser.String(kept.String())
	caserMu.Unlock()
	cache.Store(r, f)
	return f
}
