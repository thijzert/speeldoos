package web

import (
	"io"
	"log"
	"net/http"

	"github.com/thijzert/speeldoos/lib/wavreader/chunker"
)

var AudioStreamHandler audioStreamHandler

type audioStreamHandler struct{}

func (audioStreamHandler) handleAudioStream(s State, r audioStreamRequest) (State, audioStreamResponse, error) {
	var rv audioStreamResponse

	cs, err := s.Stream.NewStream()
	if err != nil {
		return s, rv, err
	}

	rv.Stream = cs

	return s, rv, nil
}

func (audioStreamHandler) DecodeRequest(r *http.Request) (Request, error) {
	return audioStreamRequest{}, nil
}

func (h audioStreamHandler) HandleRequest(s State, r Request) (State, Response, error) {
	req, ok := r.(audioStreamRequest)
	if !ok {
		return withError(s, errWrongRequestType{})
	}

	return h.handleAudioStream(s, req)
}

type audioStreamRequest struct {
}

func (audioStreamRequest) FlaggedAsRequest() {}

type audioStreamResponse struct {
	Stream chunker.ChunkStream
}

func (audioStreamResponse) FlaggedAsResponse() {}

func (a audioStreamResponse) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "Fri, 1 Apr 2005, 13:00:00 GMT")
	_, err := io.Copy(w, a.Stream)
	if err != nil {
		log.Print(err)
		return
	}
}
