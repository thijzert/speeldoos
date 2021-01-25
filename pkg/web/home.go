package web

import "net/http"

var HomeHandler homeHandler

type homeHandler struct{}

func (homeHandler) handleHome(s State, r homeRequest) (State, homeResponse, error) {
	return s, homeResponse{}, nil
}

func (homeHandler) DecodeRequest(r *http.Request) (Request, error) {
	return homeRequest{}, nil
}

func (h homeHandler) HandleRequest(s State, r Request) (State, Response, error) {
	req, ok := r.(homeRequest)
	if !ok {
		return withError(s, errWrongRequestType{})
	}

	return h.handleHome(s, req)
}

type homeRequest struct {
	Path string
}

func (homeRequest) FlaggedAsRequest() {}

type homeResponse struct{}

func (homeResponse) FlaggedAsResponse() {}
