package main

import (
	"fmt"
	"github.com/thijzert/speeldoos"
	"github.com/thijzert/speeldoos/lib/zipmap"
	"io"
	"log"
	"os"
	"os/exec"
)

func extract_main(args []string) {
	foo, err := speeldoos.ImportCarrier(Config.Extract.InputXml)
	if err != nil {
		log.Fatal(err)
	}

	zm := zipmap.New()

	for _, pf := range foo.Performances {
		title := "(no title)"
		if len(pf.Work.Title) > 0 {
			title = pf.Work.Title[0].Title
		}

		log.Printf("Now processing: %s - %s", pf.Work.Composer.Name, title)

		if len(pf.SourceFiles) == 0 {
			log.Printf("%s - %s has no source files!\n", pf.Work.Composer.Name, title)
		}
		outp := fmt.Sprintf("%s - %s.mp3", pf.Work.Composer.Name, title)

		inp := ""
		for i, _ := range pf.SourceFiles {
			inp = fmt.Sprintf("%s|/proc/self/fd/%d", inp, i+3)
		}
		inp = fmt.Sprintf("concat:%s", inp[1:])

		cmd := exec.Command("ffmpeg", "-y", "-i", inp, "-c:a", "libmp3lame", "-b:a", Config.Extract.Bitrate, outp)
		cmd.ExtraFiles = make([]*os.File, len(pf.SourceFiles))

		pipes := make([]*os.File, len(pf.SourceFiles))
		for i, _ := range pf.SourceFiles {
			cmd.ExtraFiles[i], pipes[i], err = os.Pipe()
			if err != nil {
				log.Fatal(err)
			}
		}

		cmd.Start()

		for i, fn := range pf.SourceFiles {
			f, err := zm.Get(fn.Filename)
			if err != nil {
				log.Fatal(err)
			}

			// Tee hee. The code below superimposes all movements on top of each other.
			// go func() {
			// 	defer f.Close()
			// 	defer pipes[i].Close()
			// 	io.Copy(pipes[i], f)
			// }()
			_, err = io.Copy(pipes[i], f)
			if err != nil {
				log.Fatal(err)
			}
			f.Close()
			pipes[i].Close()
		}

		err = cmd.Wait()
		if err != nil {
			log.Fatal(err)
		}
	}
	log.Printf("Finished processing\n")
}
