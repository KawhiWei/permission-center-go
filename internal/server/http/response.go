package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/luck/permission-center-go/internal/biz"
)

func writeJSON(w http.ResponseWriter, status int, value any) {
	writeRawJSON(w, status, map[string]any{
		"success":      true,
		"errorCode":    nil,
		"errorMessage": nil,
		"result":       value,
	})
}

func writeRawJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, biz.ErrInvalidArgument):
		status = http.StatusBadRequest
	case errors.Is(err, biz.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, biz.ErrAlreadyExists), errors.Is(err, biz.ErrConflict):
		status = http.StatusConflict
	case errors.Is(err, biz.ErrNotImplemented):
		status = http.StatusNotImplemented
	}
	writeRawJSON(w, status, map[string]any{
		"success":      false,
		"errorCode":    apiErrorCode(err),
		"errorMessage": err.Error(),
		"result":       nil,
	})
}

func apiErrorCode(err error) string {
	switch {
	case errors.Is(err, biz.ErrInvalidArgument):
		return "INVALID_ARGUMENT"
	case errors.Is(err, biz.ErrNotFound):
		return "NOT_FOUND"
	case errors.Is(err, biz.ErrAlreadyExists):
		return "ALREADY_EXISTS"
	case errors.Is(err, biz.ErrConflict):
		return "CONFLICT"
	case errors.Is(err, biz.ErrNotImplemented):
		return "NOT_IMPLEMENTED"
	default:
		return "INTERNAL_ERROR"
	}
}
