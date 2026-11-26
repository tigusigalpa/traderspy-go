// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Igor Sazonov

package traderspy

import (
	"net/http"
	"testing"
	"time"
)

// Backoff includes jitter between 50% and 100% of the capped delay.
func TestBackoffBounds(t *testing.T) {
	cfg := RetryConfig{MaxRetries: 3, MinWait: 100 * time.Millisecond, MaxWait: time.Second}.normalize()
	for attempt := 0; attempt < 8; attempt++ {
		w := cfg.backoff(attempt, 0)
		if w < 50*time.Millisecond || w > time.Second {
			t.Fatalf("attempt %d: wait %v out of bounds", attempt, w)
		}
	}
}

func TestBackoffNanosecond(t *testing.T) {
	cfg := RetryConfig{MinWait: time.Nanosecond, MaxWait: time.Nanosecond}.normalize()
	for _, attempt := range []int{0, 1, 64} {
		if got := cfg.backoff(attempt, 0); got != time.Nanosecond {
			t.Fatalf("attempt %d: got %v, want 1ns", attempt, got)
		}
	}
}

func TestRateLimitZeroRemaining(t *testing.T) {
	h := make(http.Header)
	h.Set("X-RateLimit-Remaining", "0")
	if limit := rateLimitFromHeader(h); limit == nil || limit.Remaining != 0 {
		t.Fatalf("got %+v, expected a reported zero remaining", limit)
	}
}

func TestBackoffHonoursRetryAfter(t *testing.T) {
	cfg := RetryConfig{MinWait: 100 * time.Millisecond, MaxWait: time.Second}.normalize()
	if w := cfg.backoff(0, 5*time.Second); w < 5*time.Second {
		t.Fatalf("wait %v below Retry-After 5s", w)
	}
	// Retry-Above MaxWait is still honored (server wins).
	if w := cfg.backoff(3, 10*time.Second); w < 10*time.Second {
		t.Fatalf("wait %v below Retry-After 10s", w)
	}
}

func TestRetryConfigNormalize(t *testing.T) {
	r := RetryConfig{}.normalize()
	if r.MaxRetries != defaultRetry.MaxRetries || r.MinWait != defaultRetry.MinWait || r.MaxWait != defaultRetry.MaxWait {
		t.Fatalf("normalize = %+v", r)
	}
	r = RetryConfig{MaxRetries: -1}.normalize()
	if r.MaxRetries != -1 {
		t.Fatalf("negative MaxRetries must survive normalize: %+v", r)
	}
}

func TestRetryableStatus(t *testing.T) {
	for code, want := range map[int]bool{200: false, 400: false, 401: false, 403: false, 404: false, 429: true, 500: false, 502: false, 503: true} {
		if got := retryableStatus(code); got != want {
			t.Fatalf("retryableStatus(%d) = %v, want %v", code, got, want)
		}
	}
}

func TestUnwrapData(t *testing.T) {
	type payload struct {
		X int `json:"x"`
	}
	var p payload
	if err := unwrapData([]byte(`{"data":{"x":7}}`), &p); err != nil || p.X != 7 {
		t.Fatalf("wrapped: %v %+v", err, p)
	}
	if err := unwrapData([]byte(`{"x":8}`), &p); err != nil || p.X != 8 {
		t.Fatalf("bare: %v %+v", err, p)
	}
}
