package main

import (
	"log"
	"net"
	"net/http"

	"github.com/thijzert/speeldoos/lib/wavreader/chunker"
	speeldoos "github.com/thijzert/speeldoos/pkg"
	"github.com/thijzert/speeldoos/pkg/web"
)

var server_chunker chunker.Chunker

func server_main(args []string) {
	log.Printf("Starting server...")

	l := speeldoos.NewLibrary(Config.LibraryDir)
	l.WAVConf = Config.WAVConf
	l.Refresh()

	conf := web.ServerConfig{
		Library: l,
	}
	s, err := web.New(conf)
	if err != nil {
		log.Fatal(err)
	}

	ln, err := net.Listen("tcp", "localhost:11884")
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Listening on localhost:11884")

	var srv http.Server
	srv.Handler = s
	log.Fatal(srv.Serve(ln))
}
