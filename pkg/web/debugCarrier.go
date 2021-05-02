package web

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	weberrors "github.com/thijzert/speeldoos/internal/web-plumbing/errors"
	speeldoos "github.com/thijzert/speeldoos/pkg"
)

var DebugCarrierHandler debugCarrierHandler

type debugCarrierHandler struct{}

func (debugCarrierHandler) handleDebugCarrier(s State, r debugCarrierRequest) (State, debugCarrierResponse, error) {
	var rv debugCarrierResponse

	str := fmt.Sprintf("carrier '%s' not found\n", r.CarrierID)
	for _, pc := range s.Library.Carriers {
		str += "\n *  " + pc.Carrier.ID
	}

	rv.ParsedCarrier.Error = fmt.Errorf("carrier '%s' not found", r.CarrierID)
	rv.ParsedCarrier.Error = fmt.Errorf("%s", str)
	rv.ParsedCarrier.Error = weberrors.WithStatus(rv.ParsedCarrier.Error, 404)

	for _, pc := range s.Library.Carriers {
		if pc.Carrier != nil && pc.Carrier.ID == r.CarrierID {
			rv.ParsedCarrier = pc
		}
	}

	return s, rv, nil
}

func (debugCarrierHandler) DecodeRequest(r *http.Request) (Request, error) {
	var rv debugCarrierRequest
	var err error

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) > 3 {
		rv.CarrierID, err = url.QueryUnescape(parts[3])
	}

	return rv, err
}

func (h debugCarrierHandler) HandleRequest(s State, r Request) (State, Response, error) {
	req, ok := r.(debugCarrierRequest)
	if !ok {
		return withError(s, errWrongRequestType{})
	}

	return h.handleDebugCarrier(s, req)
}

type debugCarrierRequest struct {
	CarrierID string
}

func (debugCarrierRequest) FlaggedAsRequest() {}

type debugCarrierResponse struct {
	ParsedCarrier speeldoos.ParsedCarrier
}

func (debugCarrierResponse) FlaggedAsResponse() {}

func (d debugCarrierResponse) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/xml")
	if d.ParsedCarrier.Error != nil {
		sc, _ := weberrors.HTTPStatusCode(d.ParsedCarrier.Error)
		if sc >= 200 && sc < 600 {
			w.WriteHeader(sc)
		}
	}

	d.writeError(w, d.ParsedCarrier.Error)
	var b bytes.Buffer
	if d.ParsedCarrier.Carrier != nil {
		xm := xml.NewEncoder(&b)
		xm.Indent("", "	")
		d.writeError(w, xm.Encode(d.ParsedCarrier.Carrier))
	}
	if b.Len() > 0 {
		io.Copy(w, &b)
	} else {
		fmt.Fprintf(w, "<error>No XML output could be generated</error>")
	}
}

func (d debugCarrierResponse) writeError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	e := fmt.Sprintf("%v", err)
	e = strings.ReplaceAll(e, "-->", "-\\->")
	e = "    " + strings.ReplaceAll(e, "\n", "\n    ")

	fmt.Fprintf(w, "<!--\n\n%s\n\n-->\n", e)
}
