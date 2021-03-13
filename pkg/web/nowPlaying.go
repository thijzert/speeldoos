package web

import (
	"net/http"

	speeldoos "github.com/thijzert/speeldoos/pkg"
)

var NowPlayingHandler nowPlayingHandler

type nowPlayingHandler struct{}

func (nowPlayingHandler) handleNowPlaying(s State, r nowPlayingRequest) (State, nowPlayingResponse, error) {
	rv := nowPlayingResponse{
		NowPlaying: s.NowPlaying,
	}

	if s.Buffers.Scheduler != nil {
		sch := s.Buffers.Scheduler.BufferStatus()
		if sch.AllAssociatedData != nil {
			for _, td := range sch.AllAssociatedData {
				if td.T < 0 || td.Data == nil {
					continue
				}
				if pf, ok := td.Data.(speeldoos.Performance); ok {
					rv.UpNext = append(rv.UpNext, pf)
				}
			}
		}
	}

	if s.PlayQueue != nil {
		for _, pfid := range s.PlayQueue {
			pf, err := s.Library.GetPerformance(pfid)
			if err == nil {
				rv.UpNext = append(rv.UpNext, pf)
			}
		}
	}

	return s, rv, nil
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
	UpNext     []speeldoos.Performance
}

func (nowPlayingResponse) FlaggedAsResponse() {}
