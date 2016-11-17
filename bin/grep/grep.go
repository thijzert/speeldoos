package main

import (
	"flag"
	"fmt"
	_ "github.com/thijzert/go-termcolours"
	"github.com/thijzert/speeldoos"
	"os"
	"path"
	"regexp"
)

var (
	case_sensitive = flag.Bool("I", false, "Perforn case-sensitive matching")
	part_context   = flag.Int("part-context", 2, "Show this much context parts around a matching part")

	speeldoos_dir = flag.String("dir", ".", "Search speeldoos files in this directory")
)

func init() {
	flag.Parse()
}

func main() {
	re := make([]*regexp.Regexp, 0)
	args := flag.Args()
	for _, a := range args {
		if !*case_sensitive {
			a = "(?i)" + a
		}
		r, er := regexp.Compile(a)
		if er != nil {
			panic(er)
		}
		re = append(re, r)
	}

	d, err := os.Open(*speeldoos_dir)
	if err != nil {
		panic(err)
	}

	files, err := d.Readdir(0)
	if err != nil {
		panic(err)
	}
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		fn := f.Name()
		if len(fn) < 5 || fn[len(fn)-4:] != ".xml" {
			continue
		}

		carrier, err := speeldoos.ImportCarrier(path.Join(*speeldoos_dir, fn))
		if err != nil {
			fmt.Printf("%s: %s\n", fn, err)
			continue
		}
		for _, pf := range carrier.Performances {
			ipp := perf{pf, carrier.ID}
			if carrier.ID == "" {
				carrier.ID = fn
			}
			if ipp.Matches(re) {
				fmt.Printf("%+v\n\n", pf)
			}
		}
	}
}

type perf struct {
	Perf      speeldoos.Performance
	CarrierID string
}

func (p perf) Matches(rre []*regexp.Regexp) bool {
	for _, r := range rre {
		if r.MatchString(p.CarrierID) {
			return true
		}
		if r.MatchString(p.Perf.Work.Composer.Name) {
			return true
		}
		for _, n := range p.Perf.Work.Title {
			if r.MatchString(n.Title) {
				return true
			}
		}
		for _, pp := range p.Perf.Work.Parts {
			if r.MatchString(pp) {
				return true
			}
		}
		for _, pf := range p.Perf.Performers {
			if r.MatchString(pf.Name) {
				return true
			}
		}
	}
	return false
}
