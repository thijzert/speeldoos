package weberrors

// A UserError is an error that may be shown to end users
type UserError interface {
	error
	Headline() string
	Message() string
}

type userError struct {
	headline string
	message  string
	cause    error
}

// WithMessage wraps an internal error with a user-facing message
func WithMessage(e error, headline, message string) error {
	if e == nil {
		return nil
	}

	return userError{
		headline: headline,
		message:  message,
		cause:    e,
	}
}

func (e userError) Error() string {
	return e.headline + ": " + e.message
}

func (e userError) Unwrap() error {
	return e.cause
}

func (e userError) Headline() string {
	return e.headline
}

func (e userError) Message() string {
	return e.message
}
