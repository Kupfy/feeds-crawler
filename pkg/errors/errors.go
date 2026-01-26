package errors

import (
	"fmt"
	"net/http"
)

// APIError represents a structured API error
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"-"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("[%d] %s: %s", e.Status, e.Code, e.Message)
}

// Common error constructors
func NewBadRequestError(message string) *APIError {
	return &APIError{
		Code:    "BAD_REQUEST",
		Message: message,
		Status:  http.StatusBadRequest,
	}
}

func NewUnauthorizedError(message string) *APIError {
	return &APIError{
		Code:    "UNAUTHORIZED",
		Message: message,
		Status:  http.StatusUnauthorized,
	}
}

func NewForbiddenError(message string) *APIError {
	return &APIError{
		Code:    "FORBIDDEN",
		Message: message,
		Status:  http.StatusForbidden,
	}
}

func NewNotFoundError(message string) *APIError {
	return &APIError{
		Code:    "NOT_FOUND",
		Message: message,
		Status:  http.StatusNotFound,
	}
}

func NewInternalError(message string) *APIError {
	return &APIError{
		Code:    "INTERNAL_ERROR",
		Message: message,
		Status:  http.StatusInternalServerError,
	}
}

func NewValidationError(message string) *APIError {
	return &APIError{
		Code:    "VALIDATION_ERROR",
		Message: message,
		Status:  http.StatusUnprocessableEntity,
	}
}

// ToAPIError converts an error to an APIError
// This is the foundation for future error transformation logic
func ToAPIError(err error) *APIError {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr
	}

	// Default to internal server error for unknown errors
	return &APIError{
		Code:    "INTERNAL_ERROR",
		Message: "An unexpected error occurred",
		Status:  http.StatusInternalServerError,
	}
}
