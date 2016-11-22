package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/thijzert/go-rcfile"
	tc "github.com/thijzert/go-termcolours"
	"github.com/thijzert/speeldoos"
	"github.com/thijzert/speeldoos/lib/zipmap"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
)

var (
	input_xml  = flag.String("input_xml", "", "Input XML file")
	output_dir = flag.String("output_dir", "seedvault", "Output directory")

	cover_image = flag.String("cover_image", "", "Path to cover image")
	inlay_image = flag.String("inlay_image", "", "Path to inlay image")
	booklet     = flag.String("booklet", "", "Path to booklet PDF")
	eac_logfile = flag.String("eac_logfile", "", "Path to EAC log file")
	cuesheet    = flag.String("cuesheet", "", "Path to cuesheet")

	name_after_composer = flag.Bool("name_after_composer", false, "Name the album after the first composer rather than the first performer")

	tracker_url = flag.String("tracker", "", "URL to private tracker")

	do_archive = flag.Bool("archive", true, "Create a speeldoos archive")
	do_320     = flag.Bool("320", false, "Also encode MP3-320")
	do_v0      = flag.Bool("v0", false, "Also encode V0")
	do_v2      = flag.Bool("v2", false, "Also encode V2")
	do_v6      = flag.Bool("v6", false, "Also encode V6 (for audiobooks)")

	conc_jobs = flag.Int("j", 2, "Number of concurrent jobs")
)

var zm = zipmap.New()

func init() {
	rcfile.Parse()
	flag.Parse()

	if *conc_jobs < 1 {
		*conc_jobs = 1
	}
}

func confirmSettings() *speeldoos.Carrier {
	fmt.Printf("\nAbout to start an encode with the following settings:\n")

	fmt.Printf("\nInput XML file: %s %s\n", checkFileExists(*input_xml), *input_xml)
	fmt.Printf("Output directory: %s\n", *output_dir)

	carrier, err := speeldoos.ImportCarrier(*input_xml)
	if err != nil {
		log.Fatal(err)
	}

	discs := make(map[int]int)
	for _, pf := range carrier.Performances {
		for _, sf := range pf.SourceFiles {
			discs[sf.Disc] = sf.Disc
		}
	}

	fmt.Printf("\nCover image:  %s %s\n", checkFileExists(*cover_image), *cover_image)
	fmt.Printf("Inlay image:  %s %s\n", checkFileExists(*inlay_image), *inlay_image)
	fmt.Printf("Booklet file: %s %s\n", checkFileExists(*booklet), *booklet)

	logfile_exists, cuesheet_exists := true, true

	if len(discs) > 1 {
		fmt.Printf("Discs:       ")
		for d, _ := range discs {
			fmt.Printf("% 3d", d)
		}
		fmt.Printf("\n")

		fmt.Printf("EAC log file: ")
		if *eac_logfile == "" {
			fmt.Printf("%s\n", checkFileExists(""))
		} else {
			for d, _ := range discs {
				sub := ""
				if d > 0 {
					sub = fmt.Sprintf("disc_%02d", d)
				}
				fmt.Printf("%s  ", checkFileExists(path.Join(sub, *eac_logfile)))
				if !zm.Exists(path.Join(sub, *eac_logfile)) {
					logfile_exists = false
				}
			}
			fmt.Printf("%s\n", *eac_logfile)
		}

		fmt.Printf("Cue sheet:    ")
		if *cuesheet == "" {
			fmt.Printf("%s\n", checkFileExists(""))
		} else {
			for d, _ := range discs {
				sub := ""
				if d > 0 {
					sub = fmt.Sprintf("disc_%02d", d)
				}
				fmt.Printf("%s  ", checkFileExists(path.Join(sub, *cuesheet)))
				if !zm.Exists(path.Join(sub, *cuesheet)) {
					cuesheet_exists = false
				}
			}
			fmt.Printf("%s\n", *cuesheet)
		}
	} else {
		fmt.Printf("EAC log file: %s %s\n", checkFileExists(*eac_logfile), *eac_logfile)
		fmt.Printf("Cue sheet:    %s %s\n", checkFileExists(*cuesheet), *cuesheet)

		logfile_exists = zm.Exists(*eac_logfile)
		cuesheet_exists = zm.Exists(*cuesheet)
	}

	if *tracker_url != "" {
		fmt.Printf("\nURL to private tracker: %s\n", tc.Blue(*tracker_url))
	}

	fmt.Printf("\nEncodes to run: FLAC    %s\n", yes(true))
	fmt.Printf("                MP3-320 %s\n", yes(*do_320))
	fmt.Printf("                MP3-V0  %s\n", yes(*do_v0))
	fmt.Printf("                MP3-V2  %s\n", yes(*do_v2))
	if *do_v6 {
		fmt.Printf("                MP3-V6  %s\n", yes(*do_v6))
	}
	fmt.Printf("                archive %s\n", yes(*do_archive))
	fmt.Printf("Number of concurrent encoding processes:  %d\n", *conc_jobs)

	fmt.Printf("\nIf the above looks good, hit <enter> to continue.\n")
	fmt.Printf("Otherwise, hit Ctrl+C to cancel the process.\n")
	bufio.NewReader(os.Stdin).ReadBytes('\n')

	if *cover_image != "" && !zm.Exists(*cover_image) {
		*cover_image = ""
	}
	if *inlay_image != "" && !zm.Exists(*inlay_image) {
		*inlay_image = ""
	}
	if *booklet != "" && !zm.Exists(*booklet) {
		*booklet = ""
	}

	if carrier.Source == "WEB" {
		if *eac_logfile != "" && !logfile_exists {
			*eac_logfile = ""
		}
		if *cuesheet != "" && !cuesheet_exists {
			*cuesheet = ""
		}
	}

	return carrier
}

