package httpapi

import (
	"context"
	"net/http"
)

type errorResponse struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"requestId"`
}

func writeError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	writeJSON(w, status, errorResponse{Error: apiError{
		Code:      code,
		Message:   message,
		RequestID: requestID(r.Context()),
	}})
}

type requestIDKey struct{}

func requestID(ctx context.Context) string {
	value, _ := ctx.Value(requestIDKey{}).(string)
	return value
}
