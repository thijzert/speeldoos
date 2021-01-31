package pkg

import (
	"context"
	"io"
	"log"

	rand "github.com/thijzert/speeldoos/lib/properrandom"
	"github.com/thijzert/speeldoos/lib/wavreader/chunker"
)

type Scheduler struct {
	Library     *Library
	AudioStream chunker.Chunker
}

func (l *Library) NewScheduler(wc chunker.WAVChunkConfig) (*Scheduler, error) {
	var err error
	rv := &Scheduler{
		Library: l,
	}

	rv.AudioStream, err = wc.New()
	if err != nil {
		return nil, err
	}

	return rv, nil
}

func (s *Scheduler) Run(ctx context.Context) {
	for ctx.Err() == nil {
		performance := s.NextPerformance()

		w, err := s.Library.GetWAV(performance)
		if err != nil {
			log.Printf("%v", err)
			continue
		}

		s.AudioStream.SetAssociatedData(performance)
		log.Printf("Queued: %s - %s", performance.Work.Composer.Name, performance.Work.Title[0].Title)

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
		// FIXME: handle this error properly
		log.Fatal("No performances found in your library.")
	}

	i := rand.Intn(len(pfii))

	return pfii[i]
}
