// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Igor Sazonov

package traderspy_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	traderspy "github.com/tigusigalpa/traderspy-go"
)

func TestTopTraders(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		serveFixture(w, "traders.json")
	})
	c := s.client(t)
	out, _, err := c.SmartMoney.TopTraders(context.Background(), &traderspy.TopTradersOptions{
		Source:      traderspy.SourceHyperliquid,
		TimeRange:   traderspy.TimeRange7D,
		RankingType: traderspy.RankingPNL,
		SortBy:      traderspy.SortByScore,
		Order:       traderspy.OrderAsc,
		Limit:       100,
	})
	if err != nil {
		t.Fatalf("TopTraders: %v", err)
	}
	got := s.last()
	if got.Path != "/v1/smart-money/traders" {
		t.Fatalf("path = %s", got.Path)
	}
	q := got.Query
	if q.Get("source") != "hyperliquid" || q.Get("timeRange") != "7D" ||
		q.Get("rankingType") != "PNL" || q.Get("sortBy") != "SCORE" ||
		q.Get("order") != "ASC" || q.Get("limit") != "100" {
		t.Fatalf("query = %v", q)
	}
	if len(out.Data) != 2 || out.Data[0].TopTraderID != "hl_0xa9f21bd3c" {
		t.Fatalf("traders = %+v", out.Data)
	}
	if *out.Data[0].SmartScore != 96.4 {
		t.Fatalf("SmartScore = %v", *out.Data[0].SmartScore)
	}
}

func TestEliteTraders(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		serveFixture(w, "traders.json")
	})
	c := s.client(t)
	out, _, err := c.SmartMoney.EliteTraders(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if s.last().Path != "/v1/smart-money/traders/elite" {
		t.Fatalf("path = %s", s.last().Path)
	}
	if len(out.Data) != 2 {
		t.Fatalf("len = %d", len(out.Data))
	}
}

func TestTraderProfile(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		serveFixture(w, "trader_profile.json")
	})
	c := s.client(t)
	out, _, err := c.SmartMoney.Trader(context.Background(), traderspy.SourceHyperliquid, "hl_0xa9f21bd3c", &traderspy.TraderOptions{
		TimeRange:   traderspy.TimeRange24h,
		RankingType: traderspy.RankingROI,
	})
	if err != nil {
		t.Fatal(err)
	}
	got := s.last()
	if got.Path != "/v1/smart-money/traders/hyperliquid/hl_0xa9f21bd3c" {
		t.Fatalf("path = %s", got.Path)
	}
	if got.Query.Get("timeRange") != "24h" || got.Query.Get("rankingType") != "ROI" {
		t.Fatalf("query = %v", got.Query)
	}
	if out.TopTraderID != "hl_0xa9f21bd3c" || len(out.Positions) != 1 {
		t.Fatalf("profile = %+v", out)
	}
	if out.Positions[0].UnrealizedPnL == nil || *out.Positions[0].UnrealizedPnL != 28310.55 {
		t.Fatalf("position = %+v", out.Positions[0])
	}
}

func TestTraderPathValidation(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		serveFixture(w, "trader_profile.json")
	})
	c := s.client(t)
	for _, src := range []traderspy.Source{traderspy.SourceAll, "kraken", ""} {
		if _, _, err := c.SmartMoney.Trader(context.Background(), src, "x", nil); err == nil {
			t.Fatalf("source %q should be rejected", src)
		}
	}
	if _, _, err := c.SmartMoney.Trader(context.Background(), traderspy.SourceBinance, "", nil); err == nil {
		t.Fatal("empty traderId should be rejected")
	}
	if _, _, err := c.SmartMoney.TraderHistory(context.Background(), "all", "x", nil); err == nil {
		t.Fatal("source=all should be rejected for history")
	}
	if s.calls() != 0 {
		t.Fatalf("calls = %d, want 0", s.calls())
	}
}

