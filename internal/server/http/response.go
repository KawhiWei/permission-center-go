package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/luck/permission-center-go/internal/biz"
)

func writeJSON(w http.ResponseWriter, status int, value any) {
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
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
