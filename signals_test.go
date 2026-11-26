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

func TestSignalsList(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		serveFixture(w, "signals_list.json")
	})
	c := s.client(t)
	since := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)

	out, resp, err := c.Signals.List(context.Background(), &traderspy.ListSignalsOptions{
		Limit:      25,
		Offset:     50,
		Importance: traderspy.ImportanceHigh,
		Coin:       "BTC",
		Strategy:   "momentum",
		Timeframe:  traderspy.Timeframe4h,
		Since:      since,
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	got := s.last()
	if got.Method != http.MethodGet || got.Path != "/v1/signals" {
		t.Fatalf("%s %s", got.Method, got.Path)
	}
	q := got.Query
	if q.Get("limit") != "25" || q.Get("offset") != "50" || q.Get("importance") != "high" ||
		q.Get("coin") != "BTC" || q.Get("strategy") != "momentum" || q.Get("timeframe") != "4h" {
		t.Fatalf("query = %v", q)
	}
	if q.Get("since") != "2026-11-25T12:00:00Z" {
		t.Fatalf("since = %q", q.Get("since"))
	}

	if len(out.Data) != 2 {
		t.Fatalf("len(Data) = %d", len(out.Data))
	}
	sig := out.Data[0]
	if sig.ID != "sig_01J8XZQ2" || sig.Action != traderspy.ActionBuy || sig.Importance != traderspy.ImportanceHigh {
		t.Fatalf("signal = %+v", sig)
	}
	if sig.CreatedAt.IsZero() || sig.CreatedAt.UTC().Format("2006-01-02") != "2026-11-26" {
		t.Fatalf("CreatedAt = %v", sig.CreatedAt)
	}
	if len(sig.Targets) != 3 || sig.Targets[2].Type != "sl" {
		t.Fatalf("Targets = %+v", sig.Targets)
	}
	if resp.Pagination == nil || !resp.Pagination.HasMore || resp.Pagination.Total != 4312 {
		t.Fatalf("Pagination = %+v", resp.Pagination)
	}
}

func TestSignalTargetPrices(t *testing.T) {
	buy := traderspy.Signal{
		Action: traderspy.ActionBuy,
		Price:  100,
		Targets: []traderspy.SignalTarget{
			{Type: "tp1", Pct: 2}, {Type: "tp2", Pct: 5}, {Type: "sl", Pct: 3},
		},
	}
	levels := buy.TargetPrices()
	if levels["tp1"] != 102 || levels["tp2"] != 105 || levels["sl"] != 97 {
		t.Fatalf("buy levels = %v", levels)
	}
	sell := traderspy.Signal{
		Action: traderspy.ActionSell,
		Price:  100,
		Targets: []traderspy.SignalTarget{
			{Type: "tp1", Pct: 2}, {Type: "sl", Pct: 3},
		},
	}
	levels = sell.TargetPrices()
	if levels["tp1"] != 98 || levels["sl"] != 103 {
		t.Fatalf("sell levels = %v", levels)
	}
}

func TestSignalsListValidation(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		serveFixture(w, "signals_list.json")
	})
	c := s.client(t)
	for _, opts := range []*traderspy.ListSignalsOptions{
		{Limit: 0},
		{Limit: 51},
		{Offset: -1},
		{Offset: 10001},
		{Importance: "extreme"},
		{Timeframe: "3m"},
	} {
		if opts.Limit == 0 {
			continue // 0 means unset: valid
		}
		_, _, err := c.Signals.List(context.Background(), opts)
		var ve *traderspy.ValidationError
		if !errors.As(err, &ve) {
			t.Fatalf("opts %+v: want *ValidationError, got %v", opts, err)
		}
	}
	if s.calls() != 0 {
		t.Fatalf("validation must stop requests before HTTP, got %d calls", s.calls())
	}
}

func TestSignalsGet(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		serveFixture(w, "signal_detail.json")
	})
	c := s.client(t)
	out, _, err := c.Signals.Get(context.Background(), "sig_01J8XZQ2")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if s.last().Path != "/v1/signals/sig_01J8XZQ2" {
		t.Fatalf("path = %s", s.last().Path)
	}
	if out.ID != "sig_01J8XZQ2" || out.LivePrice == nil || *out.LivePrice != 97510.2 {
		t.Fatalf("detail = %+v", out)
	}
	if out.AIReview == "" || out.Indicators["rsi14"] != 61.4 {
		t.Fatalf("detail extras = %+v", out)
	}
	if _, _, err := c.Signals.Get(context.Background(), ""); err == nil {
		t.Fatal("empty id should fail validation")
	}
}

func TestSignalsStats(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		serveFixture(w, "signals_stats.json")
	})
	c := s.client(t)
	out, _, err := c.Signals.Stats(context.Background(), &traderspy.SignalStatsOptions{
		Period:   traderspy.Period7d,
		Strategy: "momentum",
	})
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	q := s.last().Query
	if q.Get("period") != "7d" || q.Get("strategy") != "momentum" {
		t.Fatalf("query = %v", q)
	}
	if out.TotalSignals != 42 || out.TargetsHit != 19 || *out.WinRatePct != 63.33 {
		t.Fatalf("stats = %+v", out)
	}
	if len(out.Strategies) != 2 || out.Strategies[0].StrategyName != "Momentum Breakout AI" {
		t.Fatalf("strategies = %+v", out.Strategies)
	}
}

func TestSignalsAllIterator(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		switch r.URL.Query().Get("offset") {
		case "2":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[{"id":"sig_3","action":"sell","coin":"SOLUSDT","timeframe":"1h","price":187.9,"createdAt":"2026-11-25T10:00:00Z","targets":[],"strategyName":"s","resolutionStatus":"open","importance":"low","signalStrength":0.5,"url":"u"}],"pagination":{"limit":2,"total":3,"hasMore":false,"offset":2}}`))
		default:
			serveFixture(w, "signals_list.json")
		}
	})
	c := s.client(t)

	var ids []string
	for sig, err := range c.Signals.All(context.Background(), &traderspy.ListSignalsOptions{Limit: 2}) {
		if err != nil {
			t.Fatalf("iterator: %v", err)
		}
		ids = append(ids, sig.ID)
	}
	if len(ids) != 3 || ids[2] != "sig_3" {
		t.Fatalf("ids = %v", ids)
	}
	if s.calls() != 2 {
		t.Fatalf("calls = %d, want 2", s.calls())
	}
	// second page must have been requested with offset=2
	if s.last().Query.Get("offset") != "2" {
		t.Fatalf("offset = %q", s.last().Query.Get("offset"))
	}
}

func TestSignalsAllIteratorError(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		serveError(w, http.StatusUnauthorized, traderspy.ErrCodeInvalidAPIKey, "bad key")
	})
	c := s.client(t)
	n := 0
	for _, err := range c.Signals.All(context.Background(), nil) {
		if err == nil {
			t.Fatal("expected error")
		}
		var ae *traderspy.AuthError
		if !errors.As(err, &ae) {
			t.Fatalf("want *AuthError, got %T", err)
		}
		n++
	}
	if n != 1 {
		t.Fatalf("yielded %d errors", n)
	}
}

func TestSignalsAllEarlyBreak(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		serveFixture(w, "signals_list.json")
	})
	c := s.client(t)
	n := 0
	for _, err := range c.Signals.All(context.Background(), nil) {
		if err != nil {
			t.Fatal(err)
		}
		n++
		break // stop after the first
	}
	if n != 1 || s.calls() != 1 {
		t.Fatalf("n=%d calls=%d", n, s.calls())
	}
}
