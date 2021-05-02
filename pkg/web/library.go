package web

import (
	"net/http"
	"sort"

	speeldoos "github.com/thijzert/speeldoos/pkg"
)

var LibraryHandler libraryHandler

type libraryHandler struct{}

func (libraryHandler) handleLibrary(s State, r libraryRequest) (State, libraryResponse, error) {
	var rv libraryResponse
	rv.Performances = make([]speeldoos.Performance, 0, 50)

	for _, car := range s.Library.Carriers {
		rv.Performances = append(rv.Performances, car.Carrier.Performances...)
	}

	sort.Sort(&rv)

	return s, rv, nil
}

func (libraryHandler) DecodeRequest(r *http.Request) (Request, error) {
	return libraryRequest{}, nil
}

func (h libraryHandler) HandleRequest(s State, r Request) (State, Response, error) {
	req, ok := r.(libraryRequest)
	if !ok {
		return withError(s, errWrongRequestType{})
	}

	return h.handleLibrary(s, req)
}

type libraryRequest struct {
}

func (libraryRequest) FlaggedAsRequest() {}

type libraryResponse struct {
	Performances []speeldoos.Performance
}

func (r *libraryResponse) Len() int {
	return len(r.Performances)
}

func (r *libraryResponse) Swap(i, j int) {
	r.Performances[i], r.Performances[j] = r.Performances[j], r.Performances[i]
}

func (r *libraryResponse) Less(i, j int) bool {
	a, b := r.Performances[i], r.Performances[j]

	if a.Work.Year > b.Work.Year {
		return true
	} else if a.Work.Year == b.Work.Year && a.Year < b.Year {
		return true
	}

	return false
}

func (libraryResponse) FlaggedAsResponse() {}
