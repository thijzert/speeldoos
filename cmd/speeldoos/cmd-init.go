package main

import (
	"encoding/xml"
	"fmt"
	"github.com/thijzert/speeldoos"
	"os"
	"path"
	"strconv"
	"strings"
)

func init_main(args []string) {
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

	discsize := []int{total_tracks}
	if Config.Init.Discs != "" {
		discsize = discsize[0:0]
		d_total := 0
		dds := strings.Split(Config.Init.Discs, " ")
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
	foo.Source = "2222"
	foo.Performances = make([]speeldoos.Performance, 0, len(args))

	disc_index := 0
	track_counter := 1

	for _, n := range pfsize {
		pf := speeldoos.Performance{
			Work: speeldoos.Work{
				Composer:   speeldoos.Composer{Name: Config.Init.Composer, ID: strings.Replace(Config.Init.Composer, " ", "_", -1)},
				Title:      []speeldoos.Title{speeldoos.Title{"2222", ""}},
				OpusNumber: []speeldoos.OpusNumber{speeldoos.OpusNumber{Number: "2222"}},
				Year:       2222,
			},
			Year:        Config.Init.Year,
			Performers:  []speeldoos.Performer{},
			SourceFiles: make([]speeldoos.SourceFile, n),
		}

		if Config.Init.Soloist != "" {
			pf.Performers = append(pf.Performers, speeldoos.Performer{Name: Config.Init.Soloist, Role: "soloist"})
		}
		if Config.Init.Orchestra != "" {
			pf.Performers = append(pf.Performers, speeldoos.Performer{Name: Config.Init.Orchestra, Role: "orchestra"})
		}
		if Config.Init.Ensemble != "" {
			pf.Performers = append(pf.Performers, speeldoos.Performer{Name: Config.Init.Ensemble, Role: "ensemble"})
		}
		if Config.Init.Conductor != "" {
			pf.Performers = append(pf.Performers, speeldoos.Performer{Name: Config.Init.Conductor, Role: "conductor"})
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
					Filename: path.Join(fmt.Sprintf(Config.Init.DiscFormat, disc_index+1), fmt.Sprintf(Config.Init.TrackFormat, track_counter)),
					Disc:     disc_index + 1,
				}
			} else {
				pf.SourceFiles[j] = speeldoos.SourceFile{
					Filename: fmt.Sprintf(Config.Init.TrackFormat, track_counter),
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

	if Config.Init.OutputFile == "" {
		w := xml.NewEncoder(os.Stdout)
		w.Indent("", "	")
		croak(w.Encode(foo))
	} else {
		croak(foo.Write(Config.Init.OutputFile))
	}

	fmt.Fprintf(os.Stderr, "Success. If you saved the output of this script somewhere, use your favorite\n"+
		"text editor to fill in the missing details. Pro tip: search for '2222' to\n"+
		"quickly hop between every field that's been left blank.\n")
}
