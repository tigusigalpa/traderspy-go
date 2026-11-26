// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Igor Sazonov

package traderspy

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"
)

// Time is a timestamp the API sends as an ISO 8601 string with a zone
// ("2026-11-17T10:00:00Z") or, rarely, as a UTC date ("2026-11-17").
// It embeds [time.Time], so all time methods are available directly.
type Time struct {
	time.Time
}

// isoFormats are tried in order when decoding a Time.
var isoFormats = []string{
	time.RFC3339Nano,
	"2006-01-02T15:04:05",
	"2006-01-02",
}

const unixMillisThreshold = 1e12

// timestampText accepts an unquoted JSON number or a quoted timestamp.
func timestampText(data []byte) (string, error) {
	if len(data) > 0 && data[0] == '"' {
		var s string
		err := json.Unmarshal(data, &s)
		return s, err
	}
	return string(data), nil
}

// UnmarshalJSON implements [json.Unmarshaler].
func (t *Time) UnmarshalJSON(data []byte) error {
	s, err := timestampText(data)
	if err != nil {
		return err
	}
	if s == "" || s == "null" {
		t.Time = time.Time{}
		return nil
	}
	for _, format := range isoFormats {
		if parsed, err := time.Parse(format, s); err == nil {
			t.Time = parsed
			return nil
		}
	}
	// Last resort: a numeric Unix timestamp (seconds or milliseconds).
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		if n >= unixMillisThreshold || n <= -unixMillisThreshold {
			t.Time = time.UnixMilli(n).UTC()
		} else {
			t.Time = time.Unix(n, 0).UTC()
		}
		return nil
	}
	return fmt.Errorf("traderspy: cannot parse time %q", s)
}

// MarshalJSON implements [json.Marshaler].
func (t Time) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(t.Format(time.RFC3339Nano))
}

// UnixMillis is a timestamp the API sends as a number of milliseconds since
// the Unix epoch (for example the `sortDate` of a smart-money position).
// It embeds [time.Time], so all time methods are available directly.
type UnixMillis struct {
	time.Time
}

// UnmarshalJSON implements [json.Unmarshaler]. It accepts a JSON number or a
// quoted number. Absolute values below 1e12 are interpreted as seconds.
func (t *UnixMillis) UnmarshalJSON(data []byte) error {
	s, err := timestampText(data)
	if err != nil {
		return err
	}
	if s == "" || s == "null" {
		t.Time = time.Time{}
		return nil
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		if n > -unixMillisThreshold && n < unixMillisThreshold {
			t.Time = time.Unix(n, 0).UTC()
		} else {
			t.Time = time.UnixMilli(n).UTC()
		}
		return nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(f) || f < math.MinInt64 || f >= float64(math.MaxInt64) {
		return fmt.Errorf("traderspy: cannot parse millisecond timestamp %q", s)
	}
	if math.Abs(f) < unixMillisThreshold {
		f *= 1000
	}
	t.Time = time.UnixMilli(int64(f)).UTC()
	return nil
}

// MarshalJSON implements [json.Marshaler].
func (t UnixMillis) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("null"), nil
	}
	return []byte(strconv.FormatInt(t.UnixMilli(), 10)), nil
}
