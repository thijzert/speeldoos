package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"

	"github.com/thijzert/speeldoos"
	rand "github.com/thijzert/speeldoos/lib/properrandom"
	"github.com/thijzert/speeldoos/lib/wavreader"
)

func server_main(args []string) {
	ln, err := net.Listen("tcp", ":11884")
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Listening on *:11884")

	mux := http.NewServeMux()
	var srv http.Server

	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/stream.mp3", syncStreamHandler)

	srv.Handler = mux
	log.Fatal(srv.Serve(ln))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	fmt.Fprintf(w, "<!DOCTYPE html>\n")
	fmt.Fprintf(w, "<html><body>\n")
	fmt.Fprintf(w, "<audio controls><source src=\"stream.mp3\" type=\"audio/mpeg\" /></audio>\n")
	fmt.Fprintf(w, "</body></html>\n")
}

func syncStreamHandler(w http.ResponseWriter, r *http.Request) {
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
		// Stopgap measure: stop after playing 10 performances.
		for k := 0; k < 10; k++ {
			i := rand.Intn(len(pfii))
			w, err := l.GetWAV(pfii[i])
			if err != nil {
				log.Printf("%v", err)
				continue
			}
			playlist <- playlistItem{Performance: pfii[i], Wav: w}
		}
		close(playlist)
	}()

	w.Header().Set("Content-Type", "audio/mpeg")

	format := wavreader.StreamFormat{
		Format:   1,
		Channels: 2,
		Rate:     48000,
		Bits:     16,
	}
	output, err := Config.WAVConf.ToMP3(w, format)
	if err != nil {
		log.Fatal(err)
	}
	defer output.Close()

	for item := range playlist {
		log.Printf("Now playing: %s - %s", item.Performance.Work.Composer.Name, item.Performance.Work.Title[0].Title)
		defer item.Wav.Close()

		_, err = io.Copy(output, item.Wav)

		if err != nil {
			log.Print(err)
			return
		}
	}
}
