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

func WriteError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	WriteJSON(w, status, errorResponse{Error: apiError{
		Code:      code,
		Message:   message,
		RequestID: RequestID(r.Context()),
	}})
}

type requestIDKey struct{}

func RequestID(ctx context.Context) string {
	value, _ := ctx.Value(requestIDKey{}).(string)
	return value
}
