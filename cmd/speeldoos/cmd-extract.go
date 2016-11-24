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
	if len(args) == 0 {
		log.Fatal("Specify at least one XML file to extract")
	}
	for _, xml := range args {
		foo, err := speeldoos.ImportCarrier(xml)
		croak(err)

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

			cmd := exec.Command("ffmpeg", "-v", "8", "-y", "-i", inp, "-c:a", "libmp3lame", "-b:a", Config.Extract.Bitrate, outp)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.ExtraFiles = make([]*os.File, len(pf.SourceFiles))

			pipes := make([]*os.File, len(pf.SourceFiles))
			for i, _ := range pf.SourceFiles {
				cmd.ExtraFiles[i], pipes[i], err = os.Pipe()
				croak(err)
			}

			cmd.Start()

			for i, fn := range pf.SourceFiles {
				f, err := zm.Get(fn.Filename)
				croak(err)

				// Tee hee. The code below superimposes all movements on top of each other.
				// go func() {
				// 	defer f.Close()
				// 	defer pipes[i].Close()
				// 	io.Copy(pipes[i], f)
				// }()
				_, err = io.Copy(pipes[i], f)
				croak(err)
				f.Close()
				pipes[i].Close()
			}

			croak(cmd.Wait())
		}
		log.Printf("Finished processing\n")
	}
}
