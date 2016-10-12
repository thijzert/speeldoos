package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"github.com/thijzert/speeldoos"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

var (
	input_xml   = flag.String("input_xml", "", "Input XML file")
	cover_image = flag.String("cover_image", "", "Path to cover image")
	output_dir  = flag.String("output_dir", "seedvault", "Output directory")
)

func init() {
	flag.Parse()
}

func main() {
	foo, err := speeldoos.ImportCarrier(*input_xml)
	if err != nil {
		log.Fatal(err)
	}

	zm := &ZipMap{}
	track_counter := 0

	log.Printf("Initiating .flac run...")

	croak(os.Mkdir(*output_dir, 0755))
	croak(os.Mkdir(path.Join(*output_dir, "flac"), 0755))

	if *cover_image != "" {
		f, err := os.Open(*cover_image)
		croak(err)
		g, err := os.Create(path.Join(*output_dir, "flac", "cover.jpeg"))
		croak(err)
		_, err = io.Copy(g, f)
		croak(err)
	}

	for _, pf := range foo.Performances {
		title := "(no title)"
		if len(pf.Work.Title) > 0 {
			title = pf.Work.Title[0].Title
		}

		log.Printf("Now processing: %s - %s", pf.Work.Composer.Name, title)

		if len(pf.SourceFiles) == 0 {
			log.Printf("%s - %s has no source files!\n", pf.Work.Composer.Name, title)
			continue
		}

		for i, fn := range pf.SourceFiles {
			track_counter++
			fileTitle := title
			if len(pf.Work.Parts) > 1 && len(pf.Work.Parts) > i {
				fileTitle = fmt.Sprintf("%s - %d. %s", title, i+1, pf.Work.Parts[i])
			}
			out := fmt.Sprintf("%02d - %s - %s.flac", track_counter, pf.Work.Composer.Name, fileTitle)
			out = path.Join(*output_dir, "flac", out)

			f, err := os.Create(out)
			croak(err)
			g, err := zm.Get(fn)
			croak(err)

			_, err = io.Copy(f, g)
			croak(err)
			f.Close()
			g.Close()

			cmd := exec.Command("metaflac", "--remove-all-tags", "--no-utf8-convert", "--import-tags-from=-", out)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			pipeIn, err := cmd.StdinPipe()
			croak(err)
			cmd.Start()

			writeTag(pipeIn, "title", fileTitle)
			writeTag(pipeIn, "artist", pf.Work.Composer.Name)
			writeTag(pipeIn, "composer", pf.Work.Composer.Name)
			for _, p := range pf.Performers {
				writeTag(pipeIn, "performer", p.Name)
			}
			pipeIn.Close()
			croak(cmd.Wait())
		}
	}
	log.Printf("FLAC run complete\n")
}

type ZipMap struct {
	zips map[string]*zip.ReadCloser
}

func (z *ZipMap) Get(filename string) (io.ReadCloser, error) {
	rv, _ := os.Open(os.DevNull)

	if z.zips == nil {
		z.zips = make(map[string]*zip.ReadCloser)
	}

	var err error

	// Try opening the file itself, maybe that works...
	fi, err := os.Stat(filename)
	if err == nil {
		// Is it a regular file?
		if (fi.Mode() & os.ModeType) == 0 {
			return os.Open(filename)
		}
	}

	// FIXME: I'm of the opinion that this should work: elems := filepath.SplitList(filename)
	elems := strings.Split(filename, "/")
	for i, elem := range elems {
		if len(elem) < 5 || elem[len(elem)-4:] != ".zip" {
			continue
		}
		zipfile := filepath.Join(elems[0 : i+1]...)
		read, ok := z.zips[zipfile]
		if !ok {
			log.Printf("Opening zip file %s...\n", zipfile)
			read, err = zip.OpenReader(zipfile)
			if err != nil {
				log.Print(err)
				read = nil
			}
			z.zips[zipfile] = read
		}

		if read == nil {
			continue
		}

		localfile := filepath.Join(elems[i+1:]...)

		for _, zfp := range read.File {
			if zfp.Name == localfile {
				return zfp.Open()
			}
		}

		return rv, os.ErrNotExist
	}

	return rv, os.ErrNotExist
}

func croak(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

func writeTag(f io.Writer, key, value string) {
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
