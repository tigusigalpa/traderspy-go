# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-11-26

Initial release.

### Added

- `traderspy.New(key, opts...)` and `traderspy.NewFromEnv()` (`TRADERSPY_API_KEY`),
  options: `WithHTTPClient`, `WithBaseURL`, `WithTimeout`, `WithUserAgent`,
  `WithRetryConfig`.
- Service layer: `Signals`, `SmartMoney`, `Market`, `Analysis`, `Strategies`
  covering all 19 endpoints of the TraderSpy Signals API v1.
- Client-side validation of every option (`*ValidationError`) before any HTTP call.
- Typed API errors: `*Error` plus `*AuthError` (401), `*ForbiddenError` (403),
  `*NotFoundError` (404), `*RateLimitError` (429) for `errors.As`, with
  `IsRateLimit()` / `IsAuth()` / `IsTemporary()` helpers.
- Automatic retries on 429 / 503 / transport errors with exponential backoff +
  jitter honouring `Retry-After` (`RetryConfig{MaxRetries, MinWait, MaxWait}`).
- `*Response` metadata per call: status, `X-Request-Id`, parsed `X-RateLimit-*`
  headers, pagination; `client.RateLimit()` for the latest answer.
- Lazy pagination iterators (`iter.Seq2`): `Signals.All`,
  `SmartMoney.AllPositions`, `SmartMoney.AllTraderHistory`, `Strategies.AllAlerts`.
- Symbol helpers: `NormalizeSymbol`, `NormalizeSymbols`, `ValidateSymbolList`;
  `Signal.TargetPrices()` percent-to-price conversion of targets/stop.
- Typed enums for every parameter domain (`Interval`, `Timeframe`, `TimeRange`,
  `Period`, `Importance`, `Source`, `RankingType`, `Order`, `SortOrder`,
  `Metric`, `Op`, ...).
- Forward compatibility: unknown JSON fields are preserved in `Extra` and survive
  marshal round-trips.
- `examples/` runnable programs and `example_test.go`; test suite with
  `httptest` fixtures covering every endpoint, error and retry path.
