package storage

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type FilesystemStore struct {
	directory string
	secret    []byte
	now       func() time.Time
}

func NewFilesystemStore(directory string, secret []byte, now func() time.Time) *FilesystemStore {
	return &FilesystemStore{directory: directory, secret: append([]byte(nil), secret...), now: now}
}

func (s *FilesystemStore) Put(_ context.Context, _ string, reader io.Reader, _ string) (string, error) {
	if err := os.MkdirAll(s.directory, 0700); err != nil {
		return "", err
	}
	key := randomKey()
	file, err := os.OpenFile(filepath.Join(s.directory, key), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", err
	}
	_, copyErr := io.Copy(file, reader)
	closeErr := file.Close()
	if copyErr != nil {
		return "", copyErr
	}
	return key, closeErr
}

func (s *FilesystemStore) PrivateURL(_ context.Context, key string, ttl time.Duration) (string, error) {
	if !validKey(key) {
		return "", fmt.Errorf("invalid object key")
	}
	expires := s.now().Add(ttl).Unix()
	query := url.Values{"expires": {strconv.FormatInt(expires, 10)}, "signature": {s.signature(key, expires)}}
	return "/v1/private-images/" + key + "?" + query.Encode(), nil
}

func (s *FilesystemStore) Open(_ context.Context, key string) (io.ReadCloser, error) {
	if !validKey(key) {
		return nil, fmt.Errorf("invalid object key")
	}
	return os.Open(filepath.Join(s.directory, key))
}

func (s *FilesystemStore) Delete(_ context.Context, key string) error {
	if !validKey(key) {
		return fmt.Errorf("invalid object key")
	}
	err := os.Remove(filepath.Join(s.directory, key))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (s *FilesystemStore) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/v1/private-images/")
	expires, err := strconv.ParseInt(r.URL.Query().Get("expires"), 10, 64)
	if err != nil || !validKey(key) || s.now().Unix() > expires || !hmac.Equal([]byte(r.URL.Query().Get("signature")), []byte(s.signature(key, expires))) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	file, err := os.Open(filepath.Join(s.directory, key))
	if os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
		return
	}
	defer file.Close()
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "private, no-cache")
	_, _ = io.Copy(w, file)
}

func (s *FilesystemStore) signature(key string, expires int64) string {
	mac := hmac.New(sha256.New, s.secret)
	fmt.Fprintf(mac, "%s:%d", key, expires)
	return hex.EncodeToString(mac.Sum(nil))
}

func randomKey() string {
	value := make([]byte, 24)
	_, _ = rand.Read(value)
	return hex.EncodeToString(value)
}

func validKey(key string) bool {
	_, err := hex.DecodeString(key)
	return len(key) == 48 && err == nil
}
