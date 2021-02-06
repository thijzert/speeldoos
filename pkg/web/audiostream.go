package web

import (
	"io"
	"log"
	"net/http"

	"github.com/thijzert/speeldoos/lib/wavreader"
	"github.com/thijzert/speeldoos/lib/wavreader/chunker"
)

var MP3StreamHandler mp3StreamHandler
var WAVStreamHandler wavStreamHandler

type mp3StreamHandler struct{}

func (mp3StreamHandler) handleMP3Stream(s State, r audioStreamRequest) (State, audioStreamResponse, error) {
	var rv audioStreamResponse

	cs, err := s.MP3Stream.NewStream()
	if err != nil {
		return s, rv, err
	}

	rv.Type = typeMP3
	rv.Format = s.MP3Stream.Format()
	rv.Stream = cs

	return s, rv, nil
}

func (mp3StreamHandler) DecodeRequest(r *http.Request) (Request, error) {
	return audioStreamRequest{}, nil
}

func (h mp3StreamHandler) HandleRequest(s State, r Request) (State, Response, error) {
	req, ok := r.(audioStreamRequest)
	if !ok {
		return withError(s, errWrongRequestType{})
	}

	return h.handleMP3Stream(s, req)
}

type audioStreamType int

const (
	typeWAV audioStreamType = iota
	typeMP3
)

type audioStreamRequest struct {
}

func (audioStreamRequest) FlaggedAsRequest() {}

type audioStreamResponse struct {
	Type   audioStreamType
	Format wavreader.StreamFormat
	Stream chunker.ChunkStream
}

func (audioStreamResponse) FlaggedAsResponse() {}

func (a audioStreamResponse) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var tgt io.Writer = w

	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "Fri, 1 Apr 2005, 13:00:00 GMT")

	if a.Type == typeMP3 {
		w.Header().Set("Content-Type", "audio/mpeg")
	} else if a.Type == typeWAV {
		w.Header().Set("Content-Type", "audio/wav")
		ww := wavreader.NewWriter(w, a.Format)
		ww.Init(0)
		tgt = ww
	} else {
		w.Header().Set("Content-Type", "application/octet-steam")
	}

	_, err := io.Copy(tgt, a.Stream)

	if err != nil {
		log.Print(err)
		return
	}
}

type wavStreamHandler struct{}

func (wavStreamHandler) handleWAVStream(s State, r audioStreamRequest) (State, audioStreamResponse, error) {
	var rv audioStreamResponse

	cs, err := s.RawStream.NewStream()
	if err != nil {
		return s, rv, err
	}

	rv.Type = typeWAV
	rv.Format = s.RawStream.Format()
	rv.Stream = cs

	return s, rv, nil
}

func (wavStreamHandler) DecodeRequest(r *http.Request) (Request, error) {
	return audioStreamRequest{}, nil
}

func (h wavStreamHandler) HandleRequest(s State, r Request) (State, Response, error) {
	req, ok := r.(audioStreamRequest)
	if !ok {
		return withError(s, errWrongRequestType{})
	}

	return h.handleWAVStream(s, req)
}
