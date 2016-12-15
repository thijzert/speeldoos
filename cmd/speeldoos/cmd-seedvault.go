package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
	tc "github.com/thijzert/go-termcolours"
	"github.com/thijzert/speeldoos"
	"github.com/thijzert/speeldoos/lib/zipmap"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

var zm = zipmap.New()

var (
	checkmark = "\u2713"
)

func init() {
	// The '✓' glyph does not currently display correctly in PowerShell.
	if runtime.GOOS == "windows" {
		checkmark = "Y"
	}
}

func confirmSettings() *speeldoos.Carrier {
	fmt.Printf("\nAbout to start an encode with the following settings:\n")

	fmt.Printf("\nInput XML file: %s %s\n", checkFileExists(Config.Seedvault.InputXml), Config.Seedvault.InputXml)
	fmt.Printf("Output directory: %s\n", Config.Seedvault.OutputDir)

	carrier, err := speeldoos.ImportCarrier(Config.Seedvault.InputXml)
	if err != nil {
		log.Fatal(err)
	}

	discs := make(map[int]int)
	ccp := &commonPath{}
	for _, pf := range carrier.Performances {
		for _, sf := range pf.SourceFiles {
			discs[sf.Disc] = sf.Disc
			ccp.Add(sf.Disc, sf.Filename)
		}
	}

	Config.Seedvault.CoverImage, _ = ccp.FindOne(Config.Seedvault.CoverImage, "cover.jpg", "cover.jpeg", "folder.jpg")
	Config.Seedvault.InlayImage, _ = ccp.FindOne(Config.Seedvault.InlayImage, "inlay.jpg", "inlay.jpeg", "back.jpg")
	Config.Seedvault.Booklet, _ = ccp.FindOne(Config.Seedvault.Booklet, "booklet.pdf")

	var logfile_exists, cuesheet_exists bool
	Config.Seedvault.EACLogfile, logfile_exists = ccp.FindAllDiscs(Config.Seedvault.EACLogfile, "eac.log", "rip.log")
	Config.Seedvault.Cuesheet, cuesheet_exists = ccp.FindAllDiscs(Config.Seedvault.Cuesheet, "cuesheet.cue")

	fmt.Printf("\nCover image:  %s %s\n", checkFileExists(Config.Seedvault.CoverImage), Config.Seedvault.CoverImage)
	fmt.Printf("Inlay image:  %s %s\n", checkFileExists(Config.Seedvault.InlayImage), Config.Seedvault.InlayImage)
	fmt.Printf("Booklet file: %s %s\n", checkFileExists(Config.Seedvault.Booklet), Config.Seedvault.Booklet)

	if len(discs) > 1 {
		fmt.Printf("Discs:       %s\n", ccp.DiscPrefixes())

		fmt.Printf("EAC log file: %s\n", ccp.CheckDiscFileExists(Config.Seedvault.EACLogfile))
		fmt.Printf("Cue sheet:    %s\n", ccp.CheckDiscFileExists(Config.Seedvault.Cuesheet))
	} else {
		fmt.Printf("EAC log file: %s %s\n", checkFileExists(Config.Seedvault.EACLogfile), Config.Seedvault.EACLogfile)
		fmt.Printf("Cue sheet:    %s %s\n", checkFileExists(Config.Seedvault.Cuesheet), Config.Seedvault.Cuesheet)

		logfile_exists = zm.Exists(Config.Seedvault.EACLogfile)
		cuesheet_exists = zm.Exists(Config.Seedvault.Cuesheet)
	}

	if Config.Seedvault.Tracker != "" {
		fmt.Printf("\nURL to private tracker: %s\n", tc.Bwhite(Config.Seedvault.Tracker))
	}

	fmt.Printf("\nEncodes to run: FLAC    %s\n", yes(true))
	fmt.Printf("                MP3-320 %s\n", yes(Config.Seedvault.D320))
	fmt.Printf("                MP3-V0  %s\n", yes(Config.Seedvault.DV0))
	fmt.Printf("                MP3-V2  %s\n", yes(Config.Seedvault.DV2))
	if Config.Seedvault.DV6 {
		fmt.Printf("                MP3-V6  %s\n", yes(Config.Seedvault.DV6))
	}
	fmt.Printf("                archive %s\n", yes(Config.Seedvault.DArchive))
	fmt.Printf("Number of concurrent encoding processes:  %d\n", Config.ConcurrentJobs)

	fmt.Printf("\nIf the above looks good, hit <enter> to continue.\n")
	fmt.Printf("Otherwise, hit Ctrl+C to cancel the process.\n")
	bufio.NewReader(os.Stdin).ReadBytes('\n')

	if Config.Seedvault.CoverImage != "" && !zm.Exists(Config.Seedvault.CoverImage) {
		Config.Seedvault.CoverImage = ""
	}
	if Config.Seedvault.InlayImage != "" && !zm.Exists(Config.Seedvault.InlayImage) {
		Config.Seedvault.InlayImage = ""
	}
	if Config.Seedvault.Booklet != "" && !zm.Exists(Config.Seedvault.Booklet) {
		Config.Seedvault.Booklet = ""
	}

	if carrier.Source == "WEB" {
		if Config.Seedvault.EACLogfile != "" && !logfile_exists {
			Config.Seedvault.EACLogfile = ""
		}
		if Config.Seedvault.Cuesheet != "" && !cuesheet_exists {
			Config.Seedvault.Cuesheet = ""
		}
	}

	return carrier
}

