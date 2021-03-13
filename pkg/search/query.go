package search

import (
	"math"
	"regexp"
	"sort"

	speeldoos "github.com/thijzert/speeldoos/pkg"
)

var containsRegexChars *regexp.Regexp

func init() {
	containsRegexChars = regexp.MustCompile("[\\(\\)\\{\\}\\\\\\[\\]\\|\\.]")
}

type resulterer interface {
	GetResult(perf speeldoos.Performance) Result
}

type Query struct {
	MinimalRelevance float64
	rootMatcher      resulterer
}

func Compile(q string) (Query, error) {
	return defaultConfig.Compile(q)
}

func (q Query) Search(lib *speeldoos.Library) []Result {
	var rv resultList

	carriers := lib.AllCarriers()
	for _, carrier := range carriers {
		for _, perf := range carrier.Carrier.Performances {
			res := q.rootMatcher.GetResult(perf)

			if res.Relevance.Match > 0 && res.Relevance.Relevance() >= q.MinimalRelevance {
				rv.Results = append(rv.Results, res)
			}
		}
	}

	sort.Sort(&rv)

	return rv.Results
}

type matcherNode struct {
	f StringMatcher
}

func (n matcherNode) GetResult(perf speeldoos.Performance) Result {
	var rv Result

	rv.Work = n.getWork(perf.Work)
	rv.Performances = append(rv.Performances, n.getPerformance(perf))

	// Find some combined relevance from the work's and performances'. The match
	// should be the highest we find, and the effective relevance should be the
	// sum of the work's relevance plus all the performances' relevance.

	totalRelevance := rv.Work.Relevance.Relevance()
	rv.Relevance.Match = rv.Work.Relevance.Match

	for _, pf := range rv.Performances {
		if rv.Relevance.Match < pf.Relevance.Match {
			rv.Relevance.Match = pf.Relevance.Match
		}
		totalRelevance += pf.Relevance.Relevance()
	}

	if rv.Relevance.Match > 0 {
		rv.Relevance.Significance = totalRelevance / rv.Relevance.Match
	}

	return rv
}

func (n matcherNode) getWork(w speeldoos.Work) work {
	var rv work
	rv.Relevance.Significance = 1.0 // TODO

	rv.Composer.ID = w.Composer.ID
	rv.Composer.Name = n.f.MatchString(w.Composer.Name)
	if !rv.Composer.Name.IsEmpty() {
		rv.Relevance.Match = 1.0 // TODO
	}

	for _, title := range w.Title {
		mt := n.f.MatchString(title.Title)
		if !mt.IsEmpty() {
			rv.Relevance.Match = 1.0 // TODO
		}
		rv.Title = append(rv.Title, mt)
	}

	for _, op := range w.OpusNumber {
		mop := n.f.MatchString(op.String())
		if !mop.IsEmpty() {
			rv.Relevance.Match = 1.0 // TODO
		}
		rv.OpusNumber = append(rv.OpusNumber, mop)
	}

	for _, pt := range w.Parts {
		mpt := part{
			Number: pt.Number,
			Part:   n.f.MatchString(pt.Part),
		}
		if !mpt.Part.IsEmpty() {
			rv.Relevance.Match = 1.0 // TODO
		}
		rv.Parts = append(rv.Parts, mpt)
	}

	rv.Year = w.Year

	return rv
}

func (n matcherNode) getPerformance(p speeldoos.Performance) performance {
	var rv performance
	rv.Relevance.Significance = 1.0 // TODO

	rv.ID = p.ID
	rv.CarrierID = n.f.MatchString(p.ID.Carrier())
	rv.Year = p.Year

	for _, pf := range p.Performers {
		mpf := performer{
			Name: n.f.MatchString(pf.Name),
			Role: pf.Role,
		}
		if !mpf.Name.IsEmpty() {
			rv.Relevance.Match = 1.0 // TODO
		}
		rv.Performers = append(rv.Performers, mpf)
	}

	return rv
}

type andNode struct {
	Parts []resulterer
}

