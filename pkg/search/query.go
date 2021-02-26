package search

import (
	"errors"
	"regexp"
	"strings"

	speeldoos "github.com/thijzert/speeldoos/pkg"
	"golang.org/x/text/language"
	textsearch "golang.org/x/text/search"
)

var defaultConfig Config

type Config struct {
	MinimalRelevance float64

	CaseSensitive bool
}

var containsRegexChars *regexp.Regexp

func init() {
	containsRegexChars = regexp.MustCompile("[\\(\\)\\{\\}\\\\\\[\\]\\|\\.]")

	defaultConfig.MinimalRelevance = 0.5
}

func (c Config) Compile(q string) (Query, error) {
	// TODO: tokenize in a more clever way
	tokens := strings.Split(q, " ")

	matcher := andNode{}

	for _, queryPart := range tokens {
		queryPart = strings.TrimSpace(queryPart)
		if queryPart == "" {
			continue
		}

		textm, err := c.newTextMatcher(queryPart)
		if err != nil {
			return Query{}, err
		}

		var qpm resulterer = matcherNode{textm}

		if containsRegexChars.MatchString(queryPart) {
			rexm, err := c.newRegexMatcher(queryPart)
			if err == nil {
				qpm = orNode{
					Parts: []resulterer{
						qpm,
						matcherNode{rexm},
					},
				}
			}
		}

		matcher.Parts = append(matcher.Parts, qpm)
	}

	if len(matcher.Parts) == 0 {
		return Query{}, errors.New("empty query")
	}

	return Query{
		MinimalRelevance: c.MinimalRelevance,
		rootMatcher:      matcher,
	}, nil
}

func (c Config) newTextMatcher(s string) (textMatcher, error) {
	var rv textMatcher

	// Just combine all languages. What could possibly go wrong?
	tag, err := language.Compose(
		language.English,
		language.Afrikaans,
		language.Amharic,
		language.Arabic,
		language.Azerbaijani,
		language.Bulgarian,
		language.Bengali,
		language.Catalan,
		language.Czech,
		language.Danish,
		language.German,
		language.Greek,
		language.Spanish,
		language.Estonian,
		language.Persian,
		language.Finnish,
		language.Filipino,
		language.French,
		language.Gujarati,
		language.Hebrew,
		language.Hindi,
		language.Croatian,
		language.Hungarian,
		language.Armenian,
		language.Indonesian,
		language.Icelandic,
		language.Italian,
		language.Japanese,
		language.Georgian,
		language.Kazakh,
		language.Khmer,
		language.Kannada,
		language.Korean,
		language.Kirghiz,
		language.Lao,
		language.Lithuanian,
		language.Latvian,
		language.Macedonian,
		language.Malayalam,
		language.Mongolian,
		language.Marathi,
		language.Malay,
		language.Burmese,
		language.Nepali,
		language.Dutch,
		language.Norwegian,
		language.Punjabi,
		language.Polish,
		language.Portuguese,
		language.Romanian,
		language.Russian,
		language.Sinhala,
		language.Slovak,
		language.Slovenian,
		language.Albanian,
		language.Serbian,
		language.SerbianLatin,
		language.Swedish,
		language.Swahili,
		language.Tamil,
		language.Telugu,
		language.Thai,
		language.Turkish,
		language.Ukrainian,
		language.Urdu,
		language.Uzbek,
		language.Vietnamese,
		language.Chinese,
		language.Zulu,
	)
	if err != nil {
		return rv, err
	}

	opts := []textsearch.Option{
		textsearch.IgnoreWidth,
		textsearch.IgnoreDiacritics,
	}

	if !c.CaseSensitive {
		opts = append(opts, textsearch.IgnoreCase)
	}

	matcher := textsearch.New(tag, opts...)

	pat := matcher.CompileString(s)

	rv.Pattern = pat

	return rv, nil
}

func (c Config) newRegexMatcher(s string) (regexMatcher, error) {
	var rv regexMatcher

	if !c.CaseSensitive {
		s = "(?i)" + s
	}
	r, er := regexp.Compile(s)
	if er != nil {
		return rv, er
	}

	rv.Regex = r
	return rv, nil
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
	var rv []Result

	carriers := lib.AllCarriers()
	for _, carrier := range carriers {
		for _, perf := range carrier.Carrier.Performances {
			res := q.rootMatcher.GetResult(perf)

			if res.Relevance.Relevance() >= q.MinimalRelevance {
				rv = append(rv, res)
			}
		}
	}

	return rv
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

	rv.CarrierID = MatchedString{base: "FIXME"} // Carrier ID needs to be passed in with the speeldoos.Performance somehow
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

	// FIXME: do we even need these?
	rv.SourceFiles = append(rv.SourceFiles, p.SourceFiles...)
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

		bperf := b.Performances[j]

		mperf := performance{
			CarrierID:   perf.CarrierID.mustCombine(bperf.CarrierID),
			Year:        perf.Year,
			SourceFiles: perf.SourceFiles,
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
			total.Match = pr.Relevance.Match
			total.Significance = pr.Relevance.Relevance()
		} else {
			rv = mergeResults(rv, pr)
			total.Match += pr.Relevance.Match
			total.Significance += pr.Relevance.Relevance()
		}
	}

	// if total.Match > 0 {
	// 	total.Significance /= total.Match
	// }
	if len(n.Parts) > 0 {
		total.Match /= float64(len(n.Parts))
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

	//if total.Match > 0 {
	//	total.Significance /= total.Match
	//}
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
