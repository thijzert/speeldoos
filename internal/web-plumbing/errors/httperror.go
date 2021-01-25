package weberrors

import (
	"errors"
)

type HTTPError interface {
	error
	HTTPStatus() int
}

type httpError struct {
	StatusCode int
	Cause      error
}

func WithStatus(e error, c int) HTTPError {
	if e == nil {
		return nil
	}

	return httpError{
		StatusCode: c,
		Cause:      e,
	}
}

func (e httpError) Error() string {
	return e.Cause.Error()
}

func (e httpError) Unwrap() error {
	return e.Cause
}

func (e httpError) HTTPStatus() int {
	return e.StatusCode
}

func HTTPStatusCode(e error) (statusCode int, cause error) {
	if e == nil {
		return 200, nil
	}

	var httpcode httpError
	if errors.As(e, &httpcode) {
		return httpcode.StatusCode, httpcode.Cause
	}

	var herr HTTPError
	if errors.As(e, &herr) {
		return herr.HTTPStatus(), herr
	}

	return 0, e
}
