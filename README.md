# TraderSpy Golang Client/SDK/Library

![TraderSpy Golang Client/SDK/Library](https://i.postimg.cc/Y0tZ22vh/traderspy-app-go-client.jpg)

[![CI](https://github.com/tigusigalpa/traderspy-go/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/tigusigalpa/traderspy-go/actions/workflows/ci.yml)
[![Tests](https://github.com/tigusigalpa/traderspy-go/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/tigusigalpa/traderspy-go/actions/workflows/test.yml)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-green?style=flat-square)](LICENSE)
[![CodeQL](https://github.com/tigusigalpa/traderspy-go/actions/workflows/codeql.yml/badge.svg?branch=main)](https://github.com/tigusigalpa/traderspy-go/actions/workflows/codeql.yml)
[![Codecov](https://codecov.io/gh/tigusigalpa/traderspy-go/graph/badge.svg)](https://codecov.io/gh/tigusigalpa/traderspy-go)
[![GitHub Release](https://img.shields.io/github/v/release/tigusigalpa/traderspy-go?style=flat-square)](https://github.com/tigusigalpa/traderspy-go/releases)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue?style=flat-square&logo=go)](https://pkg.go.dev/github.com/tigusigalpa/traderspy-go)

A Go client for the read-only TraderSpy Signals API v1. Use it to pull published
crypto futures signals and their outcomes into your dashboard, track positions
from Binance, Bybit, Hyperliquid and OKX leaderboards, or screen markets and
measure what price did after an indicator condition.

The SDK requires **Go 1.23 or newer** and uses the Go standard library.
[API reference](https://traderspy.app/api) · [OpenAPI spec](https://api.traderspy.app/v1/openapi.json) · [Go package docs](https://pkg.go.dev/github.com/tigusigalpa/traderspy-go)

## Get started

Install the module from your Go project.

```bash
go get github.com/tigusigalpa/traderspy-go
```

Create a TraderSpy API key and set `TRADERSPY_API_KEY` in the environment of
your app. `NewFromEnv` reads that variable. Use `New` if your app already
gets the key from a secret manager. The client sends it in the
`Authorization: Bearer` header. [Manage API keys](https://traderspy.app/api)

```bash
export TRADERSPY_API_KEY='tspy_...'
```

```powershell
$env:TRADERSPY_API_KEY = 'tspy_...'
```

This program fetches one page of BTC signals and prints the first take-profit
price when the signal includes one.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	traderspy "github.com/tigusigalpa/traderspy-go"
)

func main() {
	client, err := traderspy.NewFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	page, resp, err := client.Signals.List(ctx, &traderspy.ListSignalsOptions{
		Coin:       "BTC",
		Importance: traderspy.ImportanceHigh,
		Limit:      10,
	})
	if err != nil {
		log.Fatal(err)
	}

	for _, signal := range page.Data {
		fmt.Printf("%s %s %s at %g\n", signal.Action, signal.Coin, signal.Timeframe, signal.Price)
		if tp1, ok := signal.TargetPrices()["tp1"]; ok {
			fmt.Printf("  first target: %g\n", tp1)
		}
	}
	if limit := resp.RateLimit; limit != nil {
		fmt.Println("API calls left today:", limit.Remaining)
	}
}
```

## API coverage

| Service | Method | HTTP method and path |
|---|---|---|
| `Signals` | `List(ctx, opts)` | `GET /v1/signals` |
| | `Stats(ctx, opts)` | `GET /v1/signals/stats` |
| | `Get(ctx, id)` | `GET /v1/signals/{id}` |
| `SmartMoney` | `TopTraders(ctx, opts)` | `GET /v1/smart-money/traders` |
| | `EliteTraders(ctx)` | `GET /v1/smart-money/traders/elite` |
| | `Trader(ctx, source, id, opts)` | `GET /v1/smart-money/traders/{source}/{traderId}` |
| | `TraderHistory(ctx, source, id, opts)` | `GET /v1/smart-money/traders/{source}/{traderId}/history` |
| | `Positions(ctx, opts)` | `GET /v1/smart-money/positions` |
| | `Stats(ctx, opts)` | `GET /v1/smart-money/stats` |
| | `Exchanges(ctx)` | `GET /v1/smart-money/exchanges` |
| `Market` | `Prices(ctx, symbols)` | `GET /v1/market/prices` |
| | `Candles(ctx, opts)` | `GET /v1/market/candles` |
| | `Symbols(ctx)` | `GET /v1/market/symbols` |
| | `Derivatives(ctx, symbols)` | `GET /v1/market/derivatives` |
| `Analysis` | `Technicals(ctx, opts)` | `GET /v1/technicals` |
| | `Screener(ctx, opts)` | `POST /v1/screener` |
| | `Backtest(ctx, opts)` | `POST /v1/backtest` |
| `Strategies` | `List(ctx)` | `GET /v1/strategies` |
| | `Alerts(ctx, opts)` | `GET /v1/strategies/alerts` |

Enable strategy access on the API key to read your own alert strategies and
their alerts. For trader profiles and history, pass the leaderboard entry's
`TopTraderID` and its exchange, such as `traderspy.SourceHyperliquid`.

### Read signals a page at a time

Use `Signals.List` to inspect the response metadata after each call. The
pagination block is available on both the result and `resp.Pagination` when
the API sends it. For polling, pass your latest stored `CreatedAt.Time` as
`ListSignalsOptions.Since` and deduplicate by signal ID.

Use `Signals.All` for lazy iteration. It requests the next page only after
you consume the current one. A `break` stops fetching, and a request error
is yielded once before the iterator ends.

```go
for signal, err := range client.Signals.All(ctx, &traderspy.ListSignalsOptions{
	Importance: traderspy.ImportanceHigh,
}) {
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(signal.CreatedAt.Format(time.RFC3339), signal.Coin, signal.Action)
}
```

| Iterator | Pagination |
|---|---|
| `Signals.All(ctx, opts)` | Signal offset |
| `SmartMoney.AllPositions(ctx, opts)` | Position offset |
| `SmartMoney.AllTraderHistory(ctx, source, id, opts)` | History page |
| `Strategies.AllAlerts(ctx, opts)` | Alert offset |

Iterators honor the endpoint's page and offset limits. They use the API's
maximum page size when you leave `Limit` at zero. Each fetched page is a
separate API call.

### Screen and backtest a condition

The screener evaluates up to three conditions together across an explicit
symbol list or a most-traded universe. Pass the same conditions to
`Backtest` to compare forward returns with the baseline for one symbol.
Horizons are measured in bars. On a 4-hour interval, six bars equal one day.

```go
conditions := []traderspy.Condition{
	{Metric: traderspy.MetricRSI, Op: traderspy.OpLT, Value: 30, Period: 14},
	{Metric: traderspy.MetricPriceVsEMA, Op: traderspy.OpGT, Value: 0, Period: 200},
}

scan, _, err := client.Analysis.Screener(ctx, &traderspy.ScreenerOptions{
	Interval:   traderspy.Interval4h,
	Conditions: conditions,
	Symbols:    []string{"BTC", "ETH", "SOL"},
})
if err != nil {
	log.Fatal(err)
}
fmt.Printf("%d markets matched\n", scan.Matched)

study, _, err := client.Analysis.Backtest(ctx, &traderspy.BacktestOptions{
	Symbol:     "BTC",
	Interval:   traderspy.Interval4h,
	Conditions: conditions,
	Horizons:   []int{1, 6, 18},
})
if err != nil {
	log.Fatal(err)
}
fmt.Printf("%d occurrences across %d candles\n", study.Occurrences, study.Bars)
for _, horizon := range study.Horizons {
	if horizon.EdgePct != nil {
		fmt.Printf("After %d bars, edge vs. baseline is %+.2f percentage points\n",
			horizon.Bars, *horizon.EdgePct)
	}
}
```

A condition backtest reports historical occurrences and forward returns
against the same market's baseline. Use an execution simulator to model
orders, fees and slippage.

## Responses, limits and errors

Endpoint methods return a typed result, response metadata and an error.
When the server replies, `resp` carries the HTTP status, request ID, raw
headers, and any rate-limit or pagination data. A validation error or a
transport failure can leave `resp` nil. Check `resp.RateLimit` before
reading it. `client.RateLimit()` returns a copy of the latest counters the
server reported. Use `resp.RateLimit` to associate counters with a specific
call when several goroutines share a client.

Use `errors.As` to read API and validation errors. Check caller cancellation
and deadlines with `errors.Is`.

```go
_, resp, err := client.Market.Prices(ctx, []string{"BTC", "ETH"})
if err != nil {
	var limited *traderspy.RateLimitError
	var invalid *traderspy.ValidationError
	var apiErr *traderspy.Error

	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		log.Printf("request stopped: %v", err)
	case errors.As(err, &invalid):
		log.Printf("invalid option %s: %s", invalid.Param, invalid.Reason)
	case errors.As(err, &limited):
		log.Printf("API quota: %s, retry after %s", limited.Code, limited.RetryAfter)
	case errors.As(err, &apiErr):
		log.Printf("API failure %d %s (request %s)", apiErr.StatusCode, apiErr.Code, apiErr.RequestID)
	default:
		log.Printf("request failed: %v", err)
	}
}
if resp != nil && resp.RateLimit != nil {
	log.Printf("calls left this minute: %d", resp.RateLimit.RemainingMinute)
}
```

| Error type | When it occurs |
|---|---|
| `*ValidationError` | The SDK rejected the options before issuing a request |
| `*AuthError` | HTTP 401 |
| `*ForbiddenError` | HTTP 403, such as a missing strategy permission |
| `*NotFoundError` | HTTP 404, including `no_data` |
| `*RateLimitError` | HTTP 429, including per-minute and daily quota limits |
| `*Error` | API and transport failures. A transport failure has status `0` and code `network_error` |

Typed HTTP errors also match `*Error`. Malformed success payloads return a
decoding error with response metadata.

### Retries and deadlines

The client retries HTTP 429 and 503, transport errors and interrupted reads of
successful responses. It makes up to three attempts by default, using capped
exponential backoff with jitter. When the server supplies `Retry-After`, that
value sets the minimum wait and can exceed the backoff cap. TraderSpy's daily
quota resets at 00:00 UTC. [TraderSpy API limits](https://traderspy.app/api)

Use a context deadline for the full call, including all attempts and waits.
The default HTTP client also has a 30-second timeout per attempt. Set
`MaxRetries: -1` to handle retries yourself. A zero value uses the default
retry count.

```go
client, err := traderspy.NewFromEnv(
	traderspy.WithTimeout(10*time.Second),
	traderspy.WithRetryConfig(traderspy.RetryConfig{
		MaxRetries: 1,
		MinWait:    500 * time.Millisecond,
		MaxWait:    5 * time.Second,
	}),
)
if err != nil {
	log.Fatal(err)
}
```

## Configuration and data types

| Option | Use it to |
|---|---|
| `WithBaseURL(url)` | Point to a test server, a staging server or a reverse proxy. Pass its root without the `/v1` suffix |
| `WithHTTPClient(client)` | Supply your own transport, proxy, redirect policy or instrumentation |
| `WithTimeout(duration)` | Set the HTTP timeout per attempt. Pass it after `WithHTTPClient` to apply it to a copy of the supplied client |
| `WithUserAgent(value)` | Override the default `traderspy-go/<version>` |
| `WithRetryConfig(config)` | Tune retry count and backoff. Zero fields use their defaults |

- `NormalizeSymbol("btc")` returns `BTCUSDT`. Market prices, candles and
  derivatives also normalize symbols for you.
- `Signal.TargetPrices()` converts percentage distances from the signal
  price into absolute levels keyed by names such as `tp1` and `sl`.
- Market candles are returned oldest first. The last bar may still be open.
- A nil pointer on a field such as `Price.Change24hPct` or
  `BacktestHorizon.EdgePct` means the API did not provide that value. Keep
  that distinction when displaying or storing data.
- Models with an `Extra` map keep unmapped JSON fields for later access or
  re-encoding. The SDK may serialize accepted payload variants into a common
  shape, such as writing the symbol list under `data`.

## Run the examples

Set your API key and run a program from the checkout. [Example source](examples)

| Command | What it shows |
|---|---|
| `go run ./examples/signals` | High-importance signals with entry, first target and stop prices |
| `go run ./examples/prices` | BTC, ETH and SOL price snapshots |
| `go run ./examples/candles` | The latest 24 hourly BTC candles |
| `go run ./examples/screener` | A 4-hour RSI scan across the 100 most-traded crypto perpetuals |
| `go run ./examples/backtest` | Forward returns from 4-hour BTC bars with RSI below 30 |
| `go run ./examples/paginate` | Lazy iteration over signals |

To try an unpublished checkout from another Go module, point that module at
your local copy.

```bash
go mod edit -replace=github.com/tigusigalpa/traderspy-go=/path/to/traderspy-go
go get github.com/tigusigalpa/traderspy-go@v0.0.0
```

The fixture-based test suite runs without an API key.

```bash
go test -race ./...
go vet ./...
```

## License

MIT © 2026 Igor Sazonov. [License](LICENSE)

