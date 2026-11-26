// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Igor Sazonov

// Command candles prints the last hourly BTC candles (oldest first; the
// last one may still be open).
//
//	export TRADERSPY_API_KEY=tspy_...
//	go run ./examples/candles
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

	candles, _, err := client.Market.Candles(context.Background(), &traderspy.CandlesOptions{
		Symbol:   "BTC",
		Interval: traderspy.Interval1h,
		Limit:    24,
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, c := range candles.Data {
		fmt.Printf("%s  O %-10g H %-10g L %-10g C %-10g V %g\n",
			c.Time.Format("01-02 15:04"), c.Open, c.High, c.Low, c.Close, c.Volume)
	}
}
