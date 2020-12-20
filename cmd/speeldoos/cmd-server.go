package main

import (
	"context"
	"log"
	"net"
	"net/http"

	"github.com/thijzert/speeldoos/lib/wavreader"
	"github.com/thijzert/speeldoos/lib/wavreader/chunker"
	speeldoos "github.com/thijzert/speeldoos/pkg"
	"github.com/thijzert/speeldoos/pkg/web"
)

var server_chunker chunker.Chunker

func server_main(args []string) {
	log.Printf("Starting server...")

	mc := chunker.MP3ChunkConfig{
		Context: context.Background(),
		Audio: wavreader.Config{
			LamePath:       Config.Tools.Lame,
			MaxBitrate:     Config.Server.Encoder.MaxBitrate,
			VBRQuality:     Config.Server.Encoder.VBRQuality,
			PlaybackFormat: wavreader.DAT,
		},
	}

	l := speeldoos.NewLibrary(Config.LibraryDir)
	l.WAVConf = Config.WAVConf
	l.Refresh()

	conf := web.ServerConfig{
		Library:      l,
		StreamConfig: mc,
	}
	s, err := web.New(conf)
	if err != nil {
		log.Fatal(err)
	}

	ln, err := net.Listen("tcp", Config.Server.Listen)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Listening on %s", Config.Server.Listen)

	var srv http.Server
	srv.Handler = s
	log.Fatal(srv.Serve(ln))
}
