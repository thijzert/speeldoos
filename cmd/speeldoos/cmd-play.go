package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"

	"github.com/thijzert/speeldoos"
	"github.com/thijzert/speeldoos/lib/wavreader"
)

func play_main(args []string) {
	l := speeldoos.NewLibrary(Config.LibraryDir)
	l.Refresh()

	var s *wavreader.Reader
	var err error = fmt.Errorf("no performances found")
firstPerformance:
	for _, car := range l.Carriers {
		for _, pf := range car.Carrier.Performances {
			s, err = l.GetWAV(pf)
			if err == nil {
				break firstPerformance
			}
		}
	}
	if err != nil {
		log.Fatal(err)
	}

	mpl := exec.Command(Config.Tools.MPlayer,
		"-really-quiet",
		"-noconsolecontrols", "-nomouseinput", "-nolirc",
		"-cache", "1024",
		"-rawaudio", fmt.Sprintf("rate=%d:channels=%d:samplesize=%d", s.SampleRate, s.Channels, (s.BitsPerSample+7)/8),
		"-demuxer", "rawaudio",
		"-")

	mpl.Stdout = os.Stdout
	mpl.Stderr = os.Stderr

	output, err := mpl.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}

	mpl.Start()
	defer mpl.Wait()

	_, err = io.Copy(output, s)
	if err != nil {
		log.Fatal(err)
	}

	output.Close()
}
