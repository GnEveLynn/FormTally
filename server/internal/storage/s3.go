package storage

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type S3Config struct {
	Endpoint, Region, Bucket, AccessKey, SecretKey string
}

type S3Store struct {
	config S3Config
	base   *url.URL
	now    func() time.Time
	client *http.Client
}

func NewS3Store(config S3Config, now func() time.Time) (*S3Store, error) {
	base, err := url.Parse(config.Endpoint)
	if err != nil || base.Scheme == "" || base.Host == "" || config.Region == "" || config.Bucket == "" || config.AccessKey == "" || config.SecretKey == "" {
		return nil, fmt.Errorf("invalid S3 configuration")
	}
	return &S3Store{config: config, base: base, now: now, client: http.DefaultClient}, nil
}

func (s *S3Store) Put(ctx context.Context, _ string, reader io.Reader, contentType string) (string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	key := randomKey()
	request, err := http.NewRequestWithContext(ctx, http.MethodPut, s.objectURL(key).String(), bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", contentType)
	s.sign(request, data)
	response, err := s.client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("S3 put returned %s", response.Status)
	}
	return key, nil
}

func (s *S3Store) Delete(ctx context.Context, key string) error {
	if !validKey(key) {
		return fmt.Errorf("invalid object key")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodDelete, s.objectURL(key).String(), nil)
	if err != nil {
		return err
	}
	s.sign(request, nil)
	response, err := s.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("S3 delete returned %s", response.Status)
	}
	return nil
}

func (s *S3Store) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	if !validKey(key) {
		return nil, fmt.Errorf("invalid object key")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, s.objectURL(key).String(), nil)
	if err != nil {
		return nil, err
	}
	s.sign(request, nil)
	response, err := s.client.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		response.Body.Close()
		return nil, fmt.Errorf("S3 get returned %s", response.Status)
	}
	return response.Body, nil
}

func (s *S3Store) PrivateURL(_ context.Context, key string, ttl time.Duration) (string, error) {
	if !validKey(key) || ttl <= 0 || ttl > 7*24*time.Hour {
		return "", fmt.Errorf("invalid private URL request")
	}
	now := s.now().UTC()
	date, scope := now.Format("20060102"), now.Format("20060102")+"/"+s.config.Region+"/s3/aws4_request"
	objectURL := s.objectURL(key)
	query := objectURL.Query()
	query.Set("X-Amz-Algorithm", "AWS4-HMAC-SHA256")
	query.Set("X-Amz-Credential", s.config.AccessKey+"/"+scope)
	query.Set("X-Amz-Date", now.Format("20060102T150405Z"))
	query.Set("X-Amz-Expires", strconv.FormatInt(int64(ttl/time.Second), 10))
	query.Set("X-Amz-SignedHeaders", "host")
	objectURL.RawQuery = query.Encode()
	canonical := strings.Join([]string{http.MethodGet, objectURL.EscapedPath(), objectURL.RawQuery, "host:" + objectURL.Host + "\n", "host", "UNSIGNED-PAYLOAD"}, "\n")
	toSign := "AWS4-HMAC-SHA256\n" + now.Format("20060102T150405Z") + "\n" + scope + "\n" + hash([]byte(canonical))
	query.Set("X-Amz-Signature", hex.EncodeToString(hmacBytes(s.signingKey(date), []byte(toSign))))
	objectURL.RawQuery = query.Encode()
	return objectURL.String(), nil
}

func (s *S3Store) objectURL(key string) *url.URL {
	copy := *s.base
	copy.Path = strings.TrimRight(copy.Path, "/") + "/" + s.config.Bucket + "/" + key
	return &copy
}

func (s *S3Store) sign(request *http.Request, body []byte) {
	now := s.now().UTC()
	payloadHash := hash(body)
	request.Header.Set("X-Amz-Date", now.Format("20060102T150405Z"))
	request.Header.Set("X-Amz-Content-Sha256", payloadHash)
	canonicalHeaders := "host:" + request.URL.Host + "\n" + "x-amz-content-sha256:" + payloadHash + "\n" + "x-amz-date:" + request.Header.Get("X-Amz-Date") + "\n"
	signedHeaders := "host;x-amz-content-sha256;x-amz-date"
	canonical := strings.Join([]string{request.Method, request.URL.EscapedPath(), request.URL.Query().Encode(), canonicalHeaders, signedHeaders, payloadHash}, "\n")
	date := now.Format("20060102")
	scope := date + "/" + s.config.Region + "/s3/aws4_request"
	toSign := "AWS4-HMAC-SHA256\n" + request.Header.Get("X-Amz-Date") + "\n" + scope + "\n" + hash([]byte(canonical))
	signature := hex.EncodeToString(hmacBytes(s.signingKey(date), []byte(toSign)))
	request.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+s.config.AccessKey+"/"+scope+", SignedHeaders="+signedHeaders+", Signature="+signature)
}

func (s *S3Store) signingKey(date string) []byte {
	dateKey := hmacBytes([]byte("AWS4"+s.config.SecretKey), []byte(date))
	regionKey := hmacBytes(dateKey, []byte(s.config.Region))
	serviceKey := hmacBytes(regionKey, []byte("s3"))
	return hmacBytes(serviceKey, []byte("aws4_request"))
}

func hmacBytes(key, value []byte) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(value)
	return mac.Sum(nil)
}
func hash(value []byte) string { sum := sha256.Sum256(value); return hex.EncodeToString(sum[:]) }
