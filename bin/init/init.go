package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"github.com/thijzert/speeldoos"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
)

var (
	output_file = flag.String("output_file", "", "Output XML file")

	track_format = flag.String("track_format", "track_%02d.flac", "Filename format of the track number")
	disc_format = flag.String("disc_format", "disc_%02d", "Directory name format of the disc number")

	composer = flag.String("composer", "2222", "Preset the composer of each work")
	year     = flag.Int("year", 2222, "Preset the year of each performance")

	soloist   = flag.String("soloist", "", "Pre-fill a soloist in each performance")
	orchestra = flag.String("orchestra", "", "Pre-fill an orchestra in each performance")
	ensemble  = flag.String("ensemble", "", "Pre-fill an ensemble in each performance")
	conductor = flag.String("conductor", "", "Pre-fill a conductor in each performance")

	discs = flag.String("discs", "", "A space separated list of the number of tracks in each disc, for a multi-disc release.")
)

func init() {
	flag.Parse()
}

func main() {
	args := flag.Args()
	if len(args) == 0 {
		croak(fmt.Errorf("Specify at least one number of parts"))
	}

	pfsize := make([]int, 0, len(args))
	total_tracks := 0

	for _, i := range args {
		n, err := strconv.Atoi(i)
		croak(err)
		if n <= 0 {
			croak(fmt.Errorf("Number of parts must be positive"))
		}
		pfsize = append(pfsize, n)
		total_tracks += n
	}

	discsize := []int{ total_tracks }
	if *discs != "" {
		discsize = discsize[0:0]
		d_total := 0
		dds := strings.Split(*discs, " ")
		for _, i := range dds {
			if i == "" {
				continue
			}
			n, err := strconv.Atoi(i)
			croak(err)
			if n <= 0 {
				croak(fmt.Errorf("Number of tracks must be positive"))
			}

			discsize = append(discsize, n)
			d_total += n
		}

		if d_total != total_tracks {
			croak(fmt.Errorf("Total tracks on all cd's (%d) does not match total number of parts (%d).", d_total, total_tracks))
		}
	}

	foo := &speeldoos.Carrier{}

	foo.Name = "2222"
	foo.ID = "2222"
	foo.Performances = make([]speeldoos.Performance, 0, len(args))

	disc_index := 0
	track_counter := 1

	for _, n := range pfsize {
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
			if len(discsize) > 1 {
				pf.SourceFiles[j] = speeldoos.SourceFile{
					Filename: path.Join(fmt.Sprintf(*disc_format, disc_index + 1), fmt.Sprintf(*track_format, track_counter)),
					Disc: disc_index + 1,
				}
			} else {
				pf.SourceFiles[j] = speeldoos.SourceFile{
					Filename: fmt.Sprintf(*track_format, track_counter),
				}
			}
			track_counter++
			if track_counter > discsize[disc_index] {
				track_counter = 1
				disc_index++
			}
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
