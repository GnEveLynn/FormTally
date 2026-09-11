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

func TestS3StoreKeepsObjectsPrivateAndSignsShortLivedReads(t *testing.T) {
	var requests []*http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Clone(context.Background()))
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	now := time.Date(2026, 9, 11, 8, 0, 0, 0, time.UTC)
	store, err := NewS3Store(S3Config{Endpoint: server.URL, Region: "us-east-1", Bucket: "private-meals", AccessKey: "access", SecretKey: "secret"}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	key, err := store.Put(context.Background(), "user_private", strings.NewReader("image"), "image/jpeg")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(key, "user_private") || requests[0].Header.Get("X-Amz-Acl") != "" || !strings.HasPrefix(requests[0].Header.Get("Authorization"), "AWS4-HMAC-SHA256") {
		t.Fatalf("unsafe put: key=%q headers=%v", key, requests[0].Header)
	}
	readURL, err := store.PrivateURL(context.Background(), key, 90*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(readURL, "X-Amz-Expires=90") || !strings.Contains(readURL, "X-Amz-Signature=") {
		t.Fatalf("unsigned url: %s", readURL)
	}
	if err := store.Delete(context.Background(), key); err != nil {
		t.Fatal(err)
	}
	if requests[1].Method != http.MethodDelete {
		t.Fatalf("delete method = %s", requests[1].Method)
	}
}
