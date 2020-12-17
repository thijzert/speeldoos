package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"
	"time"

	tc "github.com/thijzert/go-termcolours"
	"github.com/thijzert/speeldoos/lib/hivemind"
	"github.com/thijzert/speeldoos/lib/wavreader"
	"github.com/thijzert/speeldoos/lib/ziptraverser"
	speeldoos "github.com/thijzert/speeldoos/pkg"
)

type condenseJob struct {
	Wavconf   wavreader.Config
	Carrier   *speeldoos.Carrier
	OutputDir string
}

func (job condenseJob) Run(h hivemind.JC) error {
	zm := ziptraverser.New()
	defer zm.Close()

	h.SetTitle(job.Carrier.ID)
	var err error

	done := false
	for _, pf := range job.Carrier.Performances {
		for _, sf := range pf.SourceFiles {
			dn := path.Join(Config.LibraryDir, path.Dir(sf.Filename), "folder.jpg")
			if zm.Exists(dn) {
				if err = zm.CopyTo(dn, path.Join(job.OutputDir, "folder.jpg")); err != nil {
					done = true
					break
				}
			}
		}
		if done {
			break
		}
	}

	for _, pf := range job.Carrier.Performances {
		title := "(no title)"
		if len(pf.Work.Title) > 0 {
			title = pf.Work.Title[0].Title
		}

		h.Printf("%s: now processing: %s - %s", tc.Green(job.Carrier.ID), pf.Work.Composer.Name, title)

		if len(pf.SourceFiles) == 0 {
			h.Printf("%s - %s has no source files!\n", pf.Work.Composer.Name, title)
			continue
		}
		outp := fmt.Sprintf("%s - %s.mp3", pf.Work.Composer.Name, title)
		outp = path.Join(job.OutputDir, outp)

		// set up mp3writer
		out, err := os.Create(outp)
		if err != nil {
			return err
		}
		defer out.Close()

		var wout io.WriteCloser = nil

		for _, fn := range pf.SourceFiles {
			f, err := zm.Get(path.Join(Config.LibraryDir, fn.Filename))

			wav, err := job.Wavconf.FromFLAC(f)
			if err != nil {
				h.Println(err.Error())
				continue
			}
			wav.Init()
			defer wav.Close()

			if wout == nil {
				wout, err = job.Wavconf.ToMP3(out, wav.Format())
				if err != nil {
					h.Println(err.Error())
					continue
				}
			}

			_, err = io.Copy(wout, wav)
			if err != nil {
				return err
			}
		}

		if wout == nil {
			break
		}
		wout.Close()

		tags := &mFile{
			Artist:     pf.Work.Composer.Name, // MP3 players are dumb
			Soloist:    pf.Work.Composer.Name, // very dumb
			Album:      job.Carrier.Name,
			Composer:   pf.Work.Composer.Name,
			Performers: make([]string, 0, 2),
			Year:       pf.Work.Year, // Furthermore, the `id3v2` program is also extremely dumb, as it seems to ignore pre-1900 dates
		}
		for _, tit := range pf.Work.Title {
			tags.Title = tit.Title
			break
		}
		for _, p := range pf.Performers {
			tags.Performers = append(tags.Performers, p.Name)

			if p.Role == "soloist" || p.Role == "performer" || p.Role == "" || p.Role == "orchestra" || p.Role == "ensemble" {
				if tags.Orchestra == "" {
					tags.Orchestra = p.Name
				} else {
					tags.Orchestra = tags.Orchestra + "/" + p.Name
				}
			} else if p.Role == "conductor" {
				if tags.Conductor == "" {
					tags.Conductor = p.Name
				} else {
					tags.Conductor = tags.Conductor + "/" + p.Name
				}
			}
		}

		cmd := id3tags(tags, outp)
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	if err := os.Chtimes(job.OutputDir, time.Now(), time.Now()); err != nil {
		return err
	}
	return nil
}

func condense_main(args []string) {
	wavconf := wavreader.Config{
		LamePath:   Config.Tools.Lame,
		FlacPath:   Config.Tools.Flac,
		VBRQuality: Config.Condense.Quality,
	}

	d, err := allCarriers()
	croak(err)

	croak(os.MkdirAll(Config.Condense.OutputDir, 0755))

	hive := hivemind.New(Config.ConcurrentJobs)

	for _, pc := range d {
		foo := pc.Carrier

		if foo.ID == "" {
			log.Printf("Carrier in file '%s' has an empty ID. Skipping.", pc.Filename)
			continue
		}

		file, err := os.Stat(pc.Filename)
		croak(err)

		outdir := foo.ID
		outdir = strings.Replace(outdir, "/", "-", -1)
		outdir = strings.Replace(outdir, "\n", " ", -1)
		outdir = path.Join(Config.Condense.OutputDir, outdir)
		iout, err := os.Stat(outdir)
		if err == nil {
			if iout.ModTime().Before(file.ModTime()) {
				log.Printf("removing %s", outdir)
				os.RemoveAll(outdir)
			} else {
				log.Printf("skipping '%s' - up to date", foo.ID)
				continue
			}
		}
		croak(os.Mkdir(outdir, 0755))
		tim := file.ModTime().Add(-24 * time.Hour)
		croak(os.Chtimes(outdir, tim, tim))

		hive.AddJob(condenseJob{wavconf, foo, outdir})
	}

	hive.Wait()
}
