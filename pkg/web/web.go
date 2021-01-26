package web

import (
	"net/http"

	"github.com/thijzert/speeldoos/lib/wavreader/chunker"
	speeldoos "github.com/thijzert/speeldoos/pkg"
)

// The State struct represents the current state of the world
type State struct {
	Library    *speeldoos.Library
	NowPlaying speeldoos.Performance
	Stream     chunker.Chunker
	Buffers    struct {
		MP3Stream chunker.Statuser
	}
}

var (
	// ErrParser is thrown when a request object is of the wrong type
	ErrParser error = errParse{}
)

type errParse struct{}

func (errParse) Error() string {
	return "parse error while decoding request"
}

// A Request flags any request type
type Request interface {
	FlaggedAsRequest()
}

// A Response flags any response type
type Response interface {
	FlaggedAsResponse()
}

// A Handler handles requests
type Handler interface {
	// DecodeRequest turns a HTTP request into a domain-specific request type
	DecodeRequest(*http.Request) (Request, error)

	// A RequestHandler is a monadic definition of a request handler. The inputs
	// are the current state of the world, and a handler-specific request type,
	// and the output is the new state of the world (which may or may not be the
	// same), a handler-specific response type, and/or an error.
	HandleRequest(State, Request) (State, Response, error)
}
