package main

import (
	"fmt"

	tc "github.com/thijzert/go-termcolours"
	"github.com/thijzert/speeldoos/pkg/search"
)

func grep_main(args []string) {
	var q search.Query
	var qparts []search.Query
	for _, s := range args {
		p, err := search.Compile(s)
		if err != nil {
			panic(err)
		}

		qparts = append(qparts, p)
	}

	if len(qparts) == 0 {
		panic("usage: speeldoos grep PATTERN")
	} else if len(qparts) == 1 {
		q = qparts[0]
	} else {
		q = search.And(qparts[0], qparts[1:]...)
	}

	lib, err := getLibrary()
	if err != nil {
		panic(err)
	}
	results := q.Search(lib)

	mw := search.MatchWriterFunc(tc.Yellow, nil)

	for _, res := range results {
		fmt.Printf("Match: %3.1f%%, significance %g; relevance %g\n", res.Relevance.Match*100.0, res.Relevance.Significance, res.Relevance.Relevance())
		res.Work.Composer.Name.Export(mw)

		for i, t := range res.Work.Title {
			if i == 0 {
				fmt.Print("  -  ")
			} else {
				fmt.Print("  AKA ")
			}

			t.Export(mw)
			fmt.Println()
		}

		for _, perf := range res.Performances {
			fmt.Printf("%4d ", perf.Year)

			carrierMW := search.MatchWriterFunc(tc.Bcyan, tc.Cyan)
			perf.CarrierID.Export(carrierMW)

			if len(perf.Performers) > 0 {
				for i, pfm := range perf.Performers {
					if i == 0 {
						fmt.Print(" :  ")
					} else {
						fmt.Print(", ")
					}
					pfm.Name.Export(mw)
				}
			}
			fmt.Println()
		}
		for i, pp := range res.Work.Parts {
			fmt.Printf("   %3d  ", i+1)
			pp.Part.Export(mw)
			fmt.Println()
		}

		fmt.Println()
	}
}
