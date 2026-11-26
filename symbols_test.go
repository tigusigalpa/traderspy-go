// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Igor Sazonov

package traderspy_test

import (
	"errors"
	"testing"

	traderspy "github.com/tigusigalpa/traderspy-go"
)

func TestNormalizeSymbol(t *testing.T) {
	cases := map[string]string{
		"btc":      "BTCUSDT",
		"BTC":      "BTCUSDT",
		" btc ":    "BTCUSDT",
		"ethusdt":  "ETHUSDT",
		"ETHUSDT":  "ETHUSDT",
		"1000pepe": "1000PEPEUSDT",
		"BTCUSDC":  "BTCUSDC",
		"solperp":  "SOLPERP",
		"":         "",
		"HYPE":     "HYPEUSDT",
	}
	for in, want := range cases {
		if got := traderspy.NormalizeSymbol(in); got != want {
			t.Fatalf("NormalizeSymbol(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeSymbols(t *testing.T) {
	got := traderspy.NormalizeSymbols([]string{"btc", " ETHUSDT ", ""})
	if len(got) != 2 || got[0] != "BTCUSDT" || got[1] != "ETHUSDT" {
		t.Fatalf("got %v", got)
	}
}

func TestValidateSymbolList(t *testing.T) {
	var ve *traderspy.ValidationError

	if err := traderspy.ValidateSymbolList(nil, 3); !errors.As(err, &ve) {
		t.Fatalf("empty: %v", err)
	}
	if err := traderspy.ValidateSymbolList([]string{"A", "B", "C", "D"}, 3); !errors.As(err, &ve) {
		t.Fatalf("too many: %v", err)
	}
	if err := traderspy.ValidateSymbolList([]string{"A", " "}, 3); !errors.As(err, &ve) {
		t.Fatalf("empty symbol: %v", err)
	}
	if err := traderspy.ValidateSymbolList([]string{"A", "B"}, 3); err != nil {
		t.Fatalf("valid: %v", err)
	}
}
