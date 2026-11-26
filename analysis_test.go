// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Igor Sazonov

package traderspy_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	traderspy "github.com/tigusigalpa/traderspy-go"
)

func TestTechnicals(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		serveFixture(w, "technicals.json")
	})
	c := s.client(t)
	out, _, err := c.Analysis.Technicals(context.Background(), &traderspy.TechnicalsOptions{
		Symbols:    []string{"btc", "eth"},
		Intervals:  []traderspy.Interval{traderspy.Interval1h, traderspy.Interval4h},
		Indicators: []string{"rsi", "macd"},
		History:    10,
		Periods:    map[string]any{"rsi": 7, "ema": []int{9, 21, 55}},
	})
	if err != nil {
		t.Fatal(err)
	}
	q := s.last().Query
	if q.Get("symbols") != "BTCUSDT,ETHUSDT" {
		t.Fatalf("symbols = %q", q.Get("symbols"))
	}
	if q.Get("intervals") != "1h,4h" {
		t.Fatalf("intervals = %q", q.Get("intervals"))
	}
	if q.Get("indicators") != "rsi,macd" || q.Get("history") != "10" {
		t.Fatalf("query = %v", q)
	}
	var periods map[string]any
	if err := json.Unmarshal([]byte(q.Get("periods")), &periods); err != nil {
		t.Fatalf("periods not JSON: %q", q.Get("periods"))
	}
	if periods["rsi"] != 7.0 {
		t.Fatalf("periods = %v", periods)
	}

	if len(out.Data) != 1 || out.Data[0].Symbol != "BTCUSDT" {
		t.Fatalf("data = %+v", out.Data)
	}
	st := out.Data[0]
	if len(st.Intervals) != 2 || st.Intervals[0].Interval != "1h" {
		t.Fatalf("intervals = %+v", st.Intervals)
	}
	rsi := st.Intervals[0].Indicators["rsi"]
	if rsi["value"] != 52.4 {
		t.Fatalf("rsi = %v", rsi)
	}
	if st.Confluence["verdict"] != "bullish" {
		t.Fatalf("confluence = %v", st.Confluence)
	}
}

func TestTechnicalsSingleSymbol(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		serveFixture(w, "technicals.json")
	})
	c := s.client(t)
	_, _, err := c.Analysis.Technicals(context.Background(), &traderspy.TechnicalsOptions{
		Symbol:   "btc",
		Interval: traderspy.Interval4h,
	})
	if err != nil {
		t.Fatal(err)
	}
	q := s.last().Query
	if q.Get("symbol") != "BTCUSDT" || q.Get("interval") != "4h" {
		t.Fatalf("query = %v", q)
	}
}

func TestTechnicalsValidation(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {})
	c := s.client(t)
	bad := []*traderspy.TechnicalsOptions{
		nil,
		{},
		{Symbol: "BTC", Symbols: []string{"ETH"}},
		{Symbols: []string{"a", "b", "c", "d"}},
		{Symbol: "BTC", Interval: "7h"},
		{Symbol: "BTC", Intervals: []traderspy.Interval{"1h", "4h", "1d", "1m"}},
		{Symbol: "BTC", History: 21},
	}
	for i, opts := range bad {
		var ve *traderspy.ValidationError
		if _, _, err := c.Analysis.Technicals(context.Background(), opts); !errors.As(err, &ve) {
			t.Fatalf("case %d: want *ValidationError, got %v", i, err)
		}
	}
	if s.calls() != 0 {
		t.Fatalf("calls = %d", s.calls())
	}
}

func TestScreener(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		serveFixture(w, "screener.json")
	})
	c := s.client(t)
	out, _, err := c.Analysis.Screener(context.Background(), &traderspy.ScreenerOptions{
		Interval: traderspy.Interval4h,
		Conditions: []traderspy.Condition{
			{Metric: traderspy.MetricRSI, Op: traderspy.OpLT, Value: 30},
			{Metric: traderspy.MetricPriceVsEMA, Op: traderspy.OpGT, Value: 0, Period: 200},
		},
		Universe:  100,
		Assets:    traderspy.AssetsCrypto,
		Limit:     50,
		SortBy:    traderspy.ScreenerSortMetric,
		SortOrder: traderspy.SortOrderAsc,
	})
	if err != nil {
		t.Fatal(err)
	}
	got := s.last()
	if got.Method != http.MethodPost || got.Path != "/v1/screener" {
		t.Fatalf("%s %s", got.Method, got.Path)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(got.Body), &body); err != nil {
		t.Fatalf("body not JSON: %v", err)
	}
	if body["interval"] != "4h" || body["universe"] != 100.0 || body["assets"] != "crypto" ||
		body["limit"] != 50.0 || body["sortBy"] != "metric" || body["sortOrder"] != "asc" {
		t.Fatalf("body = %s", got.Body)
	}
	conds, _ := body["conditions"].([]any)
	if len(conds) != 2 {
		t.Fatalf("conditions = %v", body["conditions"])
	}
	c2, _ := conds[1].(map[string]any)
	if c2["metric"] != "priceVsEma" || c2["op"] != "gt" || c2["period"] != 200.0 {
		t.Fatalf("condition = %v", c2)
	}

	if out.Matched != 3 || len(out.Results) != 2 || out.Universe.Scanned != 100 {
		t.Fatalf("result = %+v", out)
	}
	row := out.Results[0]
	if row.Symbol != "DOGEUSDT" || *row.RSI14 != 24.6 || row.Trend != "below200" {
		t.Fatalf("row = %+v", row)
	}
}

