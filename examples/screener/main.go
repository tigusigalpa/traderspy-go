// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Igor Sazonov

// Command screener scans the 100 most-traded crypto perpetuals for RSI(14)
// under 30 on the 4h chart.
//
//	export TRADERSPY_API_KEY=tspy_...
//	go run ./examples/screener
package main

import (
	"context"
	"fmt"
	"log"

	traderspy "github.com/tigusigalpa/traderspy-go"
)

func main() {
	client, err := traderspy.NewFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	scan, _, err := client.Analysis.Screener(context.Background(), &traderspy.ScreenerOptions{
		Interval: traderspy.Interval4h,
		Conditions: []traderspy.Condition{
			{Metric: traderspy.MetricRSI, Op: traderspy.OpLT, Value: 30},
		},
		Universe:  100,
		Assets:    traderspy.AssetsCrypto,
		Limit:     50,
		SortBy:    traderspy.ScreenerSortMetric,
		SortOrder: traderspy.SortOrderAsc,
	})
	if err != nil {
		log.Fatal(err)
	}

	if scan.Universe != nil {
		fmt.Printf("matched %d of %d perps\n\n", scan.Matched, scan.Universe.Scanned)
	} else {
		fmt.Printf("matched %d perps\n\n", scan.Matched)
	}
	for _, row := range scan.Results {
		rsi := "-"
		if row.RSI14 != nil {
			rsi = fmt.Sprintf("%.1f", *row.RSI14)
		}
		fmt.Printf("%-14s %12g  RSI %5s  %s\n", row.Symbol, row.Price, rsi, row.Trend)
	}
	for _, warning := range scan.Warnings {
		fmt.Println("note:", warning)
	}
}
