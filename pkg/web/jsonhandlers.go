package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/thijzert/speeldoos/pkg/web/handlers"
)

type jsonHandler struct {
	Server         *Server
	TemplateName   string
	RequestDecoder handlers.RequestDecoder
	Handler        handlers.RequestHandler
}

// JSONFunc creates a HTTP handler that outputs JSON
func (s *Server) JSONFunc(handler handlers.RequestHandler, decoder handlers.RequestDecoder, templateName string) http.Handler {
	return jsonHandler{
		Server:         s,
		TemplateName:   templateName,
		RequestDecoder: decoder,
		Handler:        handler,
	}
}

func (h jsonHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header()["Content-Type"] = []string{"application/json"}
	w.Header()["X-Content-Type-Options"] = []string{"nosniff"}

	req, err := h.RequestDecoder(r)
	if err != nil {
		h.Error(w, r, err)
		return
	}

	state := h.Server.getState()
	newState, resp, err := h.Handler(state, req)
	if err != nil {
		h.Error(w, r, err)
		return
	}

	err = h.Server.setState(newState)
	if err != nil {
		h.Error(w, r, err)
		return
	}

	var b bytes.Buffer
	e := json.NewEncoder(&b)
	err = e.Encode(resp)
	if err != nil {
		h.Error(w, r, err)
		return
	}

	io.Copy(w, &b)
}

func (jsonHandler) Error(w http.ResponseWriter, r *http.Request, err error) {
	// TODO: we may need to set a different status entirely
	w.WriteHeader(500)
	errorResponse := struct {
		errorCode    int
		errorMessage string
	}{
		500, // TODO: error codes
		err.Error(),
	}

	var b bytes.Buffer
	e := json.NewEncoder(&b)
	err = e.Encode(errorResponse)
	if err != nil {
		fmt.Fprintf(w, "{errorCode: 500, errorMessage: \"I give up.\"}")
	} else {
		io.Copy(w, &b)
	}
}
