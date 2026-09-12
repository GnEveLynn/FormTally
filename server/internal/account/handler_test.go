package account

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/GnEveLynn/FormTally/server/internal/httpapi"
)

func TestHandlerAcceptsDeletionAndClearsCookie(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store)
	service.now = func() time.Time { return time.Date(2026, 9, 11, 7, 0, 0, 0, time.UTC) }
	handler := NewHandler(service, func(*http.Request) (string, error) { return "user_1", nil })
	router := httpapi.NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)), []string{"http://127.0.0.1:5173"}, handler.Register)
	request := httptest.NewRequest(http.MethodPost, "/v1/account-deletions", bytes.NewBufferString(`{"code":"123456","verificationRequestId":"verify_1","confirmation":"DELETE"}`))
	request.Header.Set("Origin", "http://127.0.0.1:5173")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted || !strings.Contains(response.Body.String(), `"status":"accepted"`) || len(response.Result().Cookies()) == 0 || response.Result().Cookies()[0].MaxAge >= 0 {
		t.Fatalf("response=%d body=%s cookies=%+v", response.Code, response.Body.String(), response.Result().Cookies())
	}
}
