// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Igor Sazonov

package traderspy

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Response carries the metadata of the HTTP answer behind every method
// call: the rate-limit counters, the request id and, for paginated
// endpoints, the pagination block of the body. It is returned alongside the
// decoded result, and also alongside errors so a 429 still exposes the
// limits the refusal carried.
type Response struct {
	// StatusCode is the HTTP status of the answer.
	StatusCode int
	// RateLimit holds the X-RateLimit-* headers, nil when none were sent.
	RateLimit *RateLimit
	// RequestID is the X-Request-Id header; quote it to support.
	RequestID string
	// Pagination is the body's pagination block for offset/page endpoints,
	// nil when the endpoint does not paginate.
	Pagination *Pagination
	// Header is the raw response header map.
	Header http.Header
}

// RateLimit is the usage budget the API reports on every answer through
// X-RateLimit-* headers. Daily counters reset at 00:00 UTC (see Reset).
// Fields are 0 when the corresponding header was absent.
type RateLimit struct {
	// Limit is the plan's calls per UTC day (or the key's own cap when
	// lower), from X-RateLimit-Limit.
	Limit int64
	// Remaining is the calls left today, from X-RateLimit-Remaining.
	Remaining int64
	// Reset is the moment the daily counters reset (00:00 UTC), from
	// X-RateLimit-Reset (Unix seconds). Zero when absent.
	Reset time.Time
	// LimitMinute is the plan's requests per minute, from
	// X-RateLimit-Limit-Minute.
	LimitMinute int64
	// RemainingMinute is the calls left in the current minute, from
	// X-RateLimit-Remaining-Minute.
	RemainingMinute int64
	// LimitKey is the key's own daily cap, from X-RateLimit-Limit-Key;
	// only sent when the key has one.
	LimitKey int64
	// RemainingKey is the calls left under the key's own daily cap, from
	// X-RateLimit-Remaining-Key; only sent when the key has one.
	RemainingKey int64
	// RetryAfter is the wait the API asked for (sent on 429 and 503).
	RetryAfter time.Duration
}

// rateLimitFromHeader parses the X-RateLimit-* and Retry-After headers.
// It returns nil when no rate-limit header was sent at all.
func rateLimitFromHeader(h http.Header) *RateLimit {
	rl := &RateLimit{
		Limit:           headerInt(h, "X-RateLimit-Limit"),
		Remaining:       headerInt(h, "X-RateLimit-Remaining"),
		LimitMinute:     headerInt(h, "X-RateLimit-Limit-Minute"),
		RemainingMinute: headerInt(h, "X-RateLimit-Remaining-Minute"),
		LimitKey:        headerInt(h, "X-RateLimit-Limit-Key"),
		RemainingKey:    headerInt(h, "X-RateLimit-Remaining-Key"),
		RetryAfter:      parseRetryAfter(h),
	}
	if reset := headerInt(h, "X-RateLimit-Reset"); reset != 0 {
		rl.Reset = time.Unix(reset, 0).UTC()
	}
	if *rl != (RateLimit{}) {
		return rl
	}
	for _, name := range []string{
		"X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset",
		"X-RateLimit-Limit-Minute", "X-RateLimit-Remaining-Minute",
		"X-RateLimit-Limit-Key", "X-RateLimit-Remaining-Key", "Retry-After",
	} {
		if h.Get(name) != "" {
			return rl
		}
	}
	return nil
}

func headerInt(h http.Header, name string) int64 {
	raw := h.Get(name)
	if raw == "" {
		return 0
	}
	n, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// Pagination is the body's pagination block on offset-based list endpoints
// (signals, positions, strategy alerts) and the page-based trader history.
type Pagination struct {
	// Limit is the page size used.
	Limit int `json:"limit"`
	// Total is the number of records matching the filters, when the API
	// reports it.
	Total int `json:"total"`
	// HasMore reports whether another page follows this one.
	HasMore bool `json:"hasMore"`
	// Offset is the offset this page started at (offset-based endpoints).
	Offset int `json:"offset"`
	// Page is the page number (page-based endpoints, e.g. trader history).
	Page int `json:"page"`
}

// ─── raw-payload preservation ────────────────────────────────────────────

// extraFields returns the top-level keys of a JSON object that do not match
// a known json tag of sample (a zero value of the target struct). Models
// keep them in their Extra field so nothing the API adds later is lost.
func extraFields(data []byte, sample any) map[string]json.RawMessage {
	var all map[string]json.RawMessage
	if err := json.Unmarshal(data, &all); err != nil || len(all) == 0 {
		return nil
	}
	for _, key := range jsonKeys(reflect.TypeOf(sample)) {
		delete(all, key)
	}
	if len(all) == 0 {
		return nil
	}
	return all
}

// jsonKeys lists the JSON names a struct type marshals to, recursing into
// embedded structs.
func jsonKeys(t reflect.Type) []string {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}
	var keys []string
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" && !f.Anonymous { // unexported
			continue
		}
		tag := f.Tag.Get("json")
		name, _, _ := strings.Cut(tag, ",")
		switch name {
		case "-":
			continue
		case "":
			if f.Anonymous || f.Type.Kind() == reflect.Struct {
				keys = append(keys, jsonKeys(f.Type)...)
			}
		default:
			keys = append(keys, name)
		}
	}
	return keys
}

// marshalWithExtra marshals v (an alias type without methods) and merges
// extra back into the resulting object, so round-tripping a model keeps the
// fields this SDK version does not know.
func marshalWithExtra(v any, extra map[string]json.RawMessage) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	if len(extra) == 0 {
		return data, nil
	}
	var merged map[string]json.RawMessage
	if err := json.Unmarshal(data, &merged); err != nil {
		return data, nil // v is not an object; keep its own form
	}
	for k, raw := range extra {
		if _, exists := merged[k]; !exists {
			merged[k] = raw
		}
	}
	return json.Marshal(merged)
}

// unwrapData decodes a response body into out. When the body is an object
// with a "data" member, that member is decoded; otherwise the whole body
// is. This covers both envelope and bare-object answer styles.
func unwrapData(body []byte, out any) error {
	var probe struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &probe); err == nil && len(probe.Data) > 0 {
		return json.Unmarshal(probe.Data, out)
	}
	return json.Unmarshal(body, out)
}
