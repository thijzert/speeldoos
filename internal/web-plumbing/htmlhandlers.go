package plumbing

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"path"
	"strings"

	weberrors "github.com/thijzert/speeldoos/internal/web-plumbing/errors"
	speeldoos "github.com/thijzert/speeldoos/pkg"
	"github.com/thijzert/speeldoos/pkg/search"
	"github.com/thijzert/speeldoos/pkg/web"
)

type htmlHandler struct {
	Server       *Server
	TemplateName string
	Handler      web.Handler
}

// HTMLFunc creates a HTTP handler that outputs HTML
func (s *Server) HTMLFunc(handler web.Handler, templateName string) http.Handler {
	return htmlHandler{
		Server:       s,
		TemplateName: templateName,
		Handler:      handler,
	}
}

var csp string

func init() {
}

func (h htmlHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	req, err := h.Handler.DecodeRequest(r)
	if err != nil {
		h.Error(w, r, err)
		return
	}

	tpl, err := h.Server.getTemplate(h.TemplateName)
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

	w.Header()["Content-Type"] = []string{"text/html; charset=UTF-8"}

	csp := ""
	csp += "default-src 'self' blob: data: ; "
	csp += "script-src 'self' blob: ; "
	csp += "style-src 'self' data: 'unsafe-inline'; "
	csp += "img-src 'self' blob: data: ; "
	csp += "connect-src 'self' blob: data: ; "
	csp += "frame-src 'none' ; "
	csp += "frame-ancestors 'none'; "
	csp += "form-action 'self'; "
	w.Header()["Content-Security-Policy"] = []string{csp}
	w.Header()["X-Frame-Options"] = []string{"deny"}
	w.Header()["X-XSS-Protection"] = []string{"1; mode=block"}
	w.Header()["Referrer-Policy"] = []string{"strict-origin-when-cross-origin"}
	w.Header()["X-Content-Type-Options"] = []string{"nosniff"}

	tpData := struct {
		Version       string
		AppRoot       string
		AssetLocation string
		PageCSS       string
		Request       web.Request
		Response      web.Response
	}{
		Version:       speeldoos.PackageVersion,
		AppRoot:       h.appRoot(r),
		AssetLocation: h.appRoot(r) + "assets",
		Request:       req,
		Response:      resp,
	}

	cssTemplate := h.TemplateName
	if len(cssTemplate) > 5 && cssTemplate[0:5] == "full/" {
		cssTemplate = cssTemplate[5:]
	}

	if _, err := getAsset(path.Join("dist", "css", "pages", cssTemplate+".css")); err == nil {
		tpData.PageCSS = cssTemplate
	}

	var b bytes.Buffer
	err = tpl.ExecuteTemplate(&b, "basePage", tpData)
	if err != nil {
		h.Error(w, r, err)
		return
	}
	io.Copy(w, &b)
}

func (s *Server) getTemplate(name string) (*template.Template, error) {
	if s.parsedTemplates == nil {
		s.parsedTemplates = make(map[string]*template.Template)
	}

	if assetsEmbedded {
		if tp, ok := s.parsedTemplates[name]; ok {
			return tp, nil
		}
	}

	var tp *template.Template

	b, err := getAsset(path.Join("templates", name+".html"))
	if err != nil {
		return nil, err
	}

	funcs := template.FuncMap{
		"add":       templateAddi,
		"mul":       templateMuli,
		"addf":      templateAddf,
		"mulf":      templateMulf,
		"highlight": highlightSearch,
	}

	if name == "full/basePage" {
		tp, err = template.New("basePage").Funcs(funcs).Parse(string(b))
		if err != nil {
			return nil, err
		}
	} else if len(name) > 5 && name[:5] == "full/" {
		basePage, err := s.getTemplate("full/basePage")
		if err != nil {
			return nil, err
		}

		tp, err = basePage.Clone()
		if err != nil {
			return nil, err
		}

		_, err = tp.Parse(string(b))
		if err != nil {
			return nil, err
		}
	} else {
		tp, err = template.New("basePage").Funcs(funcs).Parse(string(b))
		if err != nil {
			return nil, err
		}
	}

	s.parsedTemplates[name] = tp
	return tp, nil
}

func templateMuli(a, b int) int {
	return a * b
}
func templateAddi(a, b int) int {
	return a + b
}
func templateMulf(a, b float64) float64 {
	return a * b
}
func templateAddf(a, b float64) float64 {
	return a + b
}

type highlightStronger struct {
	Content string
}

func (h *highlightStronger) WriteMatched(str string) {
	h.Content += "<strong>" + template.HTMLEscapeString(str) + "</strong>"
}
func (h *highlightStronger) WriteUnmatched(str string) {
	h.Content += template.HTMLEscapeString(str)
}

func highlightSearch(str fmt.Stringer) template.HTML {
	if ms, ok := str.(search.MatchedString); ok {
		rv := &highlightStronger{}
		ms.Export(rv)
		return template.HTML(rv.Content)
	}

	rv := template.HTMLEscapeString(str.String())
	return template.HTML(rv)
}

// appRoot finds the relative path to the application root
func (htmlHandler) appRoot(r *http.Request) string {
	// Find the relative path for the application root by counting the number of slashes in the relative URL
	c := strings.Count(r.URL.Path, "/") - 1
	if c == 0 {
		return "./"
	}
	return strings.Repeat("../", c)
}

func (h htmlHandler) Error(w http.ResponseWriter, r *http.Request, err error) {
	st, _ := weberrors.HTTPStatusCode(err)

	if redir, ok := err.(weberrors.Redirector); ok {
		if st == 0 {
			st = 302
		}
		h.Server.redirect(w, r, h.appRoot(r)+redir.RedirectLocation(), st)
		return
	}

	if st == 0 {
		st = 500
	}

	w.WriteHeader(st)
	fmt.Fprintf(w, "Error: %s", err)
}
