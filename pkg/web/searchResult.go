package web

import (
	"net/http"

	weberrors "github.com/thijzert/speeldoos/internal/web-plumbing/errors"
	"github.com/thijzert/speeldoos/pkg/search"
)

var SearchResultHandler searchResultHandler

type searchResultHandler struct{}

func (searchResultHandler) handleSearchResult(s State, r searchResultRequest) (State, searchResultResponse, error) {
	var rv searchResultResponse

	// TODO: the actual config may not always be the zero value
	var conf search.Config
	q, err := conf.Compile(r.Query)
	if err != nil {
		err = weberrors.WithStatus(err, 400)
		return s, rv, err
	}

	rv.Results = q.Search(s.Library)

	return s, rv, nil
}

type searchResultRequest struct {
	Query string
}

type searchResultResponse struct {
	Results []search.Result
}

func (searchResultHandler) DecodeRequest(r *http.Request) (Request, error) {
	var rv searchResultRequest

	r.ParseForm()
	rv.Query = r.Form.Get("q")

	return rv, nil
}

func (h searchResultHandler) HandleRequest(s State, r Request) (State, Response, error) {
	req, ok := r.(searchResultRequest)
	if !ok {
		return withError(s, errWrongRequestType{})
	}

	return h.handleSearchResult(s, req)
}

func (searchResultRequest) FlaggedAsRequest() {}

func (searchResultResponse) FlaggedAsResponse() {}
