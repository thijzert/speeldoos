package search

import (
	speeldoos "github.com/thijzert/speeldoos/pkg"
)

type Relevance struct {
	Match        float64
	Significance float64
}

func (r Relevance) Relevance() float64 {
	return r.Match * r.Significance
}

type Result struct {
	Relevance Relevance

	Work         work
	Performances []performance
}

type composer struct {
	Name MatchedString
	ID   string
}

// A Part of a work
type part struct {
	Part   MatchedString
	Number string
}

type work struct {
	Relevance Relevance

	Composer composer

	// A work may have more than one title in different languages.
	Title []MatchedString

	// The opus number(s) for this composition. There may be more than one index
	// for identifying works by this particular composer. Or none at all. Or
	// this work may just not appear on any of them.
	OpusNumber []MatchedString

	// The parts that comprise this work, if any.
	Parts []part

	// The year this composition was completed.
	Year int
}

// A Performer in one particular recording of a Work.
// An 'artist', if you will.
type performer struct {
	Name MatchedString
	Role string
}

// A Performance represents one (recording of a) performance of a Work.
type performance struct {
	Relevance Relevance

	CarrierID MatchedString

	// The year in which the performance took place
	Year int

	Performers  []performer
	SourceFiles []speeldoos.SourceFile
}
