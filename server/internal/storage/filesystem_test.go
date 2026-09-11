package storage

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFilesystemStoreUsesOpaqueKeysAndExpiringPrivateURLs(t *testing.T) {
	now := time.Date(2026, 9, 11, 8, 0, 0, 0, time.UTC)
	store := NewFilesystemStore(t.TempDir(), []byte("test-signing-secret"), func() time.Time { return now })
	key, err := store.Put(context.Background(), "user_123", strings.NewReader("private image"), "image/jpeg")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(key, "user_123") {
		t.Fatalf("key leaks user id: %s", key)
	}
	url, err := store.PrivateURL(context.Background(), key, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, url, nil)
	response := httptest.NewRecorder()
	store.ServeHTTP(response, request)
	body, _ := io.ReadAll(response.Result().Body)
	if response.Code != http.StatusOK || string(body) != "private image" {
		t.Fatalf("before expiry: %d %q", response.Code, body)
	}
	now = now.Add(61 * time.Second)
	response = httptest.NewRecorder()
	store.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("after expiry = %d", response.Code)
	}
	if err := store.Delete(context.Background(), key); err != nil {
		t.Fatal(err)
	}
	now = now.Add(-61 * time.Second)
	response = httptest.NewRecorder()
	store.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("after delete = %d", response.Code)
	}
}
