package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"github.com/thijzert/speeldoos"
	"log"
	"os"
	"strconv"
	"strings"
)

var (
	output_file = flag.String("output_file", "", "Output XML file")

	composer = flag.String("composer", "2222", "Preset the composer of each work")
	year     = flag.Int("year", 2222, "Preset the year of each performance")

	soloist   = flag.String("soloist", "", "Pre-fill a soloist in each performance")
	orchestra = flag.String("orchestra", "", "Pre-fill an orchestra in each performance")
	ensemble  = flag.String("ensemble", "", "Pre-fill an ensemble in each performance")
	conductor = flag.String("conductor", "", "Pre-fill a conductor in each performance")
)

func init() {
	flag.Parse()
}

func main() {
	args := flag.Args()
	if len(args) == 0 {
		croak(fmt.Errorf("Specify at least one number of parts"))
	}

	foo := &speeldoos.Carrier{}

	foo.Name = "2222"
	foo.ID = "2222"
	foo.Performances = make([]speeldoos.Performance, 0, len(args))

	track_counter := 1

	for _, i := range args {
		n, err := strconv.Atoi(i)
		croak(err)
		if n <= 0 {
			croak(fmt.Errorf("Number of parts must be positive"))
		}

		pf := speeldoos.Performance{
			Work: speeldoos.Work{
				Composer:   speeldoos.Composer{Name: *composer, ID: strings.Replace(*composer, " ", "_", -1)},
				Title:      []speeldoos.Title{speeldoos.Title{"2222", ""}},
				OpusNumber: []speeldoos.OpusNumber{speeldoos.OpusNumber{Number: "2222"}},
				Year:       2222,
			},
			Year:        *year,
			Performers:  []speeldoos.Performer{},
			SourceFiles: make([]speeldoos.SourceFile, n),
		}

		if *soloist != "" {
			pf.Performers = append(pf.Performers, speeldoos.Performer{Name: *soloist, Role: "soloist"})
		}
		if *orchestra != "" {
			pf.Performers = append(pf.Performers, speeldoos.Performer{Name: *orchestra, Role: "orchestra"})
		}
		if *ensemble != "" {
			pf.Performers = append(pf.Performers, speeldoos.Performer{Name: *ensemble, Role: "ensemble"})
		}
		if *conductor != "" {
			pf.Performers = append(pf.Performers, speeldoos.Performer{Name: *conductor, Role: "conductor"})
		}

		if len(pf.Performers) == 0 {
			pf.Performers = append(pf.Performers, speeldoos.Performer{Name: "2222", Role: "2222"})
		}

		if n > 1 {
			pf.Work.Parts = make([]string, n)
		}
		for j := 0; j < n; j++ {
			if n > 1 {
				pf.Work.Parts[j] = "2222"
			}
			pf.SourceFiles[j] = speeldoos.SourceFile{Filename: fmt.Sprintf("track_%02d.flac", track_counter)}
			track_counter++
		}

		foo.Performances = append(foo.Performances, pf)
	}

	if *output_file == "" {
		w := xml.NewEncoder(os.Stdout)
		w.Indent("", "	")
		croak(w.Encode(foo))
	} else {
		croak(foo.Write(*output_file))
	}

	fmt.Fprintf(os.Stderr, "Success. If you saved the output of this script somewhere, use your favorite\n"+
		"text editor to fill in the missing details. Pro tip: search for '2222' to\n"+
		"quickly hop between every field that's been left blank.\n")
}

func croak(e error) {
	if e != nil {
		log.Fatal(e)
	}
}