func TestScreenerExplicitSymbols(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		serveFixture(w, "screener.json")
	})
	c := s.client(t)
	_, _, err := c.Analysis.Screener(context.Background(), &traderspy.ScreenerOptions{
		Symbols: []string{"btc", "eth"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	_ = json.Unmarshal([]byte(s.last().Body), &body)
	syms, _ := body["symbols"].([]any)
	if len(syms) != 2 || syms[0] != "BTCUSDT" {
		t.Fatalf("body = %s", s.last().Body)
	}
}

func TestScreenerValidation(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {})
	c := s.client(t)
	bad := []*traderspy.ScreenerOptions{
		{Conditions: make([]traderspy.Condition, 4)},
		{Conditions: []traderspy.Condition{{Metric: "nope", Op: traderspy.OpLT, Value: 1}}},
		{Conditions: []traderspy.Condition{{Metric: traderspy.MetricRSI, Op: "eq", Value: 1}}},
		{Conditions: []traderspy.Condition{{Metric: traderspy.MetricRSI, Op: traderspy.OpGT, Value: 1, Period: 1}}},
		{Universe: 4},
		{Universe: 101},
		{Limit: 51},
		{Assets: "stocks"},
		{SortBy: "price"},
		{SortOrder: "up"},
	}
	for i, opts := range bad {
		var ve *traderspy.ValidationError
		if _, _, err := c.Analysis.Screener(context.Background(), opts); !errors.As(err, &ve) {
			t.Fatalf("case %d: want *ValidationError, got %v", i, err)
		}
	}
	if s.calls() != 0 {
		t.Fatalf("calls = %d", s.calls())
	}
}

func TestBacktest(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		serveFixture(w, "backtest.json")
	})
	c := s.client(t)
	out, _, err := c.Analysis.Backtest(context.Background(), &traderspy.BacktestOptions{
		Symbol:   "btc",
		Interval: traderspy.Interval4h,
		Conditions: []traderspy.Condition{
			{Metric: traderspy.MetricRSI, Op: traderspy.OpLT, Value: 30},
		},
		Horizons: []int{1, 6, 18},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := s.last()
	if got.Method != http.MethodPost || got.Path != "/v1/backtest" {
		t.Fatalf("%s %s", got.Method, got.Path)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(got.Body), &body); err != nil {
		t.Fatalf("body not JSON: %v", err)
	}
	if body["symbol"] != "BTCUSDT" || body["interval"] != "4h" {
		t.Fatalf("body = %s", got.Body)
	}
	if horizons, _ := body["horizons"].([]any); len(horizons) != 3 {
		t.Fatalf("horizons = %v", body["horizons"])
	}

	if out.Occurrences != 41 || out.Bars != 1000 || len(out.Horizons) != 3 {
		t.Fatalf("result = %+v", out)
	}
	if *out.Horizons[1].EdgePct != 1.02 {
		t.Fatalf("horizon = %+v", out.Horizons[1])
	}
	if out.Regimes == nil || len(out.Regimes.Horizons) != 1 {
		t.Fatal("no regimes")
	}
	reg := out.Regimes.Horizons[0]
	if reg.Hours != 24 || reg.Above.Samples != 29 || *reg.Above.EdgePct != 2.74 {
		t.Fatalf("regime = %+v", reg)
	}
}

func TestBacktestValidation(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {})
	c := s.client(t)
	good := []traderspy.Condition{{Metric: traderspy.MetricRSI, Op: traderspy.OpLT, Value: 30}}
	bad := []*traderspy.BacktestOptions{
		nil,
		{Conditions: good},
		{Symbol: "BTC"},
		{Symbol: "BTC", Conditions: make([]traderspy.Condition, 4)},
		{Symbol: "BTC", Conditions: good, Horizons: []int{0}},
		{Symbol: "BTC", Conditions: good, Horizons: []int{1, 2, 3, 4, 5}},
		{Symbol: "BTC", Conditions: good, Interval: "2h"},
	}
	for i, opts := range bad {
		var ve *traderspy.ValidationError
		if _, _, err := c.Analysis.Backtest(context.Background(), opts); !errors.As(err, &ve) {
			t.Fatalf("case %d: want *ValidationError, got %v", i, err)
		}
	}
	if s.calls() != 0 {
		t.Fatalf("calls = %d", s.calls())
	}
}
