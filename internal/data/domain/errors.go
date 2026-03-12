package domain

type DomainError struct {
	Code    string
	Message string
	Cause   error
}

type ExternalError struct {
	Code    string
	Message string
}

func (e *DomainError) Error() string { return e.Message }
func (e *DomainError) Unwrap() error { return e.Cause }
func (e *DomainError) Is(target error) bool {
	t, ok := target.(*DomainError)
	return ok && t.Code == e.Code
}

func (e *DomainError) ToExternalError() ExternalError {
	return ExternalError{Code: e.Code, Message: e.Message}
}

var ErrNotFound = &DomainError{Code: "NOT_FOUND", Message: "resource not found"}

// Wrap lets callers attach a cause without leaking the sentinel's pointer
func Wrap(sentinel *DomainError, cause error) *DomainError {
	return &DomainError{
		Code:    sentinel.Code,
		Message: sentinel.Message,
		Cause:   cause,
	}
}
