// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Igor Sazonov

package traderspy_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	traderspy "github.com/tigusigalpa/traderspy-go"
)

func TestRetryOn429ThenSuccess(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		if call == 0 {
			w.Header().Set("Retry-After", "0")
			serveError(w, http.StatusTooManyRequests, traderspy.ErrCodeRateLimited, "slow down")
			return
		}
		serveFixture(w, "symbols.json")
	})
	c := s.client(t)
	out, resp, err := c.Market.Symbols(context.Background())
	if err != nil {
		t.Fatalf("Symbols: %v", err)
	}
	if s.calls() != 2 {
		t.Fatalf("calls = %d, want 2", s.calls())
	}
	if len(out.Symbols) == 0 || resp.StatusCode != 200 {
		t.Fatalf("out = %+v", out)
	}
}

func TestRetryAfterIsLowerBound(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		if call == 0 {
			w.Header().Set("Retry-After", "0.05")
			serveError(w, http.StatusServiceUnavailable, traderspy.ErrCodeServiceUnavailable, "busy")
			return
		}
		serveFixture(w, "symbols.json")
	})
	c := s.client(t)
	start := time.Now()
	if _, _, err := c.Market.Symbols(context.Background()); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed < 50*time.Millisecond {
		t.Fatalf("Retry-After not honored: waited only %v", elapsed)
	}
	if s.calls() != 2 {
		t.Fatalf("calls = %d", s.calls())
	}
}

func TestNoRetryOn400(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		serveError(w, http.StatusBadRequest, traderspy.ErrCodeInvalidRequest, "bad param")
	})
	c := s.client(t)
	_, _, err := c.Market.Symbols(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if s.calls() != 1 {
		t.Fatalf("calls = %d, want 1 (no retry on 400)", s.calls())
	}
}

func TestRetriesExhausted(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		w.Header().Set("Retry-After", "0")
		serveError(w, http.StatusTooManyRequests, traderspy.ErrCodeRateLimited, "slow down")
	})
	c := s.client(t) // MaxRetries: 2 → 3 attempts
	_, _, err := c.Market.Symbols(context.Background())
	var rl *traderspy.RateLimitError
	if !errors.As(err, &rl) {
		t.Fatalf("want *RateLimitError, got %T: %v", err, err)
	}
	if s.calls() != 3 {
		t.Fatalf("calls = %d, want 3", s.calls())
	}
}

func TestRetryDisabled(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		serveError(w, http.StatusServiceUnavailable, traderspy.ErrCodeServiceUnavailable, "busy")
	})
	c := s.client(t, traderspy.WithRetryConfig(traderspy.RetryConfig{MaxRetries: -1}))
	_, _, err := c.Market.Symbols(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if s.calls() != 1 {
		t.Fatalf("calls = %d, want 1", s.calls())
	}
}

func TestRetryOnNetworkError(t *testing.T) {
	calls := 0
	hc := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		return nil, io.ErrUnexpectedEOF
	})}
	c, err := traderspy.New(testKey,
		traderspy.WithHTTPClient(hc),
		traderspy.WithRetryConfig(traderspy.RetryConfig{MaxRetries: 1, MinWait: time.Millisecond, MaxWait: 5 * time.Millisecond}),
	)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = c.Market.Symbols(context.Background())
	var e *traderspy.Error
	if !errors.As(err, &e) {
		t.Fatalf("want *Error, got %T: %v", err, err)
	}
	if e.Code != traderspy.ErrCodeNetwork || e.StatusCode != 0 {
		t.Fatalf("got %+v", e)
	}
	if !e.IsTemporary() {
		t.Fatal("network error should be temporary")
	}
	if calls != 2 {
		t.Fatalf("transport attempts = %d, want 2", calls)
	}
}

func Test503NotCountedRetry(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		if call < 2 {
			w.Header().Set("Retry-After", "0")
			serveError(w, http.StatusServiceUnavailable, traderspy.ErrCodeServiceUnavailable, "source timeout")
			return
		}
		serveFixture(w, "symbols.json")
	})
	c := s.client(t)
	if _, _, err := c.Market.Symbols(context.Background()); err != nil {
		t.Fatal(err)
	}
	if s.calls() != 3 {
		t.Fatalf("calls = %d, want 3", s.calls())
	}
}
