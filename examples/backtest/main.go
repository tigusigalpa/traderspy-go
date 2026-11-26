// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Igor Sazonov

// Command backtest measures forward returns from bars with RSI(14) below 30 on
// BTC 4h: occurrences, forward returns, win rate and the edge over the
// baseline, split by trend phase.
//
//	export TRADERSPY_API_KEY=tspy_...
//	go run ./examples/backtest
package main

import (
	"context"
	"fmt"
	"log"

	traderspy "github.com/tigusigalpa/traderspy-go"
)

func pct(v *float64) string {
	if v == nil {
		return "-"
	}
	return fmt.Sprintf("%.2f", *v)
}

func regimeEdge(stats *traderspy.RegimeStats) string {
	if stats == nil {
		return "-"
	}
	return fmt.Sprintf("%s (%d samples)", pct(stats.EdgePct), stats.Samples)
}

func main() {
	client, err := traderspy.NewFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	result, _, err := client.Analysis.Backtest(context.Background(), &traderspy.BacktestOptions{
		Symbol:   "BTC",
		Interval: traderspy.Interval4h,
		Conditions: []traderspy.Condition{
			{Metric: traderspy.MetricRSI, Op: traderspy.OpLT, Value: 30},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("RSI < 30 on BTC 4h: %d times in %d candles\n\n", result.Occurrences, result.Bars)
	fmt.Printf("%8s %9s %8s %9s %9s\n", "after", "avg %", "win %", "edge %", "worst %")
	for _, h := range result.Horizons {
		fmt.Printf("%6dh %9s %8s %9s %9s\n", h.Bars*4, pct(h.AvgReturnPct), pct(h.WinRatePct), pct(h.EdgePct), pct(h.WorstPct))
	}
	if result.Regimes != nil {
		for _, reg := range result.Regimes.Horizons {
			fmt.Printf("\nedge after %gh: above 200-avg %s, below %s\n",
				reg.Hours, regimeEdge(reg.Above), regimeEdge(reg.Below))
		}
	}
}