type commonPath struct {
	all   string
	discs map[int]string
}

func (cp *commonPath) Add(disc int, sourcefile string) {
	if cp.discs == nil {
		cp.all = sourcefile
		cp.discs = make(map[int]string)
	} else {
		if len(cp.all) > len(sourcefile) {
			cp.all = cp.all[:len(sourcefile)]
		}
		for cp.all != sourcefile[:len(cp.all)] {
			cp.all = cp.all[:len(cp.all)-1]
		}
	}
	if _, ok := cp.discs[disc]; !ok {
		cp.discs[disc] = sourcefile
	} else {
		if len(cp.discs[disc]) > len(sourcefile) {
			cp.discs[disc] = cp.discs[disc][:len(sourcefile)]
		}
		for cp.discs[disc] != sourcefile[:len(cp.discs[disc])] {
			cp.discs[disc] = cp.discs[disc][:len(cp.discs[disc])-1]
		}
	}
}

func (cp *commonPath) FindOne(names ...string) (rv string, exists bool) {
	for _, nn := range names {
		if nn == "" {
			continue
		}
		ss := cp.all
		for ss != "." && ss != string(filepath.Separator) {
			p := path.Join(ss, nn)
			if zm.Exists(p) {
				return p, true
			}
			ss = filepath.Dir(ss)
		}
		if zm.Exists(nn) {
			return nn, true
		}
	}
	return names[0], false
}

func (cp *commonPath) FindAllDiscs(names ...string) (string, bool) {
	for _, nn := range names {
		ss := cp.all
		for ss != "." && ss != string(filepath.Separator) {
			if cp.existsForEachDisc(ss, nn) {
				return path.Join(ss, nn), true
			}
			ss = filepath.Dir(ss)
		}
		if cp.existsForEachDisc("", nn) {
			return nn, true
		}
	}
	return names[0], false
}

func (cp *commonPath) existsForEachDisc(dir, file string) bool {
	if file == "" {
		return false
	}
	for d, _ := range cp.discs {
		sub := ""
		if d > 0 {
			sub = fmt.Sprintf("disc_%02d", d)
		}
		if !zm.Exists(path.Join(dir, sub, file)) {
			return false
		}
	}
	return true
}

func (cp *commonPath) DiscPrefixes() string {
	if len(cp.discs) <= 1 {
		return ""
	}
	rv := ""
	for d, _ := range cp.discs {
		rv += fmt.Sprintf("% 3d", d)
	}
	return rv
}

