// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Igor Sazonov

package traderspy_test

import (
	"encoding/json"
	"math"
	"reflect"
	"testing"

	traderspy "github.com/tigusigalpa/traderspy-go"
)

// Every model with a custom MarshalJSON must round-trip.
func TestMarshalRoundTrips(t *testing.T) {
	fixtures := map[string]json.Marshaler{
		"signals_list.json":      &traderspy.Signal{},
		"signal_detail.json":     &traderspy.SignalDetail{},
		"traders.json":           &traderspy.Trader{},
		"positions.json":         &traderspy.Position{},
		"trader_profile.json":    &traderspy.TraderProfile{},
		"smart_money_stats.json": &traderspy.SmartMoneyStats{},
		"exchanges.json":         &traderspy.Exchange{},
		"prices.json":            &traderspy.Price{},
		"candles.json":           &traderspy.Candle{},
		"derivatives.json":       &traderspy.Derivative{},
		"screener.json":          &traderspy.ScreenerResult{},
		"backtest.json":          &traderspy.BacktestResult{},
		"strategies.json":        &traderspy.Strategy{},
		"strategy_alerts.json":   &traderspy.StrategyAlert{},
	}
	for name, v := range fixtures {
		raw := fixtureBytes(name)
		// list fixtures: decode the first element of "data"
		var probe struct {
			Data json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(raw, &probe); err == nil && len(probe.Data) > 0 && probe.Data[0] == '[' {
			var items []json.RawMessage
			if err := json.Unmarshal(probe.Data, &items); err == nil && len(items) > 0 {
				raw = items[0]
			}
		}
		if _, err := v.MarshalJSON(); err != nil {
			t.Fatalf("%s: MarshalJSON on zero value: %v", name, err)
		}
		if err := json.Unmarshal(raw, v); err != nil {
			t.Fatalf("%s: unmarshal: %v", name, err)
		}
		out, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("%s: marshal: %v", name, err)
		}
		if len(out) < 10 {
			t.Fatalf("%s: suspicious marshal output %s", name, out)
		}
	}
}

// Fields the SDK does not know yet must land in Extra and survive a marshal
// round-trip (forward compatibility).
func TestExtraFieldsPreserved(t *testing.T) {
	in := []byte(`{"id":"s1","action":"buy","coin":"BTCUSDT","timeframe":"4h","price":1.5,"targets":[],"strategyName":"s","importance":"high","signalStrength":0.9,"resolutionStatus":"open","createdAt":"2026-11-26T09:15:00Z","url":"u","futureField":{"nested":true}}`)
	var sig traderspy.Signal
	if err := json.Unmarshal(in, &sig); err != nil {
		t.Fatal(err)
	}
	raw, ok := sig.Extra["futureField"]
	if !ok {
		t.Fatal("futureField not captured in Extra")
	}
	var nested struct {
		Nested bool `json:"nested"`
	}
	if err := json.Unmarshal(raw, &nested); err != nil || !nested.Nested {
		t.Fatalf("extra value: %s", raw)
	}
	out, err := json.Marshal(sig)
	if err != nil {
		t.Fatal(err)
	}
	var round map[string]json.RawMessage
	if err := json.Unmarshal(out, &round); err != nil {
		t.Fatal(err)
	}
	if _, ok := round["futureField"]; !ok {
		t.Fatalf("futureField lost on marshal: %s", out)
	}
	if _, ok := round["id"]; !ok {
		t.Fatalf("known field lost on marshal: %s", out)
	}
}

func TestTraderProfileNestedRoundTrip(t *testing.T) {
	var profile traderspy.TraderProfile
	data := []byte(`{"trader":{"topTraderId":"t1","positions":[{"symbol":"BTCUSDT"}],"nestedMetric":12},"positions":[],"profileMetric":true}`)
	if err := json.Unmarshal(data, &profile); err != nil {
		t.Fatal(err)
	}
	if profile.TopTraderID != "t1" || profile.Positions == nil || len(profile.Positions) != 0 {
		t.Fatalf("unexpected profile: %+v", profile)
	}
	if string(profile.Extra["nestedMetric"]) != "12" || string(profile.Extra["profileMetric"]) != "true" || profile.Extra["trader"] != nil {
		t.Fatalf("unexpected extra fields: %+v", profile.Extra)
	}
	out, err := json.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}
	var roundTrip traderspy.TraderProfile
	if err := json.Unmarshal(out, &roundTrip); err != nil || !reflect.DeepEqual(profile, roundTrip) {
		t.Fatalf("round trip: %s, error = %v", out, err)
	}
}

func TestModelMarshalReportsInvalidNestedValues(t *testing.T) {
	for _, model := range []any{
		traderspy.SignalDetail{Indicators: map[string]any{"rsi": math.NaN()}},
		traderspy.TraderProfile{Positions: []traderspy.Position{{Size: math.NaN()}}},
	} {
		if _, err := json.Marshal(model); err == nil {
			t.Fatalf("%T: marshal should fail on NaN", model)
		}
	}
}

func TestCandleArrayReplacesPreviousObject(t *testing.T) {
	var candle traderspy.Candle
	if err := json.Unmarshal([]byte(`{"time":1764157200000,"volume":12,"trades":3}`), &candle); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(`[1764157260000,"2.5","3","2","2.75"]`), &candle); err != nil {
		t.Fatal(err)
	}
	if candle.Open != 2.5 || candle.Close != 2.75 || candle.Volume != 0 || candle.Extra != nil {
		t.Fatalf("unexpected candle: %+v", candle)
	}
}

func TestSymbolListRoundTrip(t *testing.T) {
	var list traderspy.SymbolList
	if err := json.Unmarshal([]byte(`{"symbols":["BTCUSDT",null,{"symbol":"ETHUSDT"}],"market":"futures"}`), &list); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(list.Symbols, []string{"BTCUSDT", "ETHUSDT"}) {
		t.Fatalf("unexpected symbols: %v", list.Symbols)
	}
	out, err := json.Marshal(list)
	if err != nil {
		t.Fatal(err)
	}
	var roundTrip traderspy.SymbolList
	if err := json.Unmarshal(out, &roundTrip); err != nil || !reflect.DeepEqual(list, roundTrip) {
		t.Fatalf("round trip: %s, error = %v", out, err)
	}
}
