// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Igor Sazonov

package traderspy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

// DefaultBaseURL is the production TraderSpy API endpoint.
const DefaultBaseURL = "https://api.traderspy.app"

// EnvAPIKey is the environment variable [NewFromEnv] reads.
const EnvAPIKey = "TRADERSPY_API_KEY"

// Client is a TraderSpy API client. It is safe for concurrent use.
//
// The API is read-only: the client exposes no way to place an order or move
// funds.
type Client struct {
	key       string
	baseURL   string
	userAgent string
	http      *http.Client
	retry     RetryConfig

	lastRateLimit atomic.Pointer[RateLimit]

	// Signals groups the AI-signals endpoints.
	Signals *SignalsService
	// SmartMoney groups the smart-money endpoints (leaderboards, profiles,
	// positions, statistics, exchanges).
	SmartMoney *SmartMoneyService
	// Market groups the market-data endpoints (prices, candles, symbols,
	// derivatives).
	Market *MarketService
	// Analysis groups the analysis endpoints (technical indicators,
	// screener, backtest).
	Analysis *AnalysisService
	// Strategies groups the endpoints that read your own alert strategies;
	// they need a key allowed to read strategies in its settings.
	Strategies *StrategiesService
}

// Option customizes a [Client].
type Option func(*Client)

// WithHTTPClient sets the *http.Client used for requests, for example to
// add a proxy or an instrumented transport. When combined with
// [WithTimeout], pass WithTimeout after it so the timeout applies to the
// custom client too.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		if hc != nil {
			c.http = hc
		}
	}
}

// WithBaseURL overrides the API base URL — for tests, staging or a proxy.
// Pass the server root (the "/v1" prefix is added by each endpoint), e.g.
// "http://localhost:8080". A trailing slash is ignored.
func WithBaseURL(rawurl string) Option {
	return func(c *Client) {
		if u := strings.TrimRight(strings.TrimSpace(rawurl), "/"); u != "" {
			c.baseURL = u
		}
	}
}

// WithTimeout sets a per-attempt timeout. The default client uses 30s.
// The supplied HTTP client, if any, is copied before changing its timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		if d > 0 {
			hc := *c.http
			hc.Timeout = d
			c.http = &hc
		}
	}
}

// WithUserAgent overrides the User-Agent header (default
// "traderspy-go/<Version>").
func WithUserAgent(ua string) Option {
	return func(c *Client) {
		if ua != "" {
			c.userAgent = ua
		}
	}
}

// WithRetryConfig replaces the retry configuration; see [RetryConfig].
// Zero fields keep their defaults. Set MaxRetries to a negative value to
// disable retries entirely.
func WithRetryConfig(rc RetryConfig) Option {
	return func(c *Client) {
		c.retry = rc.normalize()
	}
}

// New creates a client for the given API key ("tspy_..."). The key is sent
// only as an `Authorization: Bearer` header, never in a URL.
func New(key string, opts ...Option) (*Client, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("traderspy: empty API key (create one at https://traderspy.app/api, the free plan needs no card)")
	}
	c := &Client{
		key:       key,
		baseURL:   DefaultBaseURL,
		userAgent: "traderspy-go/" + Version,
		http:      &http.Client{Timeout: 30 * time.Second},
		retry:     defaultRetry,
	}
	for _, opt := range opts {
		opt(c)
	}
	c.Signals = &SignalsService{client: c}
	c.SmartMoney = &SmartMoneyService{client: c}
	c.Market = &MarketService{client: c}
	c.Analysis = &AnalysisService{client: c}
	c.Strategies = &StrategiesService{client: c}
	return c, nil
}

// NewFromEnv creates a client reading the key from the TRADERSPY_API_KEY
// environment variable. It returns an error when the variable is empty.
func NewFromEnv(opts ...Option) (*Client, error) {
	key := strings.TrimSpace(os.Getenv(EnvAPIKey))
	if key == "" {
		return nil, fmt.Errorf("traderspy: %s is not set (create a key at https://traderspy.app/api)", EnvAPIKey)
	}
	return New(key, opts...)
}

// String describes the client without ever exposing the API key.
func (c *Client) String() string {
	return fmt.Sprintf("traderspy.Client{baseURL: %s}", c.baseURL)
}

