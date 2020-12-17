package main

import (
	"io"
	"log"

	rand "github.com/thijzert/speeldoos/lib/properrandom"
	"github.com/thijzert/speeldoos/lib/wavreader"
	speeldoos "github.com/thijzert/speeldoos/pkg"
)

type playlistItem struct {
	Performance speeldoos.Performance
	Wav         wavreader.Reader
}

func play_main(args []string) {
	playlist := make(chan playlistItem)

	l := speeldoos.NewLibrary(Config.LibraryDir)
	l.WAVConf = Config.WAVConf
	l.Refresh()

	pfii := make([]speeldoos.Performance, 0, 50)

	for _, car := range l.Carriers {
		for _, pf := range car.Carrier.Performances {
			pfii = append(pfii, pf)
		}
	}

	if len(pfii) == 0 {
		log.Fatal("No performances found in your library.")
	}

	go func() {
		// Stopgap measure: stop after playing 100 performances.
		for k := 0; k < 100; k++ {
			i := rand.Intn(len(pfii))
			w, err := l.GetWAV(pfii[i])
			if err != nil {
				log.Printf("%v", err)
				continue
			}
			playlist <- playlistItem{Performance: pfii[i], Wav: w}
		}
	}()

	output, err := Config.WAVConf.AudioOutput()
	if err != nil {
		log.Fatal(err)
	}

	for item := range playlist {
		log.Printf("Now playing: %s - %s", item.Performance.Work.Composer.Name, item.Performance.Work.Title[0].Title)
		_, err = io.Copy(output, item.Wav)

		if err != nil {
			log.Fatal(err)
		}
		item.Wav.Close()
	}

	output.Close()
}