func TestTraderHistory(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		serveFixture(w, "trader_history.json")
	})
	c := s.client(t)
	out, resp, err := c.SmartMoney.TraderHistory(context.Background(), traderspy.SourceBinance, "bn_84f21c", &traderspy.TraderHistoryOptions{Page: 3, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	got := s.last()
	if got.Path != "/v1/smart-money/traders/binance/bn_84f21c/history" {
		t.Fatalf("path = %s", got.Path)
	}
	if got.Query.Get("page") != "3" || got.Query.Get("limit") != "10" {
		t.Fatalf("query = %v", got.Query)
	}
	if len(out.Data) != 1 || out.Data[0].PnL == nil || *out.Data[0].PnL != 51230.0 {
		t.Fatalf("history = %+v", out.Data)
	}
	if resp.Pagination == nil || resp.Pagination.Page != 1 {
		t.Fatalf("pagination = %+v", resp.Pagination)
	}
	if _, _, err := c.SmartMoney.TraderHistory(context.Background(), traderspy.SourceBinance, "x", &traderspy.TraderHistoryOptions{Page: 501}); err == nil {
		t.Fatal("page 501 should fail validation")
	}
}

func TestAllTraderHistoryIterator(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		if r.URL.Query().Get("page") == "2" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[{"source":"hyperliquid","topTraderId":"t1","symbol":"BTCUSDT","side":"long","size":1,"isOpen":false,"sortDate":1764060000000,"pnl":10}],"pagination":{"limit":1,"total":2,"hasMore":false,"page":2}}`))
			return
		}
		serveFixture(w, "trader_history.json")
	})
	c := s.client(t)
	n := 0
	for p, err := range c.SmartMoney.AllTraderHistory(context.Background(), traderspy.SourceHyperliquid, "t1", &traderspy.TraderHistoryOptions{Limit: 1}) {
		if err != nil {
			t.Fatal(err)
		}
		if p == nil {
			t.Fatal("nil position")
		}
		n++
	}
	if n != 2 || s.calls() != 2 {
		t.Fatalf("n=%d calls=%d", n, s.calls())
	}
}

func TestPositions(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		serveFixture(w, "positions.json")
	})
	c := s.client(t)
	out, resp, err := c.SmartMoney.Positions(context.Background(), &traderspy.PositionsOptions{
		Status: traderspy.StatusOpen,
		Source: traderspy.SourceBinance,
		Symbol: "BTCUSDT",
		Limit:  30,
		Offset: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	got := s.last()
	if got.Path != "/v1/smart-money/positions" {
		t.Fatalf("path = %s", got.Path)
	}
	q := got.Query
	if q.Get("status") != "open" || q.Get("source") != "binance" || q.Get("symbol") != "BTCUSDT" ||
		q.Get("limit") != "30" || q.Get("offset") != "60" {
		t.Fatalf("query = %v", q)
	}
	if len(out.Data) != 2 {
		t.Fatalf("positions = %d", len(out.Data))
	}
	p := out.Data[0]
	if p.Side != "short" || p.IsOpen || p.LastEvent != "partial_close" {
		t.Fatalf("position = %+v", p)
	}
	if p.SortDate.IsZero() || p.SortDate.Unix() != 1764153600 {
		t.Fatalf("SortDate = %v", p.SortDate)
	}
	if resp.Pagination == nil || resp.Pagination.Total != 5409 {
		t.Fatalf("pagination = %+v", resp.Pagination)
	}
}

func TestAllPositionsIterator(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		if r.URL.Query().Get("offset") == "2" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[],"pagination":{"limit":2,"total":2,"hasMore":false,"offset":2}}`))
			return
		}
		serveFixture(w, "positions.json")
	})
	c := s.client(t)
	n := 0
	for _, err := range c.SmartMoney.AllPositions(context.Background(), &traderspy.PositionsOptions{Limit: 2}) {
		if err != nil {
			t.Fatal(err)
		}
		n++
	}
	if n != 2 {
		t.Fatalf("n = %d", n)
	}
}

func TestSmartMoneyStats(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		serveFixture(w, "smart_money_stats.json")
	})
	c := s.client(t)
	out, _, err := c.SmartMoney.Stats(context.Background(), &traderspy.SmartMoneyStatsOptions{
		Source: traderspy.SourceBybit,
		Period: traderspy.Period8h,
	})
	if err != nil {
		t.Fatal(err)
	}
	q := s.last().Query
	if q.Get("source") != "bybit" || q.Get("period") != "8h" {
		t.Fatalf("query = %v", q)
	}
	if out.OpenPositions != 3182 || out.LongCount != 1977 || *out.WinRatePct != 58.4 {
		t.Fatalf("stats = %+v", out)
	}
}

func TestExchanges(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		serveFixture(w, "exchanges.json")
	})
	c := s.client(t)
	out, _, err := c.SmartMoney.Exchanges(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if s.last().Path != "/v1/smart-money/exchanges" {
		t.Fatalf("path = %s", s.last().Path)
	}
	if len(out.Data) != 4 || out.Data[1].ID != "hyperliquid" {
		t.Fatalf("exchanges = %+v", out.Data)
	}
}

func TestSmartMoneyValidationErrors(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {})
	c := s.client(t)
	checks := []error{}
	_, _, err := c.SmartMoney.TopTraders(context.Background(), &traderspy.TopTradersOptions{Limit: 201})
	checks = append(checks, err)
	_, _, err = c.SmartMoney.TopTraders(context.Background(), &traderspy.TopTradersOptions{Source: "ftx"})
	checks = append(checks, err)
	_, _, err = c.SmartMoney.Positions(context.Background(), &traderspy.PositionsOptions{Status: "pending"})
	checks = append(checks, err)
	_, _, err = c.SmartMoney.Stats(context.Background(), &traderspy.SmartMoneyStatsOptions{Period: "1h"})
	checks = append(checks, err)
	for i, err := range checks {
		var ve *traderspy.ValidationError
		if !errors.As(err, &ve) {
			t.Fatalf("check %d: want *ValidationError, got %v", i, err)
		}
	}
	if s.calls() != 0 {
		t.Fatalf("calls = %d", s.calls())
	}
}
