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

func TestStrategiesList(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		serveFixture(w, "strategies.json")
	})
	c := s.client(t)
	out, _, err := c.Strategies.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if s.last().Path != "/v1/strategies" {
		t.Fatalf("path = %s", s.last().Path)
	}
	if len(out.Data) != 1 || out.Data[0].ID != "str_9f2c41" || !out.Data[0].Enabled {
		t.Fatalf("strategies = %+v", out.Data)
	}
	if len(out.Data[0].Coins) != 3 {
		t.Fatalf("coins = %v", out.Data[0].Coins)
	}
}

func TestStrategyAlerts(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		serveFixture(w, "strategy_alerts.json")
	})
	c := s.client(t)
	since := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	out, resp, err := c.Strategies.Alerts(context.Background(), &traderspy.StrategyAlertsOptions{
		Limit:      50,
		Offset:     10,
		Since:      since,
		StrategyID: "str_9f2c41",
		Coin:       "SOLUSDT",
	})
	if err != nil {
		t.Fatal(err)
	}
	q := s.last().Query
	if s.last().Path != "/v1/strategies/alerts" {
		t.Fatalf("path = %s", s.last().Path)
	}
	if q.Get("limit") != "50" || q.Get("offset") != "10" || q.Get("strategyId") != "str_9f2c41" ||
		q.Get("coin") != "SOLUSDT" || q.Get("since") != "2026-11-20T00:00:00Z" {
		t.Fatalf("query = %v", q)
	}
	if len(out.Data) != 1 || out.Data[0].StrategyID != "str_9f2c41" || len(out.Data[0].Rules) != 2 {
		t.Fatalf("alerts = %+v", out.Data)
	}
	if resp.Pagination == nil || resp.Pagination.HasMore {
		t.Fatalf("pagination = %+v", resp.Pagination)
	}
}

func TestStrategyAlertsValidation(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {})
	c := s.client(t)
	for _, opts := range []*traderspy.StrategyAlertsOptions{
		{Limit: 101}, {Offset: 10001},
	} {
		var ve *traderspy.ValidationError
		if _, _, err := c.Strategies.Alerts(context.Background(), opts); !errors.As(err, &ve) {
			t.Fatalf("opts %+v: %v", opts, err)
		}
	}
	if s.calls() != 0 {
		t.Fatalf("calls = %d", s.calls())
	}
}

func TestAllAlertsIterator(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		serveFixture(w, "strategy_alerts.json") // hasMore=false → single page
	})
	c := s.client(t)
	n := 0
	for a, err := range c.Strategies.AllAlerts(context.Background(), nil) {
		if err != nil {
			t.Fatal(err)
		}
		if a.Coin != "SOLUSDT" {
			t.Fatalf("alert = %+v", a)
		}
		n++
	}
	if n != 1 || s.calls() != 1 {
		t.Fatalf("n=%d calls=%d", n, s.calls())
	}
}
