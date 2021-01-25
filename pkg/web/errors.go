package web

import (
	"errors"
	"fmt"

	weberrors "github.com/thijzert/speeldoos/internal/web-plumbing/errors"
)

type errorResponse struct {
	Error error
}

func (errorResponse) FlaggedAsResponse() {}

func withError(s State, e error) (State, Response, error) {
	return s, errorResponse{e}, e
}

type errWrongRequestType struct{}

func (errWrongRequestType) Error() string {
	return "wrong request type"
}

func (errWrongRequestType) HTTPCode() int {
	return 400
}

type errRedirect struct {
	URL string
}

func (errRedirect) Error() string {
	return "you are being redirected to another page"
}

func (errRedirect) Headline() string {
	return "Redirecting..."
}

func (e errRedirect) Message() string {
	return fmt.Sprintf("You are being redirected to the address '%s'", e.URL)
}

func (errRedirect) HTTPCode() int {
	return 302
}

func (e errRedirect) RedirectLocation() string {
	return e.URL
}

func errForbidden(headline, message string) error {
	rv := errors.New("access denied")
	rv = weberrors.WithStatus(rv, 403)

	if headline == "" {
		headline = "Access Denied"
	}
	if message == "" {
		message = "You don't have permission to access this resource"
	}

	rv = weberrors.WithMessage(rv, headline, message)
	return rv
}

func errNotFound(headline, message string) error {
	rv := errors.New("not found")
	rv = weberrors.WithStatus(rv, 404)

	if headline == "" {
		headline = "Not Found"
	}
	if message == "" {
		message = "The document or resource you requested could not be found"
	}

	rv = weberrors.WithMessage(rv, headline, message)
	return rv
}
