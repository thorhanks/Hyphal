package textfold

import (
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/rivo/uniseg"
)

func TestIndex_Diacritics(t *testing.T) {
	const s = "Crème Brûlée"
	got := Index(Fold(s), "creme brulee")
	want := []Range{{0, len(s)}}
	if !slices.Equal(got, want) {
		t.Fatalf("Index(Fold(%q), %q) = %v, want %v", s, "creme brulee", got, want)
	}
}

func TestIndex_DiacriticsInNeedle(t *testing.T) {
	const s = "creme brulee"
	got := Index(Fold(s), "BRÛLÉE")
	want := []Range{{6, len(s)}}
	if !slices.Equal(got, want) {
		t.Fatalf("Index = %v, want %v", got, want)
	}
}

func TestIndex_DecomposedOriginal(t *testing.T) {
	// "é" written as e + U+0301: the match must include the combining mark.
	const s = "café au lait"
	got := Index(Fold(s), "café")
	want := []Range{{0, len("café")}}
	if !slices.Equal(got, want) {
		t.Fatalf("Index = %v, want %v", got, want)
	}
}

func TestIndex_SharpS(t *testing.T) {
	cases := []struct {
		haystack, needle string
		want             []Range
	}{
		{"straße", "STRASSE", []Range{{0, len("straße")}}},
		{"STRASSE", "straße", []Range{{0, len("STRASSE")}}},
		{"Hauptstraße 5", "strasse", []Range{{5, 5 + len("straße")}}},
		// Half of an expansion maps to the whole ß.
		{"straße", "stras", []Range{{0, len("straß")}}},
		{"aßb", "sb", []Range{{1, len("aßb")}}},
	}
	for _, c := range cases {
		if got := Index(Fold(c.haystack), c.needle); !slices.Equal(got, c.want) {
			t.Errorf("Index(Fold(%q), %q) = %v, want %v", c.haystack, c.needle, got, c.want)
		}
	}
}

func TestIndex_AllMatchesMerged(t *testing.T) {
	cases := []struct {
		haystack, needle string
		want             []Range
	}{
		{"Ana and ANA", "ana", []Range{{0, 3}, {8, 11}}},
		{"aaaa", "aa", []Range{{0, 4}}},
		{"ßß", "s", []Range{{0, len("ßß")}}},
		{"abc", "x", nil},
		{"abc", "", nil},
		{"abc", "́", nil}, // folds to nothing
	}
	for _, c := range cases {
		if got := Index(Fold(c.haystack), c.needle); !slices.Equal(got, c.want) {
			t.Errorf("Index(Fold(%q), %q) = %v, want %v", c.haystack, c.needle, got, c.want)
		}
	}
}

func TestIndex_GraphemeClusters(t *testing.T) {
	// The family emoji is one cluster; matching the woman emoji inside it
	// highlights the whole cluster.
	const family = "\U0001F468\u200D\U0001F469\u200D\U0001F467"
	s := "a" + family + "b"
	got := Index(Fold(s), "\U0001F469")
	want := []Range{{1, 1 + len(family)}}
	if !slices.Equal(got, want) {
		t.Fatalf("Index = %v, want %v", got, want)
	}
}

