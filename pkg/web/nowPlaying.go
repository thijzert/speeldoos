package web

import (
	"net/http"

	speeldoos "github.com/thijzert/speeldoos/pkg"
)

var NowPlayingHandler nowPlayingHandler

type nowPlayingHandler struct{}

func (nowPlayingHandler) handleNowPlaying(s State, r nowPlayingRequest) (State, nowPlayingResponse, error) {
	return s, nowPlayingResponse{
		NowPlaying: s.NowPlaying,
	}, nil
}

func (nowPlayingHandler) DecodeRequest(r *http.Request) (Request, error) {
	return nowPlayingRequest{}, nil
}

func (h nowPlayingHandler) HandleRequest(s State, r Request) (State, Response, error) {
	req, ok := r.(nowPlayingRequest)
	if !ok {
		return withError(s, errWrongRequestType{})
	}

	return h.handleNowPlaying(s, req)
}

type nowPlayingRequest struct{}

func (nowPlayingRequest) FlaggedAsRequest() {}

type nowPlayingResponse struct {
	NowPlaying speeldoos.Performance
}

func (nowPlayingResponse) FlaggedAsResponse() {}
