// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Igor Sazonov

package traderspy_test

import (
	"encoding/json"
	"testing"
	"time"

	traderspy "github.com/tigusigalpa/traderspy-go"
)

func TestTimeUnmarshal(t *testing.T) {
	cases := []struct {
		in   string
		want time.Time
	}{
		{`"2026-11-26T09:15:00Z"`, time.Date(2026, 11, 26, 9, 15, 0, 0, time.UTC)},
		{`"2026-11-26T09:15:00.123Z"`, time.Date(2026, 11, 26, 9, 15, 0, 123000000, time.UTC)},
		{`"2026-11-26T11:15:00+02:00"`, time.Date(2026, 11, 26, 9, 15, 0, 0, time.UTC)},
		{`"2026-11-26"`, time.Date(2026, 11, 26, 0, 0, 0, 0, time.UTC)},
		{`null`, time.Time{}},
	}
	for _, tc := range cases {
		var v traderspy.Time
		if err := json.Unmarshal([]byte(tc.in), &v); err != nil {
			t.Fatalf("%s: %v", tc.in, err)
		}
		if !v.Equal(tc.want) {
			t.Fatalf("%s: got %v, want %v", tc.in, v.Time, tc.want)
		}
	}
	var v traderspy.Time
	if err := json.Unmarshal([]byte(`"not a time"`), &v); err == nil {
		t.Fatal("bad time should fail")
	}
}

func TestUnixMillis(t *testing.T) {
	var m traderspy.UnixMillis
	if err := json.Unmarshal([]byte(`1764157200000`), &m); err != nil {
		t.Fatal(err)
	}
	if m.Unix() != 1764157200 {
		t.Fatalf("unix = %d", m.Unix())
	}
	// seconds tolerated
	if err := json.Unmarshal([]byte(`1764157200`), &m); err != nil {
		t.Fatal(err)
	}
	if m.Unix() != 1764157200 {
		t.Fatalf("unix(seconds) = %d", m.Unix())
	}
	for input, want := range map[string]int64{
		`"1764157200.125"`: 1764157200125,
		`-1764157200001`:   -1764157200001,
		`9007199254740993`: 9007199254740993,
	} {
		if err := json.Unmarshal([]byte(input), &m); err != nil || m.UnixMilli() != want {
			t.Fatalf("%s: got %d, %v, want %d", input, m.UnixMilli(), err, want)
		}
	}
	if err := json.Unmarshal([]byte(`"NaN"`), &m); err == nil {
		t.Fatal("NaN timestamp should fail")
	}
	// null
	if err := json.Unmarshal([]byte(`null`), &m); err != nil || !m.IsZero() {
		t.Fatalf("null: %v %v", m, err)
	}
}

func TestTimeRoundTrip(t *testing.T) {
	in := traderspy.Signal{ID: "x", CreatedAt: traderspy.Time{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out traderspy.Signal
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if !out.CreatedAt.Equal(in.CreatedAt.Time) {
		t.Fatalf("round-trip = %v", out.CreatedAt)
	}
}
