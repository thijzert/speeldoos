package web

import (
	"errors"
	"io"
	"log"
	"net/http"

	rand "github.com/thijzert/speeldoos/lib/properrandom"
	"github.com/thijzert/speeldoos/lib/wavreader"
	"github.com/thijzert/speeldoos/lib/wavreader/chunker"
	speeldoos "github.com/thijzert/speeldoos/pkg"
)

type playlistItem struct {
	Performance speeldoos.Performance
	Wav         wavreader.Reader
}

func (s *Server) initAudioStream() error {
	var err error
	playlist := make(chan playlistItem)

	pfii := make([]speeldoos.Performance, 0, 50)

	for _, car := range s.config.Library.Carriers {
		for _, pf := range car.Carrier.Performances {
			pfii = append(pfii, pf)
		}
	}

	if len(pfii) == 0 {
		return errors.New("no performances found in your library")
	}

	go func() {
		// Stopgap measure: stop after playing 10 performances.
		for k := 0; k < 10; k++ {
			i := rand.Intn(len(pfii))
			w, err := s.config.Library.GetWAV(pfii[i])
			if err != nil {
				log.Printf("%v", err)
				continue
			}
			playlist <- playlistItem{Performance: pfii[i], Wav: w}
		}
		close(playlist)
	}()

	s.chunker, err = chunker.NewMP3()
	if err != nil {
		return err
	}

	go func() {
		defer s.chunker.Close()

		for item := range playlist {
			s.nowPlaying = item.Performance
			log.Printf("Now playing: %s - %s", item.Performance.Work.Composer.Name, item.Performance.Work.Title[0].Title)
			defer item.Wav.Close()

			_, err = io.Copy(s.chunker, item.Wav)

			if err != nil {
				log.Print(err)
				return
			}
		}
	}()

	return nil
}

func (s *Server) asyncStreamHandler(w http.ResponseWriter, r *http.Request) {
	cs, err := s.chunker.NewStream()
	if err != nil {
		log.Fatal(err)
	}

	w.Header().Set("Content-Type", "audio/mpeg")
	_, err = io.Copy(w, cs)
	if err != nil {
		log.Print(err)
		return
	}
}