// GoString describes the client for Go-syntax formatting (%#v).
func (c *Client) GoString() string { return c.String() }

// RateLimit returns a copy of the latest rate-limit counters reported by
// the API, or nil until a response reports them. Under concurrent use prefer the
// [*Response] returned by each call, which carries its own RateLimit.
func (c *Client) RateLimit() *RateLimit {
	if limit := c.lastRateLimit.Load(); limit != nil {
		snapshot := *limit
		return &snapshot
	}
	return nil
}

// get issues a GET request and decodes the whole JSON body into out.
func (c *Client) get(ctx context.Context, path string, query url.Values, out any) (*Response, error) {
	body, resp, err := c.do(ctx, http.MethodGet, path, query, nil)
	if err != nil {
		return resp, err
	}
	if out != nil {
		if err := json.Unmarshal(body, out); err != nil {
			return resp, fmt.Errorf("traderspy: decoding response: %w", err)
		}
	}
	return resp, nil
}

// getData issues a GET request and decodes the payload into out, accepting
// both a bare JSON object and the {"data": ...} envelope.
func (c *Client) getData(ctx context.Context, path string, query url.Values, out any) (*Response, error) {
	body, resp, err := c.do(ctx, http.MethodGet, path, query, nil)
	if err != nil {
		return resp, err
	}
	if err := unwrapData(body, out); err != nil {
		return resp, fmt.Errorf("traderspy: decoding response: %w", err)
	}
	return resp, nil
}

// post issues a POST request with a JSON body and decodes the JSON body of
// the answer into out.
func (c *Client) post(ctx context.Context, path string, body any, out any) (*Response, error) {
	data, resp, err := c.do(ctx, http.MethodPost, path, nil, body)
	if err != nil {
		return resp, err
	}
	if out != nil {
		if err := json.Unmarshal(data, out); err != nil {
			return resp, fmt.Errorf("traderspy: decoding response: %w", err)
		}
	}
	return resp, nil
}

// do performs the HTTP request with retries and returns the raw body plus
// the response metadata.
func (c *Client) do(ctx context.Context, method, path string, query url.Values, body any) ([]byte, *Response, error) {
	endpoint := c.baseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, nil, fmt.Errorf("traderspy: encoding request body: %w", err)
		}
	}

	maxRetries := max(0, c.retry.MaxRetries)
	var lastErr error
	var lastResp *Response
	var retryAfter time.Duration

	for remaining := maxRetries; remaining >= 0; remaining-- {
		if remaining < maxRetries {
			timer := time.NewTimer(c.retry.backoff(maxRetries-remaining-1, retryAfter))
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, lastResp, ctx.Err()
			case <-timer.C:
			}
		}

		req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, nil, fmt.Errorf("traderspy: building request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", c.userAgent)
		if bodyBytes != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.http.Do(req)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, lastResp, ctxErr
			}
			lastErr = networkError(err)
			lastResp = nil
			retryAfter = 0
			continue // transport errors are retryable
		}

		data, readErr := readBody(resp)
		apiResp := &Response{
			StatusCode: resp.StatusCode,
			RateLimit:  rateLimitFromHeader(resp.Header),
			RequestID:  resp.Header.Get("X-Request-Id"),
			Header:     resp.Header,
		}
		if apiResp.RateLimit != nil {
			snapshot := *apiResp.RateLimit
			c.lastRateLimit.Store(&snapshot)
		}
		lastResp = apiResp

		if readErr != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, apiResp, ctxErr
			}
		}

		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			apiErr := errorFromResponse(resp, data)
			if !retryableStatus(resp.StatusCode) {
				return nil, apiResp, apiErr
			}
			lastErr = apiErr
			retryAfter = 0
			if apiResp.RateLimit != nil {
				retryAfter = apiResp.RateLimit.RetryAfter
			}
			continue // 429/503: retry
		}

		if readErr != nil {
			lastErr = fmt.Errorf("traderspy: reading response: %w", readErr)
			if readErr == errResponseTooLarge {
				return nil, apiResp, lastErr
			}
			retryAfter = 0
			continue
		}

		return data, apiResp, nil
	}

	return nil, lastResp, lastErr
}
