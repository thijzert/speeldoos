package search

import (
	"testing"
)

func TestIntervalOverlap(t *testing.T) {
	a := interval{10, 15}
	b := interval{18, 20}
	exp := interval{10, 20}

	var c interval
	var m bool

	c, m = a.Intersect(a)
	if !m {
		t.Errorf("Expected %s to at least intersect itself, but it doesn't", a)
	} else if c != a {
		t.Errorf("Intersecting %s with itself results in %s", a, c)
	}

	c, m = a.Intersect(b)
	if m {
		t.Errorf("Expected %s and %s to not intersect, but they result in %s", a, b, c)
	}
	c, m = b.Intersect(a)
	if m {
		t.Errorf("Expected %s and %s to not intersect, but they result in %s", b, a, c)
	}

	b.A = 16
	c, m = a.Intersect(b)
	if m {
		t.Errorf("Expected %s and %s to not intersect, but they result in %s", a, b, c)
	}

	b.A = 13
	c, m = a.Intersect(b)
	if !m {
		t.Errorf("Expected %s and %s to intersect, but they don't", a, b)
	} else if c != exp {
		t.Errorf("Expected %s and %s to intersect into %s, but it results in %c instead", a, b, exp, c)
	}
	c, m = b.Intersect(a)
	if !m {
		t.Errorf("Expected %s and %s to intersect, but they don't", b, a)
	} else if c != exp {
		t.Errorf("Expected %s and %s to intersect into %s, but it results in %c instead", b, a, exp, c)
	}
	b.A = 15
	c, m = a.Intersect(b)
	if !m {
		t.Errorf("Expected %s and %s to intersect, but they don't", a, b)
	} else if c != exp {
		t.Errorf("Expected %s and %s to intersect into %s, but it results in %c instead", a, b, exp, c)
	}

	b.A = 5
	exp.A = 5
	c, m = a.Intersect(b)
	if !m {
		t.Errorf("Expected %s and %s to intersect, but they don't", a, b)
	} else if c != exp {
		t.Errorf("Expected %s and %s to intersect into %s, but it results in %c instead", a, b, exp, c)
	}
}

type expectedExport struct {
	Matched bool
	Output  string
}

type expectExporter struct {
	t            *testing.T
	contents     []expectedExport
	currentIndex int
}

func (e *expectExporter) Reset() {
	e.currentIndex = 0
}

func (e *expectExporter) AreWeThereYet() {
	if e.currentIndex != len(e.contents) {
		e.t.Errorf("Expected to have %d writes; got %d", len(e.contents), e.currentIndex)
	}
}

func (e *expectExporter) writeAnything(s string, matched bool) {
	defer func() {
		e.currentIndex++
	}()

	if e.currentIndex >= len(e.contents) {
		e.t.Errorf("Write unmatched string '%s' went past the end", s)
		return
	}
	cnt := e.contents[e.currentIndex]
	if cnt.Matched != matched {
		e.t.Errorf("Expected write %d to be matched:%v but have %v", e.currentIndex, cnt.Matched, matched)
	}
	if cnt.Output != s {
		e.t.Errorf("Expected write %d to contain '%s', but have '%s'", e.currentIndex, cnt.Output, s)
	}
}

func (e *expectExporter) WriteUnmatched(s string) {
	e.writeAnything(s, false)
}

func (e *expectExporter) WriteMatched(s string) {
	e.writeAnything(s, true)
}

func TestMatchedExport(t *testing.T) {
	ms := MatchedString{
		base: "this is a test string",
		highlights: []interval{
			{0, 4},
			{8, 14},
		},
	}

	mw := &expectExporter{
		t: t,
		contents: []expectedExport{
			{true, "this"},
			{false, " is "},
			{true, "a test"},
			{false, " string"},
		},
	}

	mw.Reset()
	ms.Export(mw)
	mw.AreWeThereYet()
}

func TestCombine(t *testing.T) {
	ms1 := MatchedString{
		base: "this is a test string",
		highlights: []interval{
			{0, 4},
			{8, 14},
		},
	}
	ms2 := MatchedString{
		base: "this is also a test string",
		highlights: []interval{
			{2, 7},
			{15, 18},
		},
	}

	_, err := ms1.Combine(ms2)
	if err == nil {
		t.Errorf("Matched strings '%s' and '%s' should not be combinable", ms1, ms2)
		return
	}

	ms2.base = "this is a test string"

	ms0, err := ms1.Combine(ms2)
	if err != nil {
		t.Errorf("Error combinging matched strings: %v", err)
	}

	mw := &expectExporter{
		t: t,
		contents: []expectedExport{
			{true, "this is"},
			{false, " "},
			{true, "a test"},
			{false, " "},
			{true, "str"},
			{false, "ing"},
		},
	}

	mw.Reset()
	ms0.Export(mw)
	mw.AreWeThereYet()
}

func TestRegexMatch(t *testing.T) {
	pm, err := defaultConfig.newRegexMatcher("d[oi]g")
	if err != nil {
		t.Error(err)
		return
	}

	ms := pm.MatchString("the quick brown fox jumps over the lazy dog")
	mw := &expectExporter{
		t: t,
		contents: []expectedExport{
			{false, "the quick brown fox jumps over the lazy "},
			{true, "dog"},
		},
	}
	mw.Reset()
	ms.Export(mw)
	mw.AreWeThereYet()

	ms = pm.MatchString("scrooge dug for gold in klondike")
	mw = &expectExporter{
		t: t,
		contents: []expectedExport{
			{false, "scrooge dug for gold in klondike"},
		},
	}
	mw.Reset()
	ms.Export(mw)
	mw.AreWeThereYet()

	ms = pm.MatchString("doug's dog digs a hole")
	mw = &expectExporter{
		t: t,
		contents: []expectedExport{
			{false, "doug's "},
			{true, "dog"},
			{false, " "},
			{true, "dig"},
			{false, "s a hole"},
		},
	}
	mw.Reset()
	ms.Export(mw)
	mw.AreWeThereYet()
}

func TestTextMatch(t *testing.T) {
	pm, ww, err := defaultConfig.newTextMatchers("dog")
	if err != nil {
		t.Error(err)
		return
	}

	ms := pm.MatchString("the quick brown fox jumps over the lazy dog")
	mw := &expectExporter{
		t: t,
		contents: []expectedExport{
			{false, "the quick brown fox jumps over the lazy "},
			{true, "dog"},
		},
	}
	mw.Reset()
	ms.Export(mw)
	mw.AreWeThereYet()

	ms = pm.MatchString("doug's dog digs a hole")
	mw = &expectExporter{
		t: t,
		contents: []expectedExport{
			{false, "doug's "},
			{true, "dog"},
			{false, " digs a hole"},
		},
	}
	mw.Reset()
	ms.Export(mw)
	mw.AreWeThereYet()

	ms = ww.MatchString("doug's dog digs a hole")
	mw.Reset()
	ms.Export(mw)
	mw.AreWeThereYet()

	ms = ww.MatchString("doug's dogs dug a hole while doug ate a hotdog")
	mw = &expectExporter{
		t: t,
		contents: []expectedExport{
			{false, "doug's dogs dug a hole while doug ate a hotdog"},
		},
	}
	mw.Reset()
	ms.Export(mw)
	mw.AreWeThereYet()
}
