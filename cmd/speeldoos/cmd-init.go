package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"

	speeldoos "github.com/thijzert/speeldoos/pkg"
)

var defaultIndexNames = map[string]string{
	"Carl Philipp Emanuel Bach": "Wq",
	"Dieterich Buxtehude":       "BuxWV",
	"Franz Schubert":            "D",
	"Georg Philipp Telemann":    "TWV",
	"Johann Sebastian Bach":     "BWV",
	"Wolfgang Amadeus Mozart":   "K",
}

type detectedDisc struct {
	path  string
	files []detectedFile
}

type detectedFile struct {
	path  string
	base  string
	track int
}

var number *regexp.Regexp = regexp.MustCompile("\\d+")

func detectFiles(dirname, ext string) []detectedFile {
	rv := make([]detectedFile, 0, 10)
	dir, err := os.Open(dirname)
	if err != nil {
		return rv
	}

	names, err := dir.Readdirnames(-1)
	if err != nil {
		return rv
	}

	sort.Strings(names)

	for _, name := range names {
		if name == "" || name[0:1] == "." || len(name) <= len(ext) {
			continue
		}
		if name[len(name)-len(ext):] != ext {
			continue
		}

		fi, err := os.Stat(path.Join(dirname, name))
		if err != nil {
			continue
		}
		if fi.IsDir() || fi.Size() < 4 {
			continue
		}

		track := 0
		digmatch := number.FindStringIndex(name)
		if digmatch != nil {
			fmt.Sscan(name[digmatch[0]:digmatch[1]], &track)
		}

		rv = append(rv, detectedFile{
			path:  path.Join(dirname, name),
			base:  name,
			track: track,
		})
	}

	return rv
}

func detectAllFiles(dirname, ext string) map[int]detectedDisc {
	rv := make(map[int]detectedDisc)

	lastDirWithFiles := ""
	dsf := detectFiles(dirname, ext)
	if len(dsf) > 0 {
		rv[0] = detectedDisc{
			path:  ".",
			files: dsf,
		}
	}

	dir, err := os.Open(dirname)
	if err != nil {
		return rv
	}

	fii, err := dir.Readdir(-1)
	if err != nil {
		return rv
	}

	for _, fi := range fii {
		if !fi.IsDir() {
			continue
		}

		name := fi.Name()
		if name == "" || name[0:1] == "." {
			continue
		}

		fullpath := path.Join(dirname, name)
		ssf := detectFiles(fullpath, ext)

		disc := 0
		digmatch := number.FindStringIndex(name)
		if digmatch != nil {
			fmt.Sscan(name[digmatch[0]:digmatch[1]], &disc)
		}

		if disc < 1 || disc > 100 {
			if len(ssf) > 0 {
				dsf = ssf
				lastDirWithFiles = fullpath
			}
			continue
		}
		if _, ok := rv[disc]; ok {
			continue
		}

		if len(ssf) > 0 {
			rv[disc] = detectedDisc{
				path:  fullpath,
				files: ssf,
			}
		}
	}

	if _, ok := rv[0]; !ok {
		if len(dsf) > 0 {
			rv[0] = detectedDisc{
				path:  lastDirWithFiles,
				files: dsf,
			}
		}
	}

	return rv
}

func init_main(args []string) {
	if len(args) == 0 {
		croak(fmt.Errorf("Specify at least one number of parts"))
	}

	detectedSourceFiles := detectAllFiles(".", ".flac")

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

	if len(discsize) > 1 {
		if len(detectedSourceFiles) > 0 {
			if len(detectedSourceFiles) != len(discsize) {
				croak(fmt.Errorf("Have %d discs, but detected %d sets of source files.", len(discsize), len(detectedSourceFiles)))
			}

			for i, size := range discsize {
				if dd, ok := detectedSourceFiles[i+1]; ok {
					if len(dd.files) != size {
						croak(fmt.Errorf("Disc %d: expected %d tracks, but detected %d source files.", i+1, size, len(dd.files)))
					}
				} else {
					croak(fmt.Errorf("Source files for disc %d not autodetected", i+1))
				}
			}
		}
	} else {
		if len(detectedSourceFiles) > 1 {
			croak(fmt.Errorf("Have 1 disc, but detected %d sets of source files.", len(detectedSourceFiles)))
		}

		if dd, ok := detectedSourceFiles[1]; ok {
			detectedSourceFiles[0] = dd
			delete(detectedSourceFiles, 1)
		}
		if dd, ok := detectedSourceFiles[0]; ok {
			if len(dd.files) != total_tracks {
				croak(fmt.Errorf("Have %d parts, but detected %d source files.", total_tracks, len(dd.files)))
			}
		}
	}

	foo := &speeldoos.Carrier{}

	foo.Name = "2222"
	foo.ID = "2222"
	foo.Source = "2222"
	foo.Performances = make([]speeldoos.Performance, 0, len(args))

	indexName := defaultIndexNames[Config.Init.Composer]

	disc_index := 0
	track_counter := 1

	for _, n := range pfsize {
		pf := speeldoos.Performance{
			Work: speeldoos.Work{
				Composer:   speeldoos.Composer{Name: Config.Init.Composer, ID: strings.Replace(Config.Init.Composer, " ", "_", -1)},
				Title:      []speeldoos.Title{speeldoos.Title{"2222", ""}},
				OpusNumber: []speeldoos.OpusNumber{speeldoos.OpusNumber{IndexName: indexName, Number: "2222"}},
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
			pf.Work.Parts = make([]speeldoos.Part, n)
		}
		for j := 0; j < n; j++ {
			if n > 1 {
				pf.Work.Parts[j].Part = "2222"
			}
			if len(discsize) > 1 {
				fn := path.Join(fmt.Sprintf(Config.Init.DiscFormat, disc_index+1), fmt.Sprintf(Config.Init.TrackFormat, track_counter))
				if dd, ok := detectedSourceFiles[disc_index+1]; ok {
					fn = dd.files[track_counter-1].path
				}

				pf.SourceFiles[j] = speeldoos.SourceFile{
					Filename: fn,
					Disc:     disc_index + 1,
				}
			} else {
				fn := fmt.Sprintf(Config.Init.TrackFormat, track_counter)
				if dd, ok := detectedSourceFiles[0]; ok {
					fn = dd.files[track_counter-1].path
				}
				pf.SourceFiles[j] = speeldoos.SourceFile{
					Filename: fn,
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
