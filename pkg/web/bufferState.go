package web

import (
	"net/http"

	"github.com/thijzert/speeldoos/lib/wavreader/chunker"
)

var BufferStatusHandler bufferStatusHandler

type bufferStatusHandler struct{}

func (bufferStatusHandler) handleBufferStatus(s State, r bufferStatusRequest) (State, bufferStatusResponse, error) {
	var rv bufferStatusResponse

	if s.Buffers.MP3Stream != nil {
		mp3Stream := s.Buffers.MP3Stream.BufferStatus()
		if !mp3Stream.Tmin.IsZero() {
			rv.MP3Stream = &mp3Stream
		}
	}
	if s.Buffers.Scheduler != nil {
		sch := s.Buffers.Scheduler.BufferStatus()
		if !sch.Tmin.IsZero() {
			rv.Scheduler = &sch
		}
	}

	return s, rv, nil
}

func (bufferStatusHandler) DecodeRequest(r *http.Request) (Request, error) {
	return bufferStatusRequest{}, nil
}

func (h bufferStatusHandler) HandleRequest(s State, r Request) (State, Response, error) {
	req, ok := r.(bufferStatusRequest)
	if !ok {
		return withError(s, errWrongRequestType{})
	}

	return h.handleBufferStatus(s, req)
}

type bufferStatusRequest struct {
}

func (bufferStatusRequest) FlaggedAsRequest() {}

type bufferStatusResponse struct {
	MP3Stream *chunker.BufferStatus
	Scheduler *chunker.BufferStatus
}

func (bufferStatusResponse) FlaggedAsResponse() {}
