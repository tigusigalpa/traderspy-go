// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Igor Sazonov

// Command prices prints the live price, 24h change and volume of BTC, ETH
// and SOL.
//
//	export TRADERSPY_API_KEY=tspy_...
//	go run ./examples/prices
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

	prices, _, err := client.Market.Prices(context.Background(), []string{"BTC", "ETH", "SOL"})
	if err != nil {
		log.Fatal(err)
	}
	for _, p := range prices.Data {
		change := "-"
		if p.Change24hPct != nil {
			change = fmt.Sprintf("%+.2f%%", *p.Change24hPct)
		}
		fmt.Printf("%-10s %12g  %s\n", p.Symbol, p.Price, change)
	}
}
