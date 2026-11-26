// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Igor Sazonov

package traderspy_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	traderspy "github.com/tigusigalpa/traderspy-go"
)

const (
	testKey     = "tspy_test_key_123"
	testBaseURL = "http://traderspy.test"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// recordedRequest captures one incoming request of the test server.
type recordedRequest struct {
	Method string
	Path   string
	Query  url.Values
	Header http.Header
	Body   string
}

// server records HTTP requests and calls the fixture handler in memory.
type server struct {
	handler  func(http.ResponseWriter, *http.Request, int)
	mu       sync.Mutex
	requests []recordedRequest
}

// newServer builds a test API transport. The handler receives the recorder;
// it may serve different payloads per path or per call number.
func newServer(t *testing.T, handler func(w http.ResponseWriter, r *http.Request, call int)) *server {
	t.Helper()
	return &server{handler: handler}
}

func (s *server) RoundTrip(r *http.Request) (*http.Response, error) {
	var body []byte
	if r.Body != nil {
		var err error
		body, err = readAll(r)
		if err != nil {
			return nil, err
		}
	}
	if err := r.Context().Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	call := len(s.requests)
	s.requests = append(s.requests, recordedRequest{
		Method: r.Method,
		Path:   r.URL.Path,
		Query:  r.URL.Query(),
		Header: r.Header.Clone(),
		Body:   string(body),
	})
	s.mu.Unlock()
	w := httptest.NewRecorder()
	s.handler(w, r, call)
	return w.Result(), nil
}

// calls returns how many requests the server saw.
func (s *server) calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.requests)
}

// last returns the most recent recorded request.
func (s *server) last() recordedRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.requests[len(s.requests)-1]
}

// client builds an SDK client pointed at the test server.
func (s *server) client(t *testing.T, opts ...traderspy.Option) *traderspy.Client {
	t.Helper()
	base := []traderspy.Option{
		traderspy.WithBaseURL(testBaseURL),
		traderspy.WithHTTPClient(&http.Client{Transport: s}),
		traderspy.WithRetryConfig(traderspy.RetryConfig{MaxRetries: 2, MinWait: time.Millisecond, MaxWait: 10 * time.Millisecond}),
	}
	c, err := traderspy.New(testKey, append(base, opts...)...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

// serveFixture answers 200 with a testdata fixture and the standard
// rate-limit headers.
func serveFixture(w http.ResponseWriter, name string) {
	setRateLimitHeaders(w)
	w.Header().Set("X-Request-Id", "req_test_1")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(fixtureBytes(name))
}

var fixtureCache = struct {
	sync.Mutex
	m map[string][]byte
}{m: map[string][]byte{}}

func fixtureBytes(name string) []byte {
	fixtureCache.Lock()
	defer fixtureCache.Unlock()
	if b, ok := fixtureCache.m[name]; ok {
		return b
	}
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		panic("missing fixture " + name + ": " + err.Error())
	}
	fixtureCache.m[name] = data
	return data
}

// setRateLimitHeaders writes the headers the API sends on every answer.
func setRateLimitHeaders(w http.ResponseWriter) {
	w.Header().Set("X-RateLimit-Limit", "300")
	w.Header().Set("X-RateLimit-Remaining", "271")
	w.Header().Set("X-RateLimit-Reset", "1764201600")
	w.Header().Set("X-RateLimit-Limit-Minute", "60")
	w.Header().Set("X-RateLimit-Remaining-Minute", "58")
	w.Header().Set("X-RateLimit-Limit-Key", "100")
	w.Header().Set("X-RateLimit-Remaining-Key", "77")
}

// serveError answers an API error envelope.
func serveError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("X-Request-Id", "req_err_1")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"error":{"code":"` + code + `","message":"` + message + `","requestId":"req_err_1"}}`))
}

// readAll drains a request body.
func readAll(r *http.Request) ([]byte, error) {
	defer func() { _ = r.Body.Close() }()
	return io.ReadAll(r.Body)
}
