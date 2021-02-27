package search

import (
	"errors"
	"fmt"
	"regexp"
	"unicode"
	"unicode/utf8"

	textsearch "golang.org/x/text/search"
)

type StringMatcher interface {
	MatchString(string) MatchedString
}

type textMatcher struct {
	Pattern *textsearch.Pattern
}

func (n textMatcher) MatchString(s string) MatchedString {
	rv := MatchedString{
		base: s,
	}

	offset := 0
	for offset < len(s) {
		start, end := n.Pattern.IndexString(s[offset:], 0)
		if start >= 0 {
			rv.highlights = append(rv.highlights, interval{
				start + offset,
				end + offset,
			})

			offset += start + 1
		} else {
			break
		}
	}

	return rv
}

type wordMatcher struct {
	Pattern *textsearch.Pattern
}

func (n wordMatcher) MatchString(s string) MatchedString {
	rv := MatchedString{
		base: s,
	}

	offset := 0
	for offset < len(s) {
		start, end := n.Pattern.IndexString(s[offset:], 0)
		if start >= 0 {
			prev, _ := utf8.DecodeLastRuneInString(s[:offset+start])
			next, _ := utf8.DecodeRuneInString(s[offset+end:])

			if n.notWord(prev) && n.notWord(next) {
				rv.highlights = append(rv.highlights, interval{
					start + offset,
					end + offset,
				})
			}

			offset += start + 1
		} else {
			break
		}
	}

	return rv
}

func (n wordMatcher) notWord(r rune) bool {
	return r == utf8.RuneError || unicode.IsSpace(r) || unicode.IsSymbol(r) || unicode.IsPunct(r)
}

type regexMatcher struct {
	Regex *regexp.Regexp
}

func (n regexMatcher) MatchString(s string) MatchedString {
	rv := MatchedString{
		base: s,
	}

	if n.Regex != nil {
		indices := n.Regex.FindAllStringIndex(s, -1)
		for _, intv := range indices {
			rv.highlights = append(rv.highlights, interval{intv[0], intv[1]})
		}
	}

	return rv
}

type MatchWriter interface {
	WriteMatched(string)
	WriteUnmatched(string)
}

func MatchWriterFunc(matched, unmatched func(string) string) MatchWriter {
	return filterMatchWriter{
		Fmatched:   matched,
		Funmatched: unmatched,
	}
}

type filterMatchWriter struct {
	Fmatched   func(string) string
	Funmatched func(string) string
}

func (f filterMatchWriter) WriteMatched(s string) {
	if f.Fmatched != nil {
		s = f.Fmatched(s)
	}
	fmt.Print(s)
}

func (f filterMatchWriter) WriteUnmatched(s string) {
	if f.Funmatched != nil {
		s = f.Funmatched(s)
	}
	fmt.Print(s)
}

type interval struct {
	A, B int
}

func (a interval) String() string {
	return fmt.Sprintf("[%d %d)", a.A, a.B)
}

// Intersect checks if two intervals overlap or are directly adjacent, and,
// confusingly, returns the union if they do.
func (a interval) Intersect(b interval) (rv interval, overlaps bool) {
	rv.A = a.A
	rv.B = a.B
	overlaps = true

	if b.A >= a.A && b.A <= a.B {
		if b.B > a.B {
			rv.B = b.B
		}
	} else if b.B >= (a.A-1) && b.B <= a.B {
		if b.A < a.A {
			rv.A = b.A
		}
	} else if a.B >= b.A && a.B < b.B {
		// B completely contains A
		return b.Intersect(a)
	} else {
		overlaps = false
	}

	return
}

type MatchedString struct {
	base       string
	highlights []interval
}

func (ms MatchedString) String() string {
	return ms.base
}

func (ms MatchedString) IsEmpty() bool {
	return len(ms.highlights) == 0
}

func (ms MatchedString) Export(mw MatchWriter) {
	offset := 0

	for _, intv := range ms.highlights {
		if offset < intv.A {
			mw.WriteUnmatched(ms.base[offset:intv.A])
		}
		if intv.A < intv.B {
			mw.WriteMatched(ms.base[intv.A:intv.B])
		}
		offset = intv.B
	}

	if offset < len(ms.base) {
		mw.WriteUnmatched(ms.base[offset:])
	}
}

func (ms MatchedString) Combine(b MatchedString) (MatchedString, error) {
	rv := MatchedString{
		base: ms.base,
	}

	if ms.base != b.base {
		return rv, errors.New("matched strings are of different base strings")
	}

	type vInterval struct {
		Interval interval
		Visited  bool
	}

	// Step 1: merge both sets of highlights into one list, ordered by their start coordinates
	segments := make([]vInterval, 0, len(ms.highlights)+len(ms.highlights))

	offsetA := 0
	offsetB := 0
	for offsetA < len(ms.highlights) || offsetB < len(b.highlights) {
		var seg vInterval

		if offsetA < len(ms.highlights) && offsetB < len(b.highlights) {
			if ms.highlights[offsetA].A < b.highlights[offsetB].A {
				seg.Interval = ms.highlights[offsetA]
				offsetA++
			} else {
				seg.Interval = b.highlights[offsetB]
				offsetB++
			}
		} else if offsetA < len(ms.highlights) {
			seg.Interval = ms.highlights[offsetA]
			offsetA++
		} else {
			seg.Interval = b.highlights[offsetB]
			offsetB++
		}

		segments = append(segments, seg)
	}

	// Step 2: compress the list down as much as possible
	for i, seg := range segments {
		if seg.Visited {
			continue
		}
		segments[i].Visited = true

		for j, bseg := range segments[i+1:] {
			if bseg.Visited {
				continue
			}

			cseg, intersect := seg.Interval.Intersect(bseg.Interval)
			if intersect {
				segments[i+j+1].Visited = true
				seg.Interval = cseg
			}
		}

		rv.highlights = append(rv.highlights, seg.Interval)
	}

	return rv, nil
}

func (ms MatchedString) mustCombine(b MatchedString) MatchedString {
	if comb, err := ms.Combine(b); err == nil {
		return comb
	}
	return ms
}
