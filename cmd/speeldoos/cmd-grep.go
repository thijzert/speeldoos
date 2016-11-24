package main

import (
	"fmt"
	tc "github.com/thijzert/go-termcolours"
	"github.com/thijzert/speeldoos"
	"os"
	"path"
	"regexp"
)

func grep_main(args []string) {
	re := make([]*regexp.Regexp, 0)
	for _, a := range args {
		if !Config.Grep.CaseSensitive {
			a = "(?i)" + a
		}
		r, er := regexp.Compile(a)
		if er != nil {
			panic(er)
		}
		re = append(re, r)
	}

	d, err := os.Open(Config.LibraryDir)
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

		carrier, err := speeldoos.ImportCarrier(path.Join(Config.LibraryDir, fn))
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
				fmt.Println(tc.Cyan(ipp.CarrierID))
				fmt.Printf("%s", matchedString(ipp.Perf.Work.Composer.Name, re))
				for i, t := range ipp.Perf.Work.Title {
					if i == 0 {
						fmt.Printf("  -  %s\n", matchedString(t.Title, re))
					} else {
						fmt.Printf("  AKA %s\n", matchedString(t.Title, re))
					}
				}
				if len(ipp.Perf.Performers) > 0 {
					for i, pfm := range ipp.Perf.Performers {
						if i == 0 {
							fmt.Printf("%s", matchedString(pfm.Name, re))
						} else {
							fmt.Printf(", %s", matchedString(pfm.Name, re))
						}
					}
					fmt.Printf("\n")
				}
				for i, pp := range ipp.Perf.Work.Parts {
					fmt.Printf("   %3d  %s\n", i+1, matchedString(pp, re))
				}
				fmt.Println()
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

type imatch struct {
	A, B int
}

type imatchSet struct {
	M []imatch
}

func newMatchSet() *imatchSet {
	rv := &imatchSet{M: make([]imatch, 0, 10)}
	return rv
}

func (s *imatchSet) Add(a, b int) {
	if b < a {
		// Sod this.
		a, b = b, a
	}

	if len(s.M) == 0 {
		s.M = append(s.M, imatch{a, b})
		return
	}
	if s.M[0].A > b+1 {
		s.M = append(s.M, imatch{})
		copy(s.M[1:], s.M[0:len(s.M)-1])
		s.M[0].A = a
		s.M[0].B = b
		return
	}

	for i, m := range s.M {
		if m.A < a && m.B < a-1 {
			continue
		}
		if m.A > b+1 {
			s.M = append(s.M, imatch{})
			copy(s.M[i+1:], s.M[i:len(s.M)-1])
			s.M[i].A = a
			s.M[i].B = b
		} else {
			if a < m.A {
				s.M[i].A = a
			}
			if b > m.B {
				s.M[i].B = b
			}
			for i < len(s.M)-1 && s.M[i+1].A <= s.M[i].B+1 {
				if s.M[i].B < s.M[i+1].B {
					s.M[i].B = s.M[i+1].B
				}

				copy(s.M[i+1:], s.M[i+2:])
				s.M = s.M[:len(s.M)-1]
			}
		}

		return
	}

	s.M = append(s.M, imatch{a, b})
}

func (s *imatchSet) String() string {
	rv := "{"
	for i, m := range s.M {
		if i > 0 {
			rv += " "
		}
		rv += fmt.Sprintf("[%d %d]", m.A, m.B)
	}
	return rv + "}"
}

func matchedString(input string, regexes []*regexp.Regexp) (rv string) {
	ms := newMatchSet()
	for _, r := range regexes {
		mm := r.FindAllStringIndex(input, -1)
		for _, m := range mm {
			ms.Add(m[0], m[1])
		}
	}

	rv = ""
	last := 0
	for _, s := range ms.M {
		rv += input[last:s.A]
		rv += tc.Yellow(input[s.A:s.B])
		last = s.B
	}
	rv += input[last:]
	return
}
