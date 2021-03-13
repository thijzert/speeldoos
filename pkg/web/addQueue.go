package web

import (
	"net/http"

	weberrors "github.com/thijzert/speeldoos/internal/web-plumbing/errors"
	speeldoos "github.com/thijzert/speeldoos/pkg"
)

var AddQueueHandler addQueueHandler

type addQueueHandler struct{}

func (addQueueHandler) handleAddQueue(s State, r addQueueRequest) (State, addQueueResponse, error) {
	_, err := s.Library.GetPerformance(r.PerformanceID)
	if err != nil {
		err = weberrors.WithStatus(err, 404)
	} else {
		s.PlayQueue = append(s.PlayQueue, r.PerformanceID)
		s.PlayQueueDirty = true
	}

	rv := addQueueResponse{
		Queue: make([]speeldoos.Performance, 0, len(s.PlayQueue)),
	}
	for _, pfid := range s.PlayQueue {
		p, er := s.Library.GetPerformance(pfid)
		if er == nil {
			rv.Queue = append(rv.Queue, p)
		}
	}

	return s, rv, err
}

func (addQueueHandler) DecodeRequest(r *http.Request) (Request, error) {
	var err error
	rv := addQueueRequest{}

	pfid := r.PostFormValue("id")
	rv.PerformanceID, err = speeldoos.ParsePerformanceID(pfid)

	if err != nil {
		err = weberrors.WithStatus(err, 400)
	}
	return rv, err
}

func (h addQueueHandler) HandleRequest(s State, r Request) (State, Response, error) {
	req, ok := r.(addQueueRequest)
	if !ok {
		return withError(s, errWrongRequestType{})
	}

	return h.handleAddQueue(s, req)
}

type addQueueRequest struct {
	PerformanceID speeldoos.PerformanceID
}

func (addQueueRequest) FlaggedAsRequest() {}

type addQueueResponse struct {
	Queue []speeldoos.Performance
}

func (addQueueResponse) FlaggedAsResponse() {}
