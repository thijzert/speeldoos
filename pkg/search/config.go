package search

import (
	"errors"
	"regexp"
	"strings"

	"golang.org/x/text/language"
	textsearch "golang.org/x/text/search"
)

var defaultConfig Config

type Config struct {
	MinimalRelevance float64

	CaseSensitive bool
}

func init() {
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
