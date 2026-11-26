// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Igor Sazonov

// Command signals prints the top-10 high-importance AI signals with their
// entry, first target and stop as prices.
//
//	export TRADERSPY_API_KEY=tspy_...
//	go run ./examples/signals
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

	page, resp, err := client.Signals.List(context.Background(), &traderspy.ListSignalsOptions{
		Importance: traderspy.ImportanceHigh,
		Limit:      10,
	})
	if err != nil {
		log.Fatal(err)
	}

	for _, s := range page.Data {
		side := "LONG "
		if s.Action == traderspy.ActionSell {
			side = "SHORT"
		}
		levels := s.TargetPrices()
		outcome := s.ResolutionStatus
		if outcome == "" {
			outcome = "open"
		}
		fmt.Printf("%s  %s %-12s %3s  entry %-12g TP1 %-12g SL %-12g %s  %s\n",
			s.CreatedAt.Format("01-02 15:04"), side, s.Coin, s.Timeframe,
			s.Price, levels["tp1"], levels["sl"], outcome, s.StrategyName)
	}
	fmt.Printf("\n%d signals on this page\n", len(page.Data))
	if limit := resp.RateLimit; limit != nil {
		fmt.Printf("Calls left today: %d\n", limit.Remaining)
	}
}
