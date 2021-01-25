package web

import "net/http"

var StatusHandler statusHandler

type statusHandler struct{}

func (statusHandler) handleStatus(s State, r statusRequest) (State, statusResponse, error) {
	return s, statusResponse{}, nil
}

func (statusHandler) DecodeRequest(r *http.Request) (Request, error) {
	return statusRequest{}, nil
}

func (h statusHandler) HandleRequest(s State, r Request) (State, Response, error) {
	req, ok := r.(statusRequest)
	if !ok {
		return withError(s, errWrongRequestType{})
	}

	return h.handleStatus(s, req)
}

type statusRequest struct {
}

func (statusRequest) FlaggedAsRequest() {}

type statusResponse struct{}

func (statusResponse) FlaggedAsResponse() {}
