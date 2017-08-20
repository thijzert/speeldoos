package main

import (
	"fmt"
	tc "github.com/thijzert/go-termcolours"
	"github.com/thijzert/speeldoos"
	"github.com/thijzert/speeldoos/lib/zipmap"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

type condenseJob struct {
	Carrier   *speeldoos.Carrier
	OutputDir string
}

func condense_main(args []string) {
	d, err := os.Open(Config.LibraryDir)
	croak(err)

	files, err := d.Readdir(0)
	croak(err)

	croak(os.MkdirAll(Config.Condense.OutputDir, 0755))

	jobs := make(chan condenseJob)
	wg := &sync.WaitGroup{}

	for i := 0; i < Config.ConcurrentJobs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

		NextJob:
			for job := range jobs {

				zm := zipmap.New()
				defer zm.Close()

				shout := func(e error) bool {
					if e != nil {
						log.Printf("%s: %s", tc.Red(job.Carrier.ID), e)
						return true
					}
					return false
				}

				done := false
				for _, pf := range job.Carrier.Performances {
					for _, sf := range pf.SourceFiles {
						dn := path.Join(Config.LibraryDir, path.Dir(sf.Filename), "folder.jpg")
						if zm.Exists(dn) {
							if !shout(zm.CopyTo(dn, path.Join(job.OutputDir, "folder.jpg"))) {
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

					log.Printf("%s: now processing: %s - %s", tc.Green(job.Carrier.ID), pf.Work.Composer.Name, title)

					if len(pf.SourceFiles) == 0 {
						log.Printf("%s - %s has no source files!\n", pf.Work.Composer.Name, title)
					}
					outp := fmt.Sprintf("%s - %s.mp3", pf.Work.Composer.Name, title)
					outp = path.Join(job.OutputDir, outp)

					inp := ""
					for i, _ := range pf.SourceFiles {
						inp = fmt.Sprintf("%s|/proc/self/fd/%d", inp, i+3)
					}
					inp = fmt.Sprintf("concat:%s", inp[1:])

					cmd := exec.Command("ffmpeg", "-v", "8", "-y", "-i", inp, "-c:a", "libmp3lame", "-q:a", strconv.Itoa(Config.Condense.Quality), outp)
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					cmd.ExtraFiles = make([]*os.File, len(pf.SourceFiles))

					pipes := make([]*os.File, len(pf.SourceFiles))
					for i, _ := range pf.SourceFiles {
						cmd.ExtraFiles[i], pipes[i], err = os.Pipe()
						if shout(err) {
							continue NextJob
						}
					}

					cmd.Start()

					for i, fn := range pf.SourceFiles {
						f, err := zm.Get(path.Join(Config.LibraryDir, fn.Filename))
						if shout(err) {
							continue NextJob
						}

						_, err = io.Copy(pipes[i], f)
						if shout(err) {
							continue NextJob
						}
						f.Close()
						pipes[i].Close()
					}

					if shout(cmd.Wait()) {
						continue NextJob
					}

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

					cmd = id3tags(tags, outp)
					if shout(cmd.Run()) {
						continue NextJob
					}
				}

				if shout(os.Chtimes(job.OutputDir, time.Now(), time.Now())) {
					continue NextJob
				}
			}
		}()
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		xml := path.Join(Config.LibraryDir, file.Name())
		if len(xml) < 5 || xml[len(xml)-4:] != ".xml" {
			continue
		}

		foo, err := speeldoos.ImportCarrier(xml)
		if err != nil {
			log.Print(err)
			continue
		}

		if foo.ID == "" {
			log.Printf("Carrier in file '%s' has an empty ID. Skipping.", file.Name())
			continue
		}

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

		jobs <- condenseJob{foo, outdir}
	}

	close(jobs)

	wg.Wait()
}