func (cp *commonPath) CheckDiscFileExists(file string) string {
	if file == "" {
		return checkFileExists("")
	}

	dir := filepath.Dir(file)
	file = filepath.Base(file)

	rv := ""

	for d, _ := range cp.discs {
		sub := ""
		if d > 0 {
			sub = fmt.Sprintf("disc_%02d", d)
		}
		rv += checkFileExists(path.Join(dir, sub, file)) + "  "
	}

	if dir != "" {
		if len(cp.discs) > 1 {
			file = path.Join(dir, "*", file)
		} else {
			file = path.Join(dir, file)
		}
	}

	return rv + file
}

func checkFileExists(filename string) string {
	if filename == "" {
		return tc.Bblack("(not specified)")
	}

	if zm.Exists(filename) {
		return tc.Green(checkmark)
	}

	return tc.Red("not found")
}

func yes(i bool) string {
	if i {
		return tc.Green("yes")
	}
	return tc.Bblack("no")
}

func seedvault_main(argv []string) {
	foo := confirmSettings()

	track_counter := 0
	current_disc := 0

	log.Printf("Initiating .flac run...")

	albus := newAlbum(foo.Name)
	for _, pf := range foo.Performances {
		if Config.Seedvault.NameAfterComposer {
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

	albus.AdditionalFiles = argv

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

	croak(os.MkdirAll(Config.Seedvault.OutputDir, 0755))
	croak(os.Mkdir(path.Join(Config.Seedvault.OutputDir, cleanFilename(albus.Name)+" [FLAC]"), 0755))

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
			if pf.Work.Composer.Name == "" {
				out = fmt.Sprintf("%02d - %s", track_counter, fileTitle)
			}

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
				croak(os.MkdirAll(path.Join(Config.Seedvault.OutputDir, "wav", sub), 0755))
			}
			dir := path.Join(Config.Seedvault.OutputDir, cleanFilename(albus.Name)+" [FLAC]", sub)
			croak(os.MkdirAll(dir, 0755))

			out = path.Join(dir, cleanFilename(out)+".flac")
			croak(zm.CopyTo(fn, out))

			last_bps = all_bps.Check(out)
		}
	}
	croak(all_bps.Consistent())
	log.Printf("FLAC run complete")

	if last_bps > 16 {
		dir := path.Join(Config.Seedvault.OutputDir, cleanFilename(albus.Name))
		os.Rename(dir+" [FLAC]", dir+fmt.Sprintf(" [FLAC%d]", last_bps))

		log.Printf("Decoding to 16-bit WAV...")
		croak(os.MkdirAll(path.Join(Config.Seedvault.OutputDir, "wav"), 0755))
		albus.Job(fmt.Sprintf("FLAC%d", last_bps), func(mf *mFile, wav, out string) []runner {
			flac := out + ".flac"
			return []runner{
				exec.Command(Config.Tools.Flac, "-d", "-s", "-o", wav, flac),
				metaflac(mf, flac),
			}
		})
		log.Printf("Done decoding")

	} else if Config.Seedvault.DV2 || Config.Seedvault.DV0 || Config.Seedvault.D320 {
		log.Printf("Decoding to WAV...")
		croak(os.MkdirAll(path.Join(Config.Seedvault.OutputDir, "wav"), 0755))
		albus.Job("FLAC", func(mf *mFile, wav, out string) []runner {
			flac := out + ".flac"
			return []runner{
				exec.Command(Config.Tools.Flac, "-d", "-s", "-o", wav, flac),
				metaflac(mf, flac),
			}
		})
		log.Printf("Done decoding.")
	} else {
		albus.Job("FLAC", func(mf *mFile, wav, out string) []runner {
			flac := out + ".flac"
			return []runner{
				metaflac(mf, flac),
			}
		})
	}

	if Config.Seedvault.DV6 {
		log.Printf("Encoding V6 profile...")
		albus.Job("V6", lameRun("-V6", "--vbr-new"))
		log.Printf("Done encoding.")
	}

	if Config.Seedvault.DV2 {
		log.Printf("Encoding V2 profile...")
		albus.Job("V2", lameRun("-V2", "--vbr-new"))
		log.Printf("Done encoding.")
	}

	if Config.Seedvault.DV0 {
		log.Printf("Encoding V0 profile...")
		albus.Job("V0", lameRun("-V0", "--vbr-new"))
		log.Printf("Done encoding.")
	}

	if Config.Seedvault.D320 {
		log.Printf("Encoding 320 profile...")
		albus.Job("320", lameRun("-b", "320"))
		log.Printf("Done encoding.")
	}

	if Config.Seedvault.DArchive {
		log.Printf("Creating speeldoos archive...")
		source_dir := " [FLAC]"
		if last_bps > 16 {
			source_dir = fmt.Sprintf(" [FLAC%d]", last_bps)
		}
		source_dir = cleanFilename(albus.Name) + source_dir

		archive_name := foo.ID
		if archive_name == "" {
			archive_name = "speeldoos"
		}
		archive_name = cleanFilename(archive_name)
		archive_name = strings.Replace(archive_name, " ", "-", -1)

		croak(zipit(path.Join(Config.Seedvault.OutputDir, source_dir), path.Join(Config.Seedvault.OutputDir, archive_name+".zip")))

		bar, err := speeldoos.ImportCarrier(Config.Seedvault.InputXml)
		croak(err)

		h := sha256.New()
		f, err := os.Open(path.Join(Config.Seedvault.OutputDir, archive_name+".zip"))
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

		bar.Write(path.Join(Config.Seedvault.OutputDir, archive_name+".xml"))

		log.Printf("Done.")
	}

	// Delete temporary wav/ dir
	croak(os.RemoveAll(path.Join(Config.Seedvault.OutputDir, "wav")))
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
	value = strings.Replace(value, "*", "-", -1)
	value = strings.Replace(value, "<", "-", -1)
	value = strings.Replace(value, ">", "-", -1)
	value = strings.Replace(value, "|", "-", -1)
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

