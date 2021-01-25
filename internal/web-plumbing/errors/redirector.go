package weberrors

type Redirector interface {
	error
	RedirectLocation() string
}
