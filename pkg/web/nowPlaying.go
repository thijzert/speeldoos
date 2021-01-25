package web

import (
	"net/http"

	speeldoos "github.com/thijzert/speeldoos/pkg"
)

// NowPlayingDecoder decodes a request for the home page
func NowPlayingDecoder(r *http.Request) (Request, error) {
	return emptyRequest{}, nil
}

type nowPlayingResponse struct {
	NowPlaying speeldoos.Performance
}

func (nowPlayingResponse) FlaggedAsResponse() {}

// NowPlayingHandler handles requests for the home page
func NowPlayingHandler(s State, _ Request) (State, Response, error) {
	return s, nowPlayingResponse{
		NowPlaying: s.NowPlaying,
	}, nil
}
