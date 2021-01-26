package plumbing

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/thijzert/speeldoos/pkg/web"
)

type jsonHandler struct {
	Server  *Server
	Handler web.Handler
}

// JSONFunc creates a HTTP handler that outputs JSON
func (s *Server) JSONFunc(handler web.Handler) http.Handler {
	return jsonHandler{
		Server:  s,
		Handler: handler,
	}
}

func (h jsonHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	req, err := h.Handler.DecodeRequest(r)
	if err != nil {
		h.Error(w, r, err)
		return
	}

	state := h.Server.getState()
	newState, resp, err := h.Handler.HandleRequest(state, req)
	if err != nil {
		h.Error(w, r, err)
		return
	}

	err = h.Server.setState(newState)
	if err != nil {
		h.Error(w, r, err)
		return
	}

	// Alternative path: this response can write its own headers and response body
	if h, ok := resp.(http.Handler); ok {
		h.ServeHTTP(w, r)
		return
	}

	w.Header()["Content-Type"] = []string{"application/json"}
	w.Header()["X-Content-Type-Options"] = []string{"nosniff"}

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

	w.Header()["Content-Type"] = []string{"application/json"}
	w.Header()["X-Content-Type-Options"] = []string{"nosniff"}

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
