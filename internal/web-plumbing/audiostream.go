package plumbing

import (
	"io"
	"log"
	"time"

	"github.com/thijzert/speeldoos/lib/wavreader"
	"github.com/thijzert/speeldoos/lib/wavreader/chunker"
	speeldoos "github.com/thijzert/speeldoos/pkg"
)

type playlistItem struct {
	Performance speeldoos.Performance
	Wav         wavreader.Reader
}

func (s *Server) initAudioStream() error {
	wc := chunker.WAVChunkConfig{
		StreamFormat: s.config.StreamConfig.Audio.PlaybackFormat,
	}

	var err error

	s.scheduler, err = s.config.Library.NewScheduler(wc)
	if err != nil {
		log.Fatal(err)
	}

	go s.scheduler.Run(s.context)

	s.chunker, err = s.config.StreamConfig.NewMP3()
	if err != nil {
		return err
	}

	stream, err := s.scheduler.AudioStream.NewStreamWithOffset(25 * time.Second)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		io.Copy(s.chunker, stream)
		s.chunker.Close()
	}()

	return nil
}