// TestFold_Unicode documents behavior that follows cases.Fold (full folding,
// no language tailoring) after NFD and mark removal.
func TestFold_Unicode(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"ascii", "Hello, World 42", "hello, world 42"},
		{"latin diacritics", "Crème Brûlée", "creme brulee"},
		{"sharp s", "Straße", "strasse"},
		{"capital sharp s", "STRAẞE", "strasse"},
		{"turkish dotted capital I", "İstanbul", "istanbul"},
		{"turkish dotless i stays distinct", "ılık", "ılık"},
		{"plain capital I", "IŞIK", "isik"},
		{"greek sigma forms", "ΣΊΣΥΦΟΣ", "σισυφοσ"},
		{"greek final sigma", "σίσυφος", "σισυφοσ"},
		{"greek tonos removed", "Άλφα", "αλφα"},
		{"cyrillic", "ЁЛКА", "елка"},
		{"cjk unchanged", "東京タワー", "東京タワー"},
		{"fullwidth latin", "ＡＢＣ", "ａｂｃ"},
		{"ligature", "ﬁle", "file"},
		{"vietnamese stacked marks", "Tiếng Việt", "tieng viet"},
		{"invalid utf-8", "a\xffb", "a�b"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Fold(c.in).S; got != c.want {
				t.Fatalf("Fold(%q).S = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestFold_Map(t *testing.T) {
	const s = "Aé ß"
	f := Fold(s)
	if f.S != "ae ss" {
		t.Fatalf("Fold(%q).S = %q", s, f.S)
	}
	want := []int{0, 1, 3, 4, 4, len(s)}
	if !slices.Equal(f.Map, want) {
		t.Fatalf("Fold(%q).Map = %v, want %v", s, f.Map, want)
	}
}

func FuzzIndex(f *testing.F) {
	for _, s := range [][2]string{
		{"Crème Brûlée", "creme"},
		{"straße", "ss"},
		{"café", "e"},
		{"\U0001F468\u200D\U0001F469", "\U0001F469"},
		{"İstanbul ΣΊΣΥΦΟΣ", "ς"},
		{"a\xffb\xc3", "\xff"},
		{"🇩🇪🇫🇷", "🇪🇫"},
	} {
		f.Add(s[0], s[1])
	}
	f.Fuzz(func(t *testing.T, haystack, needle string) {
		h := Fold(haystack)
		if len(h.Map) != len(h.S)+1 || h.Map[len(h.S)] != len(haystack) {
			t.Fatalf("Map has %d entries for %d folded bytes", len(h.Map), len(h.S))
		}
		n := Fold(needle).S
		got := Index(h, needle)
		if (len(got) > 0) != (n != "" && strings.Contains(h.S, n)) {
			t.Fatalf("Index found %v, but folded containment is %v", got, strings.Contains(h.S, n))
		}
		bounds := graphemeBoundsRef(haystack)
		prevEnd := -1
		for _, r := range got {
			if r.Start < 0 || r.End > len(haystack) || r.Start >= r.End {
				t.Fatalf("invalid range %v in %q", r, haystack)
			}
			if r.Start <= prevEnd {
				t.Fatalf("ranges overlap or touch: %v", got)
			}
			prevEnd = r.End
			if !bounds[r.Start] || !bounds[r.End] {
				t.Fatalf("range %v is not at grapheme boundaries of %q", r, haystack)
			}
			if !strings.Contains(Fold(haystack[r.Start:r.End]).S, n) {
				t.Fatalf("range %v (%q) does not contain folded needle %q", r, haystack[r.Start:r.End], n)
			}
		}
	})
}

// graphemeBoundsRef computes the set of grapheme boundaries independently of
// the package's helper.
func graphemeBoundsRef(s string) map[int]bool {
	b := map[int]bool{0: true}
	g := uniseg.NewGraphemes(s)
	for g.Next() {
		_, end := g.Positions()
		b[end] = true
	}
	return b
}

// BenchmarkFold100k folds 100,000 strings of 40 runes per iteration (AC5:
// < 50 ms on CI hardware).
func BenchmarkFold100k(b *testing.B) {
	words := []string{
		"the", "Quick", "brown", "fox", "jumps", "over", "lazy", "dog",
		"Crème", "brûlée", "straße", "naïve", "Café", "İstanbul", "Σίσυφος",
		"東京", "Ёлка", "meeting", "notes", "TODO", "project", "review",
	}
	rng := rand.New(rand.NewPCG(1, 2))
	strs := make([]string, 100_000)
	for i := range strs {
		var sb strings.Builder
		for n := 0; n < 40; {
			if n > 0 {
				sb.WriteByte(' ')
				n++
			}
			for _, r := range words[rng.IntN(len(words))] {
				if n == 40 {
					break
				}
				sb.WriteRune(r)
				n++
			}
		}
		strs[i] = sb.String()
	}
	for b.Loop() {
		for _, s := range strs {
			_ = Fold(s)
		}
	}
}
