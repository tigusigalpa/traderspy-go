// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Igor Sazonov

// Command paginate walks every published AI signal with the lazy
// pagination iterator.
//
//	export TRADERSPY_API_KEY=tspy_...
//	go run ./examples/paginate
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

	total := 0
	for signal, err := range client.Signals.All(context.Background(), nil) {
		if err != nil {
			log.Fatal(err)
		}
		total++
		if total <= 5 {
			fmt.Println(signal.CreatedAt.Format("01-02 15:04"), signal.Action, signal.Coin)
		}
	}
	fmt.Printf("\n%d signals read\n", total)
	if limit := client.RateLimit(); limit != nil {
		fmt.Printf("Calls left today: %d\n", limit.Remaining)
	}
}