func checkFileExists(filename string) string {
	if filename == "" {
		return tc.Bblack("(not specified)")
	}

	if zm.Exists(filename) {
		return tc.Green("\u2713")
	}

	return tc.Red("not found")
}

func yes(i bool) string {
	if i {
		return tc.Green("yes")
	}
	return tc.Bblack("no")
}

func main() {
	foo := confirmSettings()

	track_counter := 0
	current_disc := 0

	log.Printf("Initiating .flac run...")

	albus := newAlbum(foo.Name)
	for _, pf := range foo.Performances {
		if *name_after_composer {
			if pf.Work.Composer.Name != "" {
				albus.Name = pf.Work.Composer.Name + " - " + foo.Name
			}
		} else {
			for _, p := range pf.Performers {
				if p.Name != "" {
					albus.Name = p.Name + " - " + foo.Name
					break
				}
			}
		}
		if albus.Name != foo.Name {
			break
		}
	}
	if foo.Source != "" {
		albus.Name = albus.Name + fmt.Sprintf(" [%s]", foo.Source)
	}

	discs := make(map[int]int)
	for _, pf := range foo.Performances {
		for _, sf := range pf.SourceFiles {
			discs[sf.Disc] = sf.Disc
		}
	}
	albus.Discs = make([]int, 0, len(discs))
	for d, _ := range discs {
		albus.Discs = append(albus.Discs, d)
	}

	croak(os.Mkdir(*output_dir, 0755))
	croak(os.Mkdir(path.Join(*output_dir, cleanFilename(albus.Name)+" [FLAC]"), 0755))

	last_bps := 0
	all_bps := &bitness{}

	for _, pf := range foo.Performances {
		title := "(no title)"
		if len(pf.Work.Title) > 0 {
			title = pf.Work.Title[0].Title
		}

		if pf.Work.OpusNumber != nil && len(pf.Work.OpusNumber) > 0 {
			title = title + ", " + pf.Work.OpusNumber[0].String()
		}

		log.Printf("Now processing: %s - %s", pf.Work.Composer.Name, title)

		if len(pf.SourceFiles) == 0 {
			log.Printf("%s - %s has no source files!\n", pf.Work.Composer.Name, title)
			continue
		}

		for i, sf := range pf.SourceFiles {
			fn := sf.Filename
			track_counter++

			if sf.Disc != current_disc {
				track_counter = 1
				current_disc = sf.Disc
			}

			fileTitle := title
			if len(pf.Work.Parts) > 1 && len(pf.Work.Parts) > i {
				fileTitle = fmt.Sprintf("%s - %d. %s", title, i+1, pf.Work.Parts[i])
			}
			out := fmt.Sprintf("%02d - %s - %s", track_counter, pf.Work.Composer.Name, fileTitle)

			mm := &mFile{
				Basename:   out,
				Title:      fileTitle,
				Album:      foo.Name,
				Composer:   pf.Work.Composer.Name,
				Performers: make([]string, 0, 2),
				Year:       pf.Year,
				Track:      track_counter,
				Disc:       sf.Disc,
			}

			for _, p := range pf.Performers {
				mm.Performers = append(mm.Performers, p.Name)

				if mm.Artist == "" {
					mm.Artist = p.Name
				}

				if p.Role == "soloist" || p.Role == "performer" || p.Role == "" {
					if mm.Soloist == "" {
						mm.Soloist = p.Name
					} else {
						mm.Soloist = mm.Soloist + "/" + p.Name
					}
				} else if p.Role == "orchestra" || p.Role == "ensemble" {
					if mm.Orchestra == "" {
						mm.Orchestra = p.Name
					} else {
						mm.Orchestra = mm.Orchestra + "/" + p.Name
					}
				} else if p.Role == "conductor" {
					if mm.Conductor == "" {
						mm.Conductor = p.Name
					} else {
						mm.Conductor = mm.Conductor + "/" + p.Name
					}
				}
			}

			albus.Add(mm)

			sub := ""
			if sf.Disc != 0 {
				sub = fmt.Sprintf("disc_%02d", sf.Disc)
				croak(os.MkdirAll(path.Join(*output_dir, "wav", sub), 0755))
			}
			dir := path.Join(*output_dir, cleanFilename(albus.Name)+" [FLAC]", sub)
			croak(os.MkdirAll(dir, 0755))

			out = path.Join(dir, cleanFilename(out)+".flac")
			croak(zm.CopyTo(fn, out))

			last_bps = all_bps.Check(out)
		}
	}
	croak(all_bps.Consistent())
	log.Printf("FLAC run complete")

	if last_bps > 16 {
		log.Fatalf("You appear to have a FLAC%d source. Congrats on that, but it isn't yet supported.", last_bps)
	} else if *do_v2 || *do_v0 || *do_320 {
		log.Printf("Decoding to WAV...")
		croak(os.MkdirAll(path.Join(*output_dir, "wav"), 0755))
		albus.Job("FLAC", func(mf *mFile, wav, out string) []*exec.Cmd {
			flac := out + ".flac"
			return []*exec.Cmd{
				exec.Command("flac", "-d", "-s", "-o", wav, flac),
				metaflac(mf, flac),
			}
		})
		log.Printf("Done decoding.")
	} else {
		albus.Job("FLAC", func(mf *mFile, wav, out string) []*exec.Cmd {
			flac := out + ".flac"
			return []*exec.Cmd{
				metaflac(mf, flac),
			}
		})
	}

	if *do_v6 {
		log.Printf("Encoding V6 profile...")
		albus.Job("V6", lameRun("-V6", "--vbr-new"))
		log.Printf("Done encoding.")
	}

	if *do_v2 {
		log.Printf("Encoding V2 profile...")
		albus.Job("V2", lameRun("-V2", "--vbr-new"))
		log.Printf("Done encoding.")
	}

	if *do_v0 {
		log.Printf("Encoding V0 profile...")
		albus.Job("V0", lameRun("-V0", "--vbr-new"))
		log.Printf("Done encoding.")
	}

	if *do_320 {
		log.Printf("Encoding 320 profile...")
		albus.Job("320", lameRun("-b", "320"))
		log.Printf("Done encoding.")
	}

	if *do_archive {
		archive_name := foo.ID
		if archive_name == "" {
			archive_name = "speeldoos"
		}
		archive_name = cleanFilename(archive_name)
		archive_name = strings.Replace(archive_name, " ", "-", -1)
		c := exec.Command("zip", "--quiet", "-r", "-Z", "store", path.Join("..", archive_name+".zip"), ".")
		c.Dir = path.Join(*output_dir, cleanFilename(albus.Name)+" [FLAC]")
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		c.Start()
		croak(c.Wait())

		bar, err := speeldoos.ImportCarrier(*input_xml)
		croak(err)

		h := sha256.New()
		f, err := os.Open(path.Join(*output_dir, archive_name+".zip"))
		croak(err)
		_, err = io.Copy(h, f)
		croak(err)

		bar.Hash = "sha256-" + hex.EncodeToString(h.Sum(nil))

		sourcefile_counter := 0
		for i, pf := range bar.Performances {
			for j, sf := range pf.SourceFiles {
				disc := ""
				if sf.Disc != 0 {
					disc = fmt.Sprintf("disc_%02d", sf.Disc)
				}
				bar.Performances[i].SourceFiles[j].Filename = path.Join(archive_name+".zip", disc, cleanFilename(albus.Tracks[sourcefile_counter].Basename)+".flac")
				sourcefile_counter++
			}
		}

		bar.Write(path.Join(*output_dir, archive_name+".xml"))
	}
}