type runner interface {
	Run() error
}

type funFunc func() error

func (r funFunc) Run() error {
	return r()
}

type jobFun func(*mFile, string, string) []runner

type mFile struct {
	Basename                                string
	Title, Artist, Album                    string
	Composer, Soloist, Orchestra, Conductor string
	Performers                              []string
	Year, Disc, Track                       int
}

type album struct {
	Name            string
	Discs           []int
	Tracks          []*mFile
	AdditionalFiles []string
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
	working_dir := path.Join(Config.Seedvault.OutputDir, cleanFilename(a.Name)+" ["+dir+"]")
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

	cmds := make(chan []runner)
	var wg sync.WaitGroup

	for i := 0; i < Config.ConcurrentJobs; i++ {
		go func() {
			wg.Add(1)
			for cmdList := range cmds {
				for _, toRun := range cmdList {
					if cmd, ok := toRun.(*exec.Cmd); ok {
						cmd.Stdout = os.Stdout
						cmd.Stderr = os.Stderr
						cmd.Start()
						croak(cmd.Wait())
					} else {
						croak(toRun.Run())
					}
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
		wav := path.Join(Config.Seedvault.OutputDir, "wav", sub, cbn+".wav")
		out := path.Join(working_dir, sub, cbn)

		cmds <- fun(mf, wav, out)
	}

	close(cmds)

	if Config.Seedvault.CoverImage != "" {
		err := zm.CopyTo(Config.Seedvault.CoverImage, path.Join(working_dir, "folder.jpg"))
		if err != nil {
			log.Print(err)
		}
		for _, d := range a.Discs {
			if d != 0 {
				dd = fmt.Sprintf("disc_%02d", d)
				err := zm.CopyTo(Config.Seedvault.CoverImage, path.Join(working_dir, dd, "folder.jpg"))
				if err != nil {
					log.Print(err)
				}
			}
		}
	}

	if Config.Seedvault.InlayImage != "" {
		err := zm.CopyTo(Config.Seedvault.InlayImage, path.Join(working_dir, "inlay.jpg"))
		if err != nil {
			log.Print(err)
		}
	}

	if Config.Seedvault.Booklet != "" {
		err := zm.CopyTo(Config.Seedvault.Booklet, path.Join(working_dir, "booklet.pdf"))
		if err != nil {
			log.Print(err)
		}
	}

	// Add any additional files to the root dir
	for _, f := range a.AdditionalFiles {
		err := zm.CopyTo(f, path.Join(working_dir, path.Base(f)))
		if err != nil {
			log.Print(err)
		}
	}

	if Config.Seedvault.Cuesheet != "" {
		for _, d := range a.Discs {
			source_dir := filepath.Dir(Config.Seedvault.Cuesheet)
			source_file := filepath.Base(Config.Seedvault.Cuesheet)
			sd, dd := "", ""
			if d != 0 {
				sd = fmt.Sprintf("disc_%02d", d)
				dd = fmt.Sprintf("disc_%02d", d)
			}
			err := zm.CopyTo(path.Join(source_dir, sd, source_file), path.Join(working_dir, dd, "cuesheet.cue"))
			if err != nil {
				log.Print(err)
			}
		}
	}

	if Config.Seedvault.EACLogfile != "" {
		for _, d := range a.Discs {
			source_dir := filepath.Dir(Config.Seedvault.EACLogfile)
			source_file := filepath.Base(Config.Seedvault.EACLogfile)
			sd, dd := "", ""
			if d != 0 {
				sd = fmt.Sprintf("disc_%02d", d)
				dd = fmt.Sprintf("disc_%02d", d)
			}
			err := zm.CopyTo(path.Join(source_dir, sd, source_file), path.Join(working_dir, dd, "eac.log"))
			if err != nil {
				log.Print(err)
			}
		}
	}

	wg.Wait()

	if Config.Seedvault.Tracker != "" {
		croak(createTorrent(working_dir, working_dir+".torrent", Config.Seedvault.Tracker))
	}
}

func lameRun(extraArgs ...string) jobFun {
	return func(s *mFile, wav, out string) []runner {
		mp3 := out + ".mp3"
		cmdline := []string{"--quiet", "--replaygain-accurate", "--id3v2-only"}

		cmdline = append(cmdline, extraArgs...)
		cmdline = append(cmdline, wav, mp3)

		return []runner{
			exec.Command(Config.Tools.Lame, cmdline...),
			id3tags(s, mp3),
		}
	}
}

func id3tags(s *mFile, mp3 string) *exec.Cmd {
	args := []string{
		"-t", s.Title,
		"-a", s.Artist,
		"--genre", "32", // FIXME
	}
	if s.Composer != "" {
		args = append(args, "--TCOM", s.Composer)
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

	return exec.Command(Config.Tools.ID3v2, args...)
}

func metaflac(mm *mFile, flac string) *exec.Cmd {
	cmd := exec.Command(Config.Tools.Metaflac, "--remove-all-tags", "--no-utf8-convert", "--import-tags-from=-", flac)
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
	if mm.Composer != "" {
		writeFlacTag(pipeIn, "composer", mm.Composer)
	}
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
	cmd := exec.Command(Config.Tools.Metaflac, "--show-bps", file)
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

// Source: https://gist.github.com/svett/424e6784facc0ba907ae
func zipit(source, target string) error {
	zipfile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer zipfile.Close()

	archive := zip.NewWriter(zipfile)
	defer archive.Close()

	info, err := os.Stat(source)
	if err != nil {
		return err
	}

	var baseDir string
	if info.IsDir() {
		// baseDir = filepath.Base(source)
	}

	filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == source {
			return nil
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		if baseDir != "" {
			header.Name = filepath.Join(baseDir, strings.TrimPrefix(path, source))
		}

		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Store
		}

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(writer, file)
		return err
	})

	return err
}

func createTorrent(source, target, announce string) error {
	outf, err := os.Create(target)
	if err != nil {
		return err
	}
	defer outf.Close()

	mi := metainfo.MetaInfo{
		AnnounceList: [][]string{[]string{announce}},
	}
	mi.SetDefaults()
	// mi.CreatedBy = "github.com/thijzert/speeldoos"
	mi.CreationDate = time.Now().Unix()
	mi.Comment = ""

	private := true
	info := metainfo.Info{
		PieceLength: 256 * 1024,
		Private:     &private,
	}
	err = info.BuildFromFilePath(source)
	if err != nil {
		return err
	}

	mi.InfoBytes, err = bencode.Marshal(info)
	if err != nil {
		return err
	}

	return mi.Write(outf)
}
