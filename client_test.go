// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Igor Sazonov

package traderspy_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	traderspy "github.com/tigusigalpa/traderspy-go"
)

func TestNewRequiresKey(t *testing.T) {
	for _, key := range []string{"", "   "} {
		if _, err := traderspy.New(key); err == nil {
			t.Fatalf("New(%q) should fail", key)
		}
	}
}

func TestNewFromEnv(t *testing.T) {
	t.Setenv(traderspy.EnvAPIKey, "")
	if _, err := traderspy.NewFromEnv(); err == nil {
		t.Fatal("NewFromEnv with empty env should fail")
	}
	t.Setenv(traderspy.EnvAPIKey, " tspy_env ")
	c, err := traderspy.NewFromEnv()
	if err != nil {
		t.Fatalf("NewFromEnv: %v", err)
	}
	if c == nil {
		t.Fatal("nil client")
	}
}

func TestClientStringHidesKey(t *testing.T) {
	c, err := traderspy.New(testKey)
	if err != nil {
		t.Fatal(err)
	}
	for _, format := range []string{"%v", "%+v", "%#v"} {
		if strings.Contains(fmt.Sprintf(format, c), testKey) {
			t.Fatalf("%q formatting leaks the key", format)
		}
	}
}

func TestAuthHeaderAndNoKeyInURL(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		serveFixture(w, "symbols.json")
	})
	c := s.client(t)

	if _, _, err := c.Market.Symbols(context.Background()); err != nil {
		t.Fatalf("Symbols: %v", err)
	}
	got := s.last()
	if auth := got.Header.Get("Authorization"); auth != "Bearer "+testKey {
		t.Fatalf("Authorization = %q", auth)
	}
	if strings.Contains(got.Path+got.Query.Encode(), testKey) || strings.Contains(got.Query.Encode(), testKey) {
		t.Fatal("API key leaked into URL")
	}
	if accept := got.Header.Get("Accept"); accept != "application/json" {
		t.Fatalf("Accept = %q", accept)
	}
	if ua := got.Header.Get("User-Agent"); !strings.HasPrefix(ua, "traderspy-go/") {
		t.Fatalf("User-Agent = %q", ua)
	}
}

func TestCustomUserAgent(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		serveFixture(w, "symbols.json")
	})
	c := s.client(t, traderspy.WithUserAgent("my-bot/1.0"))
	if _, _, err := c.Market.Symbols(context.Background()); err != nil {
		t.Fatal(err)
	}
	if ua := s.last().Header.Get("User-Agent"); ua != "my-bot/1.0" {
		t.Fatalf("User-Agent = %q", ua)
	}
}

func TestRateLimitParsing(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		serveFixture(w, "symbols.json")
	})
	c := s.client(t)

	_, resp, err := c.Market.Symbols(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if resp.RequestID != "req_test_1" {
		t.Fatalf("RequestID = %q", resp.RequestID)
	}
	rl := resp.RateLimit
	if rl == nil {
		t.Fatal("no RateLimit")
	}
	if rl.Limit != 300 || rl.Remaining != 271 || rl.LimitMinute != 60 || rl.RemainingMinute != 58 || rl.LimitKey != 100 || rl.RemainingKey != 77 {
		t.Fatalf("RateLimit = %+v", rl)
	}
	if got := rl.Reset.Unix(); got != 1764201600 {
		t.Fatalf("Reset = %d", got)
	}
	// Client-level snapshot of the last answer.
	if c.RateLimit() == nil || c.RateLimit().Remaining != 271 {
		t.Fatalf("client.RateLimit = %+v", c.RateLimit())
	}
	rl.Remaining = 0
	snapshot := c.RateLimit()
	snapshot.Remaining = 1
	if got := c.RateLimit().Remaining; got != 271 {
		t.Fatalf("modifying a returned value changed client's snapshot to %d", got)
	}
}

func TestOptions(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		serveFixture(w, "symbols.json")
	})
	hc := &http.Client{Transport: s, Timeout: 5 * time.Second}
	c, err := traderspy.New(testKey,
		traderspy.WithBaseURL(testBaseURL+"///"),
		traderspy.WithBaseURL(""), // ignored: empty keeps previous
		traderspy.WithHTTPClient(hc),
		traderspy.WithHTTPClient(nil), // ignored
		traderspy.WithTimeout(10*time.Second),
		traderspy.WithTimeout(0), // ignored
		traderspy.WithUserAgent(""),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := c.Market.Symbols(context.Background()); err != nil {
		t.Fatal(err)
	}
	if hc.Timeout != 5*time.Second {
		t.Fatalf("WithTimeout modified caller's client: %v", hc.Timeout)
	}
	if got := s.last().Path; got != "/v1/market/symbols" {
		t.Fatalf("path = %q", got)
	}
}

func TestValidationErrorString(t *testing.T) {
	var err error = &traderspy.ValidationError{Param: "limit", Reason: "too big"}
	if err.Error() == "" {
		t.Fatal("empty message")
	}
}

func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		w.Header().Set("Retry-After", "60")
		serveError(w, http.StatusTooManyRequests, traderspy.ErrCodeRateLimited, "slow down")
		cancel()
	})
	c := s.client(t)
	_, resp, err := c.Market.Symbols(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if resp == nil || resp.StatusCode != http.StatusTooManyRequests || s.calls() != 1 {
		t.Fatalf("response = %+v, attempts = %d", resp, s.calls())
	}
}

func TestContextCancellationDuringTransport(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	hc := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		cancel()
		return nil, r.Context().Err()
	})}
	c, err := traderspy.New(testKey, traderspy.WithHTTPClient(hc),
		traderspy.WithRetryConfig(traderspy.RetryConfig{MaxRetries: -1}))
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = c.Market.Symbols(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestOversizedResponseIsNotRetried(t *testing.T) {
	calls := 0
	hc := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		// The first 16 MiB alone are valid JSON, followed by one extra byte.
		body := `{"data":[]}` + strings.Repeat(" ", (16<<20)-len(`{"data":[]}`)+1)
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	c, err := traderspy.New(testKey, traderspy.WithHTTPClient(hc))
	if err != nil {
		t.Fatal(err)
	}
	_, resp, err := c.Market.Symbols(context.Background())
	if err == nil || !strings.Contains(err.Error(), "exceeds 16 MiB") {
		t.Fatalf("expected response size error, got %v", err)
	}
	if resp == nil || resp.StatusCode != http.StatusOK || calls != 1 {
		t.Fatalf("response = %+v, attempts = %d", resp, calls)
	}
}
