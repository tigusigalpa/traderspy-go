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

func TestPrices(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		serveFixture(w, "prices.json")
	})
	c := s.client(t)
	out, _, err := c.Market.Prices(context.Background(), []string{"btc", "ETH", "solusdt"})
	if err != nil {
		t.Fatal(err)
	}
	got := s.last()
	if got.Path != "/v1/market/prices" {
		t.Fatalf("path = %s", got.Path)
	}
	if got.Query.Get("symbols") != "BTCUSDT,ETHUSDT,SOLUSDT" {
		t.Fatalf("symbols = %q", got.Query.Get("symbols"))
	}
	if len(out.Data) != 3 || out.Data[0].Price != 97412.5 {
		t.Fatalf("prices = %+v", out.Data)
	}
	if *out.Data[0].Change24hPct != 1.84 {
		t.Fatalf("change = %v", *out.Data[0].Change24hPct)
	}
}

func TestPricesMapForm(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"ETHUSDT":{"price":3300.1,"change24hPct":-1.2},"BTCUSDT":{"price":97000.0}}}`))
	})
	c := s.client(t)
	out, _, err := c.Market.Prices(context.Background(), []string{"BTC", "ETH"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Data) != 2 || out.Data[0].Symbol != "BTCUSDT" || out.Data[1].Symbol != "ETHUSDT" {
		t.Fatalf("map form not normalized: %+v", out.Data)
	}
}

func TestPricesValidation(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {})
	c := s.client(t)
	if _, _, err := c.Market.Prices(context.Background(), nil); err == nil {
		t.Fatal("empty list should fail")
	}
	tooMany := make([]string, 21)
	for i := range tooMany {
		tooMany[i] = "X"
	}
	var ve *traderspy.ValidationError
	if _, _, err := c.Market.Prices(context.Background(), tooMany); !errors.As(err, &ve) {
		t.Fatalf("want *ValidationError, got %v", err)
	}
	if s.calls() != 0 {
		t.Fatalf("calls = %d", s.calls())
	}
}

func TestCandles(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		serveFixture(w, "candles.json")
	})
	c := s.client(t)
	out, _, err := c.Market.Candles(context.Background(), &traderspy.CandlesOptions{
		Symbol:   "btc",
		Interval: traderspy.Interval1h,
		Limit:    200,
	})
	if err != nil {
		t.Fatal(err)
	}
	q := s.last().Query
	if q.Get("symbol") != "BTCUSDT" || q.Get("interval") != "1h" || q.Get("limit") != "200" {
		t.Fatalf("query = %v", q)
	}
	if len(out.Data) != 3 {
		t.Fatalf("candles = %d", len(out.Data))
	}
	cd := out.Data[0]
	if cd.Open != 96800.1 || cd.High != 97250.0 || cd.Close != 97180.2 || cd.Volume != 812.44 {
		t.Fatalf("candle = %+v", cd)
	}
	if cd.Time.Unix() != 1764136800 {
		t.Fatalf("time = %v", cd.Time)
	}
}

func TestCandlesArrayForm(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[[1764136800000,96800.1,97250.0,96410.5,97180.2,812.44]]}`))
	})
	c := s.client(t)
	out, _, err := c.Market.Candles(context.Background(), &traderspy.CandlesOptions{Symbol: "BTC"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Data) != 1 || out.Data[0].Open != 96800.1 || out.Data[0].Volume != 812.44 {
		t.Fatalf("array form: %+v", out.Data)
	}
	if out.Data[0].Time.Unix() != 1764136800 {
		t.Fatalf("time = %v", out.Data[0].Time)
	}
}

func TestCandlesValidation(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {})
	c := s.client(t)
	for _, opts := range []*traderspy.CandlesOptions{
		nil,
		{Symbol: ""},
		{Symbol: "BTC", Interval: "3m"},
		{Symbol: "BTC", Limit: 501},
	} {
		var ve *traderspy.ValidationError
		if _, _, err := c.Market.Candles(context.Background(), opts); !errors.As(err, &ve) {
			t.Fatalf("opts %+v: want *ValidationError, got %v", opts, err)
		}
	}
}

func TestSymbols(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		serveFixture(w, "symbols.json")
	})
	c := s.client(t)
	out, _, err := c.Market.Symbols(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Symbols) != 5 || out.Symbols[0] != "BTCUSDT" {
		t.Fatalf("symbols = %v", out.Symbols)
	}
}

func TestSymbolsObjectForm(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"symbols":[{"symbol":"BTCUSDT"},{"name":"ETHUSDT"}]}`))
	})
	c := s.client(t)
	out, _, err := c.Market.Symbols(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Symbols) != 2 || out.Symbols[1] != "ETHUSDT" {
		t.Fatalf("object form: %v", out.Symbols)
	}
}

func TestDerivatives(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request, call int) {
		serveFixture(w, "derivatives.json")
	})
	c := s.client(t)
	out, _, err := c.Market.Derivatives(context.Background(), []string{"BTC", "ETH"})
	if err != nil {
		t.Fatal(err)
	}
	if s.last().Query.Get("symbols") != "BTCUSDT,ETHUSDT" {
		t.Fatalf("symbols = %q", s.last().Query.Get("symbols"))
	}
	if len(out.Data) != 2 {
		t.Fatalf("derivatives = %d", len(out.Data))
	}
	d := out.Data[0]
	if *d.FundingRate != 0.0001 || *d.OpenInterest != 51240.7 || d.Note == "" {
		t.Fatalf("derivative = %+v", d)
	}
	if _, _, err := c.Market.Derivatives(context.Background(), []string{"a", "b", "c", "d", "e", "f"}); err == nil {
		t.Fatal("6 symbols should fail (max 5)")
	}
}
