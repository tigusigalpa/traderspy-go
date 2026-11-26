// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Igor Sazonov

package traderspy

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// API error codes returned in the `code` field of the error envelope.
const (
	ErrCodeInvalidRequest       = "invalid_request"
	ErrCodeMissingAPIKey        = "missing_api_key"
	ErrCodeInvalidAPIKey        = "invalid_api_key"
	ErrCodePlanNotIncluded      = "plan_not_included"
	ErrCodeIPNotAllowed         = "ip_not_allowed"
	ErrCodeOriginNotAllowed     = "origin_not_allowed"
	ErrCodeStrategiesNotAllowed = "strategies_not_allowed"
	ErrCodeNotFound             = "not_found"
	ErrCodeNoData               = "no_data"
	ErrCodeRateLimited          = "rate_limited"
	ErrCodeDailyLimitReached    = "daily_limit_reached"
	ErrCodeKeyDailyLimitReached = "key_daily_limit_reached"
	ErrCodeBodyTooLarge         = "body_too_large"
	ErrCodeInternalError        = "internal_error"
	ErrCodeServiceUnavailable   = "service_unavailable"

	// ErrCodeNetwork is used for errors that never reached the API
	// (DNS, TCP, TLS, timeouts): the *Error has StatusCode 0.
	ErrCodeNetwork = "network_error"
)

// ErrorDetail is one entry of the optional `details` array of an API error:
// the parameter that was wrong and what was wrong with it.
type ErrorDetail struct {
	Param   string `json:"param"`
	Message string `json:"message"`
}

// Error is a refusal returned by the TraderSpy API, or a request that never
// reached it. The API answers errors as
//
//	{"error": {"code": "...", "message": "...", "requestId": "req_..."}}
//
// with a matching HTTP status. Use errors.As to test for the typed variants:
// [*AuthError] (401), [*ForbiddenError] (403), [*NotFoundError] (404) and
// [*RateLimitError] (429); every variant also matches *Error.
type Error struct {
	// Code is the API error code ("rate_limited", "invalid_api_key", ...),
	// or "network_error" when the request never got an answer.
	Code string
	// Message is the human-readable explanation from the API.
	Message string
	// RequestID is the value of the X-Request-Id header / `requestId` field.
	// Quote it when contacting TraderSpy support.
	RequestID string
	// StatusCode is the HTTP status (0 for network failures).
	StatusCode int
	// RetryAfter is the wait the API asked for via the Retry-After header
	// (sent on 429 and 503), or 0 when none was sent.
	RetryAfter time.Duration
	// Details names the offending parameters on invalid_request errors.
	Details []ErrorDetail
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.StatusCode != 0 {
		return fmt.Sprintf("traderspy: %d %s: %s", e.StatusCode, e.Code, e.Message)
	}
	return fmt.Sprintf("traderspy: %s: %s", e.Code, e.Message)
}

// IsRateLimit reports whether the request was refused for a rate limit
// (HTTP 429: rate_limited, daily_limit_reached or key_daily_limit_reached).
func (e *Error) IsRateLimit() bool { return e.StatusCode == http.StatusTooManyRequests }

// IsAuth reports whether the request was refused on authentication
// (HTTP 401: missing_api_key or invalid_api_key).
func (e *Error) IsAuth() bool { return e.StatusCode == http.StatusUnauthorized }

// IsTemporary reports whether the failure may go away by itself: rate
// limits, a busy data source (5xx) or a network problem. Other 4xx errors
// will fail the same way on a retry.
func (e *Error) IsTemporary() bool {
	switch e.StatusCode {
	case 0, http.StatusTooManyRequests, http.StatusInternalServerError,
		http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	}
	return false
}

// ErrBase aliases [Error]. The typed errors embed it under this name so
// that the embedded field does not clash with the Error() method; all
// fields and helpers of *Error stay promoted on the typed errors.
type ErrBase = Error

// AuthError is an *Error for HTTP 401 (missing_api_key, invalid_api_key).
// All *Error fields and helpers (IsAuth, RetryAfter, ...) are promoted.
type AuthError struct{ *ErrBase }

// Unwrap returns the underlying *Error, so errors.As(err, &e) with an
// *Error target matches through the typed wrapper.
func (e *AuthError) Unwrap() error { return e.ErrBase }

// ForbiddenError is an *Error for HTTP 403 (plan_not_included,
// ip_not_allowed, origin_not_allowed, strategies_not_allowed).
type ForbiddenError struct{ *ErrBase }

// Unwrap returns the underlying *Error.
func (e *ForbiddenError) Unwrap() error { return e.ErrBase }

