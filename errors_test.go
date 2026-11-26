// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Igor Sazonov

package traderspy_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	traderspy "github.com/tigusigalpa/traderspy-go"
)

func call(t *testing.T, status int, code, message string, headers map[string]string) (*traderspy.Response, error) {
	t.Helper()
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		for k, v := range headers {
			w.Header().Set(k, v)
		}
		serveError(w, status, code, message)
	})
	c := s.client(t)
	return func() (*traderspy.Response, error) {
		_, resp, err := c.Market.Symbols(context.Background())
		return resp, err
	}()
}

func TestTypedErrors(t *testing.T) {
	cases := []struct {
		status int
		code   string
		as     func(error) bool
	}{
		{400, traderspy.ErrCodeInvalidRequest, func(err error) bool {
			var e *traderspy.Error
			return errors.As(err, &e)
		}},
		{401, traderspy.ErrCodeMissingAPIKey, func(err error) bool {
			var e *traderspy.AuthError
			return errors.As(err, &e)
		}},
		{401, traderspy.ErrCodeInvalidAPIKey, func(err error) bool {
			var e *traderspy.AuthError
			return errors.As(err, &e)
		}},
		{403, traderspy.ErrCodePlanNotIncluded, func(err error) bool {
			var e *traderspy.ForbiddenError
			return errors.As(err, &e)
		}},
		{403, traderspy.ErrCodeStrategiesNotAllowed, func(err error) bool {
			var e *traderspy.ForbiddenError
			return errors.As(err, &e)
		}},
		{404, traderspy.ErrCodeNotFound, func(err error) bool {
			var e *traderspy.NotFoundError
			return errors.As(err, &e)
		}},
		{404, traderspy.ErrCodeNoData, func(err error) bool {
			var e *traderspy.NotFoundError
			return errors.As(err, &e)
		}},
		{413, traderspy.ErrCodeBodyTooLarge, func(err error) bool {
			var e *traderspy.Error
			return errors.As(err, &e)
		}},
		{500, traderspy.ErrCodeInternalError, func(err error) bool {
			var e *traderspy.Error
			return errors.As(err, &e)
		}},
	}
	for _, tc := range cases {
		resp, err := call(t, tc.status, tc.code, "nope", nil)
		if err == nil {
			t.Fatalf("%d %s: expected error", tc.status, tc.code)
		}
		if !tc.as(err) {
			t.Fatalf("%d %s: unexpected error type %T", tc.status, tc.code, err)
		}
		var e *traderspy.Error
		if !errors.As(err, &e) {
			t.Fatalf("%d %s: cannot reach *Error via errors.As", tc.status, tc.code)
		}
		if e.Code != tc.code || e.StatusCode != tc.status || e.RequestID != "req_err_1" {
			t.Fatalf("%d %s: got %+v", tc.status, tc.code, e)
		}
		if resp == nil || resp.StatusCode != tc.status {
			t.Fatalf("%d: response metadata lost", tc.status)
		}
	}
}

func TestErrorHelpers(t *testing.T) {
	_, err := call(t, 401, traderspy.ErrCodeInvalidAPIKey, "bad key", nil)
	var e *traderspy.Error
	if !errors.As(err, &e) {
		t.Fatal("no *Error")
	}
	if !e.IsAuth() || e.IsRateLimit() || e.IsTemporary() {
		t.Fatalf("helpers wrong for 401: %+v", e)
	}

	_, err = call(t, 429, traderspy.ErrCodeRateLimited, "slow down", map[string]string{"Retry-After": "7"})
	if !errors.As(err, &e) {
		t.Fatal("no *Error")
	}
	if !e.IsRateLimit() || !e.IsTemporary() {
		t.Fatalf("helpers wrong for 429: %+v", e)
	}
	if e.RetryAfter != 7*time.Second {
		t.Fatalf("RetryAfter = %v", e.RetryAfter)
	}
	var rl *traderspy.RateLimitError
	if !errors.As(err, &rl) {
		t.Fatal("no *RateLimitError")
	}
	if rl.RetryAfter != 7*time.Second { // promoted field
		t.Fatalf("RateLimitError.RetryAfter = %v", rl.RetryAfter)
	}
}

func TestErrorDetails(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write(fixtureBytes("error.json"))
	})
	c := s.client(t)
	_, _, err := c.Market.Symbols(context.Background())
	var e *traderspy.Error
	if !errors.As(err, &e) {
		t.Fatalf("no *Error: %v", err)
	}
	if e.Code != traderspy.ErrCodeInvalidRequest {
		t.Fatalf("Code = %q", e.Code)
	}
	if len(e.Details) != 1 || e.Details[0].Param != "limit" {
		t.Fatalf("Details = %+v", e.Details)
	}
	if e.Error() == "" {
		t.Fatal("empty Error()")
	}
}

func TestErrorNonJSONBody(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("bad gateway"))
	})
	c := s.client(t, traderspy.WithRetryConfig(traderspy.RetryConfig{MaxRetries: -1}))
	_, _, err := c.Market.Symbols(context.Background())
	var e *traderspy.Error
	if !errors.As(err, &e) {
		t.Fatalf("no *Error: %v", err)
	}
	if e.StatusCode != http.StatusBadGateway {
		t.Fatalf("StatusCode = %d", e.StatusCode)
	}
}
