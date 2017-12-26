package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/thijzert/speeldoos"
	"github.com/thijzert/speeldoos/lib/wavreader"
)

func play_main(args []string) {
	l := speeldoos.NewLibrary(Config.LibraryDir)
	l.WAVConf = Config.WAVConf
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

	output, err := Config.WAVConf.AudioOutput()
	if err != nil {
		log.Fatal(err)
	}

	if Config.Play.TapFilename == "" {
		_, err = io.Copy(output, s)
	} else {
		aud, err := wavreader.Convert(s, Config.WAVConf.PlaybackFormat)
		if err != nil {
			log.Fatal(err)
		}

		var tap *os.File
		tap, err = os.Create(Config.Play.TapFilename)
		if err == nil {
			tapOut := wavreader.NewWriter(tap, Config.WAVConf.PlaybackFormat)
			defer tapOut.Close()

			out := io.MultiWriter(tapOut, output)
			_, err = io.Copy(out, aud)
		}
	}

	if err != nil {
		log.Fatal(err)
	}

	output.Close()
}