func croak(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

func ncroak(n int, e error) int {
	croak(e)
	return n
}

func cleanFilename(value string) string {
	value = strings.Replace(value, "\n", " ", -1)
	value = strings.Replace(value, "\r", " ", -1)
	value = strings.Replace(value, "\t", " ", -1)
	value = strings.Replace(value, "\x00", " ", -1)
	value = strings.Replace(value, "=", "", -1)
	value = strings.Replace(value, "\"", "", -1)
	value = strings.Replace(value, ":", "", -1)
	value = strings.Replace(value, "?", "", -1)
	value = strings.Replace(value, "!", "", -1)
	value = strings.Replace(value, "/", "-", -1)
	value = strings.Replace(value, "\\", "-", -1)

	return value
}

func writeFlacTag(f io.Writer, key, value string) {
	if value == "" {
		return
	}
	value = strings.Replace(value, "\n", " ", -1)
	value = strings.Replace(value, "\r", " ", -1)
	value = strings.Replace(value, "\t", " ", -1)
	value = strings.Replace(value, "\x00", " ", -1)
	value = strings.Replace(value, "=", "-", -1)

	d := fmt.Sprintf("%s=%s\n", strings.ToUpper(key), value)

	f.Write([]byte(d))
}

type jobFun func(*mFile, string, string) []*exec.Cmd

type mFile struct {
	Basename                                string
	Title, Artist, Album                    string
	Composer, Soloist, Orchestra, Conductor string
	Performers                              []string
	Year, Disc, Track                       int
}

type album struct {
	Name   string
	Discs  []int
	Tracks []*mFile
}

func newAlbum(name string) *album {
	return &album{
		Name:   name,
		Tracks: make([]*mFile, 0, 20),
	}
}

func (a *album) Add(mm *mFile) {
	a.Tracks = append(a.Tracks, mm)
}

func (a *album) Job(dir string, fun jobFun) {
	working_dir := path.Join(*output_dir, cleanFilename(a.Name)+" ["+dir+"]")
	croak(os.MkdirAll(working_dir, 0755))

	discs := make(map[int]int)
	for _, mf := range a.Tracks {
		if mf.Disc != 0 {
			found, _ := discs[mf.Disc]
			if found != 3 {
				croak(os.MkdirAll(path.Join(working_dir, fmt.Sprintf("disc_%02d", mf.Disc)), 0755))
				discs[mf.Disc] = 3
			}
		}
	}

	cmds := make(chan []*exec.Cmd)
	var wg sync.WaitGroup

	for i := 0; i < *conc_jobs; i++ {
		go func() {
			wg.Add(1)
			for cmdList := range cmds {
				for _, cmd := range cmdList {
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					cmd.Start()
					croak(cmd.Wait())
				}
			}
			wg.Done()
		}()
	}

	for _, mf := range a.Tracks {
		sub := ""
		if mf.Disc != 0 {
			sub = fmt.Sprintf("disc_%02d", mf.Disc)
		}
		cbn := cleanFilename(mf.Basename)
		wav := path.Join(*output_dir, "wav", sub, cbn+".wav")
		out := path.Join(working_dir, sub, cbn)

		cmds <- fun(mf, wav, out)
	}

	close(cmds)

	if *cover_image != "" {
		err := zm.CopyTo(*cover_image, path.Join(working_dir, "folder.jpg"))
		if err != nil {
			log.Print(err)
		}
	}

	if *inlay_image != "" {
		err := zm.CopyTo(*inlay_image, path.Join(working_dir, "inlay.jpg"))
		if err != nil {
			log.Print(err)
		}
	}

	if *booklet != "" {
		err := zm.CopyTo(*booklet, path.Join(working_dir, "booklet.pdf"))
		if err != nil {
			log.Print(err)
		}
	}

	// Add any additional files to the root dir
	args := flag.Args()
	for _, f := range args {
		err := zm.CopyTo(f, path.Join(working_dir, path.Base(f)))
		if err != nil {
			log.Print(err)
		}
	}

	if *cuesheet != "" {
		for _, d := range a.Discs {
			sd, dd := "", ""
			if d != 0 {
				sd = fmt.Sprintf("disc_%02d", d)
				dd = fmt.Sprintf("disc_%02d", d)
			}
			err := zm.CopyTo(path.Join(sd, *cuesheet), path.Join(working_dir, dd, "cuesheet.cue"))
			if err != nil {
				log.Print(err)
			}
		}
	}

	if *eac_logfile != "" {
		for _, d := range a.Discs {
			sd, dd := "", ""
			if d != 0 {
				sd = fmt.Sprintf("disc_%02d", d)
				dd = fmt.Sprintf("disc_%02d", d)
			}
			err := zm.CopyTo(path.Join(sd, *eac_logfile), path.Join(working_dir, dd, "eac.log"))
			if err != nil {
				log.Print(err)
			}
		}
	}

	wg.Wait()

	if *tracker_url != "" {
		// working_dir := path.Join(*output_dir, a.Name+" ["+dir+"]")
		cmd := exec.Command("mktorrent", "-a", *tracker_url, "-p", working_dir, "-o", working_dir+".torrent")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Start()
		croak(cmd.Wait())
	}
}

func lameRun(extraArgs ...string) jobFun {
	return func(s *mFile, wav, out string) []*exec.Cmd {
		mp3 := out + ".mp3"
		cmdline := []string{"--quiet", "--replaygain-accurate", "--id3v2-only"}

		cmdline = append(cmdline, extraArgs...)
		cmdline = append(cmdline, wav, mp3)

		return []*exec.Cmd{
			exec.Command("lame", cmdline...),
			id3tags(s, mp3),
		}
	}
}

func id3tags(s *mFile, mp3 string) *exec.Cmd {
	args := []string{
		"-t", s.Title,
		"-a", s.Artist,
		"--TCOM", s.Composer,
		"--genre", "32", // FIXME
	}
	if s.Album != "" {
		args = append(args, "-A", s.Album)
	}

	if s.Year != 0 {
		args = append(args, "-y", fmt.Sprintf("%d", s.Year))
	}
	if s.Track != 0 {
		args = append(args, "-T", fmt.Sprintf("%d", s.Track))
	}

	if s.Soloist != "" {
		args = append(args, "--TPE1", s.Soloist)
	}
	if s.Orchestra != "" {
		args = append(args, "--TPE2", s.Orchestra)
	}
	if s.Conductor != "" {
		args = append(args, "--TPE3", s.Conductor)
	}

	args = append(args, mp3)

	return exec.Command("id3v2", args...)
}

func metaflac(mm *mFile, flac string) *exec.Cmd {
	cmd := exec.Command("metaflac", "--remove-all-tags", "--no-utf8-convert", "--import-tags-from=-", flac)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cecinestpas, err := cmd.StdinPipe()
	croak(err)

	pipeIn := &bytes.Buffer{}

	writeFlacTag(pipeIn, "title", mm.Title)
	writeFlacTag(pipeIn, "artist", mm.Artist)
	writeFlacTag(pipeIn, "album", mm.Album)
	if mm.Disc != 0 {
		writeFlacTag(pipeIn, "discnumber", fmt.Sprintf("%d", mm.Disc))
	}
	writeFlacTag(pipeIn, "tracknumber", fmt.Sprintf("%d", mm.Track))
	writeFlacTag(pipeIn, "composer", mm.Composer)
	if mm.Conductor != "" {
		writeFlacTag(pipeIn, "conductor", mm.Conductor)
	}
	for _, p := range mm.Performers {
		writeFlacTag(pipeIn, "performer", p)
	}
	writeFlacTag(pipeIn, "date", fmt.Sprintf("%d", mm.Year))
	writeFlacTag(pipeIn, "genre", "classical") // FIXME

	go func() {
		pipeIn.WriteTo(cecinestpas)
		cecinestpas.Close()
	}()
	return cmd
}

type bitness struct {
	seen map[int]int
}

func (b *bitness) Check(file string) int {
	cmd := exec.Command("metaflac", "--show-bps", file)
	cmd.Stderr = os.Stderr
	cecinestpas, err := cmd.StdoutPipe()
	croak(err)
	croak(cmd.Start())
	rv := 0
	ncroak(fmt.Fscanln(cecinestpas, &rv))
	croak(cmd.Wait())

	if b.seen == nil {
		b.seen = make(map[int]int)
	}
	if b.seen[rv] == 0 {
		b.seen[rv] = 1
	} else {
		b.seen[rv] = b.seen[rv] + 1
	}

	return rv
}

func (b *bitness) Consistent() error {
	if b.seen == nil || len(b.seen) == 0 {
		return nil
	}
	if len(b.seen) == 1 {
		return nil
	}

	rv := ""
	for k, v := range b.seen {
		rv = fmt.Sprintf("%s, %dx %dbit", rv, v, k)
	}

	return fmt.Errorf("Inconsistent bit depth across source files: I've seen %s", rv[2:])
}
