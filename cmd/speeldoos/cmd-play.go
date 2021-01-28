package main

import (
	"context"
	"io"
	"log"

	"github.com/thijzert/speeldoos/lib/wavreader/chunker"
	speeldoos "github.com/thijzert/speeldoos/pkg"
)

func play_main(args []string) {

	l := speeldoos.NewLibrary(Config.LibraryDir)
	l.WAVConf = Config.WAVConf
	l.Refresh()

	wc := chunker.WAVChunkConfig{
		StreamFormat: Config.WAVConf.PlaybackFormat,
	}

	sch, err := l.NewScheduler(context.Background(), wc)
	if err != nil {
		log.Fatal(err)
	}

	output, err := Config.WAVConf.AudioOutput()
	if err != nil {
		log.Fatal(err)
	}
	defer output.Close()

	stream, err := sch.AudioStream.NewStream()
	if err != nil {
		log.Fatal(err)
	}

	io.Copy(output, stream)
}