func mergeResults(a, b Result) Result {
	c := Result{
		Relevance: a.Relevance,
		Work: work{
			Composer: a.Work.Composer,
			Year:     a.Work.Year,
		},
	}

	c.Work.Composer.Name = a.Work.Composer.Name.mustCombine(b.Work.Composer.Name)

	for i, title := range a.Work.Title {
		if i < len(b.Work.Title) {
			title = title.mustCombine(b.Work.Title[i])
		}
		c.Work.Title = append(c.Work.Title, title)
	}

	for i, opus := range a.Work.OpusNumber {
		if i < len(b.Work.OpusNumber) {
			opus = opus.mustCombine(b.Work.OpusNumber[i])
		}
		c.Work.OpusNumber = append(c.Work.OpusNumber, opus)
	}

	for i, part := range a.Work.Parts {
		if i < len(b.Work.Parts) {
			part.Part = part.Part.mustCombine(b.Work.Parts[i].Part)
		}
		c.Work.Parts = append(c.Work.Parts, part)
	}

	for j, perf := range a.Performances {
		if j >= len(b.Performances) {
			c.Performances = append(c.Performances, perf)
			continue
		}

		// TODO: compare performance IDs to handle the case when they're in a different order, or don't have the same performances
		bperf := b.Performances[j]

		mperf := performance{
			ID:        perf.ID,
			CarrierID: perf.CarrierID.mustCombine(bperf.CarrierID),
			Year:      perf.Year,
		}

		for i, prfm := range perf.Performers {
			if i < len(bperf.Performers) {
				prfm.Name = prfm.Name.mustCombine(bperf.Performers[i].Name)
			}
			mperf.Performers = append(mperf.Performers, prfm)
		}

		c.Performances = append(c.Performances, mperf)
	}

	return c
}

func (n andNode) GetResult(perf speeldoos.Performance) Result {
	var rv Result

	var total Relevance

	for i, part := range n.Parts {
		pr := part.GetResult(perf)
		if i == 0 {
			rv = pr
			total.Match = pr.Relevance.Match * pr.Relevance.Match
			total.Significance = pr.Relevance.Relevance()
		} else {
			rv = mergeResults(rv, pr)
			total.Match += pr.Relevance.Match * pr.Relevance.Match
			total.Significance += pr.Relevance.Relevance()
		}
	}

	if len(n.Parts) > 0 {
		total.Match /= float64(len(n.Parts))
	}
	if total.Match > 0 {
		total.Match = math.Sqrt(total.Match)
		total.Significance /= math.Sqrt(total.Match)
	}

	rv.Relevance = total
	return rv
}

func And(a Query, bs ...Query) Query {
	rm := andNode{
		Parts: []resulterer{a.rootMatcher},
	}
	for _, b := range bs {
		rm.Parts = append(rm.Parts, b.rootMatcher)
	}

	return Query{
		MinimalRelevance: a.MinimalRelevance,
		rootMatcher:      rm,
	}
}

type orNode struct {
	Parts []resulterer
}

func (n orNode) GetResult(perf speeldoos.Performance) Result {
	var rv Result

	var total Relevance

	for i, part := range n.Parts {
		pr := part.GetResult(perf)
		if i == 0 {
			rv = pr
			total.Match = pr.Relevance.Match
			total.Significance = pr.Relevance.Relevance()
		} else {
			rv = mergeResults(rv, pr)
			if pr.Relevance.Match > total.Match {
				total.Match = pr.Relevance.Match
			}
			total.Significance += pr.Relevance.Relevance()
		}
	}

	if total.Match > 0 {
		total.Significance /= total.Match
	}
	rv.Relevance = total

	return rv
}

func Or(a Query, bs ...Query) Query {
	rm := orNode{
		Parts: []resulterer{a.rootMatcher},
	}
	for _, b := range bs {
		rm.Parts = append(rm.Parts, b.rootMatcher)
	}

	return Query{
		MinimalRelevance: a.MinimalRelevance,
		rootMatcher:      rm,
	}
}
