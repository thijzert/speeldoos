package pkg

import (
	"context"
	"io"
	"log"

	rand "github.com/thijzert/speeldoos/lib/properrandom"
	"github.com/thijzert/speeldoos/lib/wavreader/chunker"
)

type Scheduler struct {
	Context     context.Context
	Library     *Library
	AudioStream chunker.Chunker
}

func (l *Library) NewScheduler(ctx context.Context, wc chunker.WAVChunkConfig) (*Scheduler, error) {
	var err error
	rv := &Scheduler{
		Context: ctx,
		Library: l,
	}

	rv.AudioStream, err = wc.New()
	if err != nil {
		return nil, err
	}

	go rv.run()

	return rv, nil
}

func (s *Scheduler) run() {
	for s.Context.Err() == nil {
		performance := s.NextPerformance()

		w, err := s.Library.GetWAV(performance)
		if err != nil {
			log.Printf("%v", err)
			continue
		}

		log.Printf("Now playing: %s - %s", performance.Work.Composer.Name, performance.Work.Title[0].Title)
		_, err = io.Copy(s.AudioStream, w)

		if err != nil {
			log.Fatal(err)
		}
		w.Close()
	}
}

func (s *Scheduler) NextPerformance() Performance {
	pfii := make([]Performance, 0, 50)

	for _, car := range s.Library.Carriers {
		for _, pf := range car.Carrier.Performances {
			pfii = append(pfii, pf)
		}
	}

	if len(pfii) == 0 {
		log.Fatal("No performances found in your library.")
	}

	log.Printf("found %d performances in your library", len(pfii))
	i := rand.Intn(len(pfii))

	return pfii[i]
}
