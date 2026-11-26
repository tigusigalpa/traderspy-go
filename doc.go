// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Igor Sazonov

// Package traderspy is a Go client library for the TraderSpy Signals API v1.
//
// TraderSpy is a read-only REST API for crypto futures research: AI trading
// signals with outcomes, smart-money traders and their positions on Binance,
// Hyperliquid, Bybit and OKX, prices and candles, technical indicators,
// funding and open interest, a market screener and condition backtests.
//
// Create a client with [New] (or [NewFromEnv], which reads the
// TRADERSPY_API_KEY environment variable) and call one of the service
// clients:
//
//	client, err := traderspy.New("tspy_...")
//	if err != nil {
//		log.Fatal(err)
//	}
//	signals, resp, err := client.Signals.List(ctx, &traderspy.ListSignalsOptions{
//		Importance: traderspy.ImportanceHigh,
//		Limit:      10,
//	})
//
// Endpoint methods return a [Response] when an HTTP response was received.
// It includes the request ID and any reported rate-limit counters. Use
// errors.As to inspect API failures as [Error] or one of its typed variants,
// such as [RateLimitError]. The request context covers all retry attempts.
//
// API reference: https://traderspy.app/api
// OpenAPI spec:  https://api.traderspy.app/v1/openapi.json
//
// Market research data, not financial advice.
package traderspy
