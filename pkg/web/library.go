package web

import (
	"net/http"
	"sort"
	"strings"

	speeldoos "github.com/thijzert/speeldoos/pkg"
)

var LibraryHandler libraryHandler

type libraryHandler struct{}

func (libraryHandler) handleLibrary(s State, r libraryRequest) (State, libraryResponse, error) {
	var rv libraryResponse

	for _, car := range s.Library.Carriers {
		if car.Error == nil {
			rv.Performances = append(rv.Performances, car.Carrier.Performances...)
		} else {
			rv.FailedCarriers = append(rv.FailedCarriers, car)
		}
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
	Performances   []speeldoos.Performance
	FailedCarriers []speeldoos.ParsedCarrier
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
	} else if a.Work.Year < b.Work.Year {
		return false
	}

	if len(a.Work.OpusNumber) > 0 && len(b.Work.OpusNumber) > 0 {
		if s := strings.Compare(a.Work.OpusNumber[0].Number, b.Work.OpusNumber[0].Number); s != 0 {
			return s < 0
		}
	}

	if a.Year < b.Year {
		return true
	} else if a.Year > b.Year {
		return false
	}

	return false
}

func (libraryResponse) FlaggedAsResponse() {}
