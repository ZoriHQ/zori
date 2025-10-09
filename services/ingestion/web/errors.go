package web

type ErrorMissingVisitorID struct{}

func (e *ErrorMissingVisitorID) Error() string {
	return "missing visitor ID"
}

func NewErrorMissingVisitorID() *ErrorMissingVisitorID {
	return &ErrorMissingVisitorID{}
}
