// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Igor Sazonov

package traderspy_test

import (
	"context"
	"fmt"
	"log"
	"time"

	traderspy "github.com/tigusigalpa/traderspy-go"
)

// Create a client from the TRADERSPY_API_KEY environment variable and list
// the newest high-importance signals.
func ExampleNewFromEnv() {
	client, err := traderspy.NewFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	signals, resp, err := client.Signals.List(context.Background(), &traderspy.ListSignalsOptions{
		Importance: traderspy.ImportanceHigh,
		Limit:      10,
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, s := range signals.Data {
		fmt.Println(s.Action, s.Coin, s.Timeframe, s.StrategyName, s.URL)
	}
	if limit := resp.RateLimit; limit != nil {
		fmt.Println("calls left today:", limit.Remaining)
	}
}

// Every method returns a *Response with the rate-limit headers of its
// answer, and errors inspectable with errors.As.
func ExampleClient_rateLimit() {
	client, _ := traderspy.New("tspy_...")

	_, resp, err := client.Market.Prices(context.Background(), []string{"BTC", "ETH"})
	if err != nil {
		log.Fatal(err)
	}
	if limit := resp.RateLimit; limit != nil {
		fmt.Println("minute calls left:", limit.RemainingMinute)
		fmt.Println("daily counter resets at:", limit.Reset.Format(time.RFC3339))
	}
}

// Walk every signal matching the filters with the Go 1.23 iterator; pages
// are fetched lazily.
func ExampleSignalsService_All() {
	client, _ := traderspy.New("tspy_...")

	for signal, err := range client.Signals.All(context.Background(), &traderspy.ListSignalsOptions{
		Importance: traderspy.ImportanceHigh,
	}) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(signal.CreatedAt.Format(time.DateOnly), signal.Coin, signal.Action)
	}
}
