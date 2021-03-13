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

type resultList struct {
	Results []Result
}

// Len is the number of elements in the collection.
func (r *resultList) Len() int {
	return len(r.Results)
}

// Less reports whether the element with index i
// must sort before the element with index j.
func (r *resultList) Less(i, j int) bool {
	return r.Results[i].Relevance.Relevance() > r.Results[j].Relevance.Relevance()
}

// Swap swaps the elements with indexes i and j.
func (r *resultList) Swap(i, j int) {
	r.Results[i], r.Results[j] = r.Results[j], r.Results[i]
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

	ID        speeldoos.PerformanceID
	CarrierID MatchedString

	// The year in which the performance took place
	Year int

	Performers []performer
}
