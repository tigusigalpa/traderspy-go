// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Igor Sazonov

package traderspy

import (
	"math/rand"
	"net/http"
	"time"
)

// RetryConfig controls the built-in retry behavior of the client.
//
// The client retries only failures worth retrying: HTTP 429 (rate
// limited), HTTP 503 (service unavailable — the API says such calls are not
// counted) and transport errors. Other statuses, including every other 4xx,
// fail immediately. The wait between attempts is exponential backoff with
// jitter, never shorter than the server's Retry-After when it sent one.
type RetryConfig struct {
	// MaxRetries is the number of retries after the first attempt, so the
	// total number of attempts is 1 + MaxRetries. Default 2 (3 attempts).
	// A negative value disables retries.
	MaxRetries int
	// MinWait is the base wait between attempts. Default 500ms.
	MinWait time.Duration
	// MaxWait caps the exponential backoff before Retry-After. Default 30s.
	MaxWait time.Duration
}

// defaultRetry is used when WithRetryConfig was not given.
var defaultRetry = RetryConfig{
	MaxRetries: 2,
	MinWait:    500 * time.Millisecond,
	MaxWait:    30 * time.Second,
}

// normalize fills zero fields with the defaults.
func (r RetryConfig) normalize() RetryConfig {
	if r.MinWait <= 0 {
		r.MinWait = defaultRetry.MinWait
	}
	if r.MaxWait <= 0 {
		r.MaxWait = defaultRetry.MaxWait
	}
	if r.MaxRetries == 0 {
		r.MaxRetries = defaultRetry.MaxRetries
	}
	return r
}

// retryableStatus reports whether an HTTP status is worth retrying.
func retryableStatus(code int) bool {
	return code == http.StatusTooManyRequests || code == http.StatusServiceUnavailable
}

// backoff doubles MinWait per retry up to MaxWait, then subtracts up to
// half as jitter. Retry-After is a lower bound, even above MaxWait.
func (r RetryConfig) backoff(attempt int, retryAfter time.Duration) time.Duration {
	wait := r.MaxWait
	if r.MinWait <= r.MaxWait>>attempt {
		wait = r.MinWait << attempt
	}
	if half := int64(wait) / 2; half > 0 {
		wait -= time.Duration(rand.Int63n(half))
	}
	return max(wait, retryAfter)
}
