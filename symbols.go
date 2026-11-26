// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Igor Sazonov

package traderspy

import (
	"fmt"
	"strings"
)

// Maximum symbol counts accepted by the list-style endpoints.
const (
	// MaxPriceSymbols is the limit of GET /v1/market/prices.
	MaxPriceSymbols = 20
	// MaxDerivativeSymbols is the limit of GET /v1/market/derivatives.
	MaxDerivativeSymbols = 5
	// MaxTechnicalsSymbols is the limit of the `symbols` parameter of
	// GET /v1/technicals.
	MaxTechnicalsSymbols = 3
	// MaxScreenerSymbols is the limit of the `symbols` list of
	// POST /v1/screener.
	MaxScreenerSymbols = 100
)

// quoteSuffixes are the quote currencies recognized as part of a futures
// symbol already, so NormalizeSymbol does not append "USDT" to them.
var quoteSuffixes = []string{"USDT", "USDC", "PERP", "USD"}

// NormalizeSymbol converts a user-typed ticker into the symbol form the API
// expects: it is uppercased and trimmed, and a bare coin gets the USDT
// perpetual suffix. "btc" and "BTC" become "BTCUSDT", "ethusdt" stays
// "ETHUSDT".
func NormalizeSymbol(symbol string) string {
	s := strings.ToUpper(strings.TrimSpace(symbol))
	if s == "" {
		return ""
	}
	for _, suffix := range quoteSuffixes {
		if strings.HasSuffix(s, suffix) {
			return s
		}
	}
	return s + "USDT"
}

// NormalizeSymbols maps [NormalizeSymbol] over a list, dropping empty
// entries.
func NormalizeSymbols(symbols []string) []string {
	out := make([]string, 0, len(symbols))
	for _, s := range symbols {
		if n := NormalizeSymbol(s); n != "" {
			out = append(out, n)
		}
	}
	return out
}

// ValidateSymbolList checks a symbol list against an endpoint's maximum
// (e.g. [MaxPriceSymbols] for prices, [MaxDerivativeSymbols] for
// derivatives, [MaxTechnicalsSymbols] for technicals). It returns a
// *ValidationError naming the first problem, before any HTTP call is made.
func ValidateSymbolList(symbols []string, max int) error {
	if len(symbols) == 0 {
		return invalid("symbols", "at least one symbol is required")
	}
	if len(symbols) > max {
		return invalid("symbols", fmt.Sprintf("at most %d symbols allowed, got %d", max, len(symbols)))
	}
	for _, s := range symbols {
		if strings.TrimSpace(s) == "" {
			return invalid("symbols", "empty symbol in list")
		}
	}
	return nil
}