// NotFoundError is an *Error for HTTP 404 (not_found, no_data).
type NotFoundError struct{ *ErrBase }

// Unwrap returns the underlying *Error.
func (e *NotFoundError) Unwrap() error { return e.ErrBase }

// RateLimitError is an *Error for HTTP 429 (rate_limited,
// daily_limit_reached, key_daily_limit_reached). RetryAfter carries the
// Retry-After header the API sent.
type RateLimitError struct{ *ErrBase }

// Unwrap returns the underlying *Error.
func (e *RateLimitError) Unwrap() error { return e.ErrBase }

// errorEnvelope mirrors the API's error body.
type errorEnvelope struct {
	Error struct {
		Code      string        `json:"code"`
		Message   string        `json:"message"`
		RequestID string        `json:"requestId"`
		Details   []ErrorDetail `json:"details"`
	} `json:"error"`
}

// errorFromResponse builds the typed error for a non-2xx response.
func errorFromResponse(resp *http.Response, body []byte) error {
	e := &Error{StatusCode: resp.StatusCode}

	var env errorEnvelope
	if err := json.Unmarshal(body, &env); err == nil && env.Error.Code != "" {
		e.Code = env.Error.Code
		e.Message = env.Error.Message
		e.RequestID = env.Error.RequestID
		e.Details = env.Error.Details
	} else {
		e.Code = "http_error"
		snippet := string(body)
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		e.Message = fmt.Sprintf("%s: %s", resp.Status, snippet)
	}
	if e.RequestID == "" {
		e.RequestID = resp.Header.Get("X-Request-Id")
	}
	e.RetryAfter = parseRetryAfter(resp.Header)

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return &AuthError{e}
	case http.StatusForbidden:
		return &ForbiddenError{e}
	case http.StatusNotFound:
		return &NotFoundError{e}
	case http.StatusTooManyRequests:
		return &RateLimitError{e}
	default:
		return e
	}
}

// networkError wraps a transport failure as an *Error with StatusCode 0.
func networkError(err error) *Error {
	return &Error{
		Code:       ErrCodeNetwork,
		Message:    err.Error(),
		StatusCode: 0,
	}
}

// parseRetryAfter reads the Retry-After header (seconds or an HTTP date).
func parseRetryAfter(h http.Header) time.Duration {
	raw := strings.TrimSpace(h.Get("Retry-After"))
	if raw == "" {
		return 0
	}
	if secs, err := strconv.ParseFloat(raw, 64); err == nil {
		if math.IsNaN(secs) || math.IsInf(secs, 0) || secs <= 0 {
			return 0
		}
		ns := secs * float64(time.Second)
		if ns >= float64(math.MaxInt64) {
			return time.Duration(math.MaxInt64)
		}
		return time.Duration(math.Ceil(ns))
	}
	if at, err := http.ParseTime(raw); err == nil {
		return max(0, time.Until(at))
	}
	return 0
}

// ValidationError is returned by the SDK before a request is sent when an
// option is out of the range the API accepts (for example a limit above the
// maximum). No HTTP call is made and no call is counted against the plan.
type ValidationError struct {
	// Param is the option field / API parameter that failed validation.
	Param string
	// Reason explains what is wrong.
	Reason string
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("traderspy: invalid %s: %s", e.Param, e.Reason)
}

// invalid builds a *ValidationError. It returns error so callers that
// `return invalid(...)` never produce a typed-nil error interface.
func invalid(param, reason string) error {
	return &ValidationError{Param: param, Reason: reason}
}

func checkRange(param string, value, min, max int) error {
	if value < min || value > max {
		return invalid(param, fmt.Sprintf("must be between %d and %d, got %d", min, max, value))
	}
	return nil
}

// checkEnum reports an error when v is set (non-empty) but not valid.
// It returns error (not *ValidationError) so that a nil result is a true
// nil error for callers that return it directly.
func checkEnum(param, value string, valid bool) error {
	if value != "" && !valid {
		return invalid(param, fmt.Sprintf("unknown value %q", value))
	}
	return nil
}

// maxBody is the largest response body the SDK will read (16 MiB); API
// answers are far smaller, this only bounds a misbehaving endpoint.
const maxBody = 16 << 20

var errResponseTooLarge = errors.New("response body exceeds 16 MiB")

// readBody reads and closes the body, detecting data beyond maxBody.
func readBody(resp *http.Response) ([]byte, error) {
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	if len(data) > maxBody {
		return data[:maxBody], errResponseTooLarge
	}
	return data, err
}
