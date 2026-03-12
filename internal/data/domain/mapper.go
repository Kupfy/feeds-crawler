package domain

import (
	"encoding/json"
	"errors"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var httpCodeMap = map[string]int{
	"NOT_FOUND":     http.StatusNotFound,
	"UNAUTHORIZED":  http.StatusUnauthorized,
	"INVALID_INPUT": http.StatusBadRequest,
	"CONFLICT":      http.StatusConflict,
}

var grpcCodeMap = map[string]codes.Code{
	"NOT_FOUND":     codes.NotFound,
	"UNAUTHORIZED":  codes.Unauthenticated,
	"INVALID_INPUT": codes.InvalidArgument,
	"CONFLICT":      codes.AlreadyExists,
}

func ToHTTPError(w http.ResponseWriter, err error) {
	var de *DomainError
	if errors.As(err, &de) {
		sts, ok := httpCodeMap[de.Code]
		if !ok {
			sts = http.StatusInternalServerError
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(sts)
		_ = json.NewEncoder(w).Encode(de.ToExternalError())
		return
	}

	// Unknown error
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	_ = json.NewEncoder(w).Encode(ExternalError{
		Code:    "INTERNAL",
		Message: "an unexpected error occurred",
	})
}

func ToGRPCError(err error) error {
	var de *DomainError
	if errors.As(err, &de) {
		c := grpcCodeMap[de.Code]
		if c == 0 {
			c = codes.Internal
		}
		return status.Error(c, de.Message)
	}
	return status.Error(codes.Internal, "internal error")
}
