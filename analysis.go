// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Igor Sazonov

package traderspy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// AnalysisService groups the analysis endpoints: technical indicators, the
// market screener and condition backtests. Obtain it from
// [Client.Analysis].
//
// Docs: https://traderspy.app/api ("Technical indicators", "Screener",
// "Backtest a condition")
type AnalysisService struct {
	client *Client
}

// Condition is one screener/backtest condition. Up to 3 conditions are
// AND-ed: "lt"/"gt" compare the latest bar, "crossAbove"/"crossBelow" are
// one-bar events.
//
// Examples: RSI oversold {"rsi", "lt", 30}; above the 200 EMA
// {"priceVsEma", "gt", 0, Period: 200}; golden cross {"emaSpread",
// "crossAbove", 0, Period: 50, Period2: 200}.
type Condition struct {
	// Metric is what is measured, required.
	Metric Metric `json:"metric"`
	// Op is the comparison, required.
	Op Op `json:"op"`
	// Value is the threshold, required.
	Value float64 `json:"value"`
	// Period is the metric's lookback, 2–200 (e.g. RSI period or the fast
	// EMA of emaSpread). Optional.
	Period int `json:"period,omitempty"`
	// Period2 is the second lookback of two-series metrics, 2–200.
	// Optional.
	Period2 int `json:"period2,omitempty"`
}

// validate checks a condition's fields against the API's ranges.
func (c Condition) validate() error {
	if !c.Metric.Valid() {
		return invalid("conditions[].metric", fmt.Sprintf("unknown metric %q", c.Metric))
	}
	if !c.Op.Valid() {
		return invalid("conditions[].op", fmt.Sprintf("unknown op %q", c.Op))
	}
	if c.Period != 0 {
		if err := checkRange("conditions[].period", c.Period, 2, 200); err != nil {
			return err
		}
	}
	if c.Period2 != 0 {
		if err := checkRange("conditions[].period2", c.Period2, 2, 200); err != nil {
			return err
		}
	}
	return nil
}

// validateConditions checks a condition list: at most 3, each valid.
func validateConditions(conditions []Condition, required bool) error {
	if len(conditions) == 0 {
		if required {
			return invalid("conditions", "at least one condition is required")
		}
		return nil
	}
	if len(conditions) > 3 {
		return invalid("conditions", fmt.Sprintf("at most 3 conditions allowed, got %d", len(conditions)))
	}
	for _, c := range conditions {
		if err := c.validate(); err != nil {
			return err
		}
	}
	return nil
}

// ─── technical indicators ────────────────────────────────────────────────

// IndicatorData is one technical indicator reading. The fields depend on
// the indicator (an RSI carries "value" and "series", Bollinger carries the
// three bands, MACD its three lines), so the payload is kept generic.
type IndicatorData map[string]any

// IntervalTechnicals are the computed indicators of one symbol on one
// timeframe. Fields not mapped in this SDK version are kept in Extra.
type IntervalTechnicals struct {
	// Interval is the candle timeframe.
	Interval string `json:"interval"`
	// Indicators are the requested indicators keyed by name.
	Indicators map[string]IndicatorData `json:"indicators"`
	// Bias is the timeframe's bullish/bearish summary, when reported.
	Bias string `json:"bias"`

	// Extra holds fields this SDK version does not map yet.
	Extra map[string]json.RawMessage `json:"-"`
}

// UnmarshalJSON implements [json.Unmarshaler].
func (t *IntervalTechnicals) UnmarshalJSON(data []byte) error {
	type alias IntervalTechnicals
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*t = IntervalTechnicals(a)
	t.Extra = extraFields(data, IntervalTechnicals{})
	return nil
}

// MarshalJSON implements [json.Marshaler].
func (t IntervalTechnicals) MarshalJSON() ([]byte, error) {
	type alias IntervalTechnicals
	return marshalWithExtra(alias(t), t.Extra)
}

// SymbolTechnicals are the computed indicators of one symbol, on one or
// several timeframes. Fields not mapped in this SDK version are kept in
// Extra.
type SymbolTechnicals struct {
	// Symbol is the trading pair.
	Symbol string `json:"symbol"`
	// Intervals are the per-timeframe results, when several were
	// requested. When a single interval is requested the API may instead
	// put the indicators under Indicators directly.
	Intervals []IntervalTechnicals `json:"intervals"`
	// Indicators are the single-interval results, keyed by name.
	Indicators map[string]IndicatorData `json:"indicators"`
	// Bias is the summary verdict, when reported.
	Bias string `json:"bias"`
	// Confluence is the multi-timeframe verdict of an `intervals` request,
	// when reported.
	Confluence map[string]any `json:"confluence"`

	// Extra holds fields this SDK version does not map yet.
	Extra map[string]json.RawMessage `json:"-"`
}

// UnmarshalJSON implements [json.Unmarshaler].
func (t *SymbolTechnicals) UnmarshalJSON(data []byte) error {
	type alias SymbolTechnicals
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*t = SymbolTechnicals(a)
	t.Extra = extraFields(data, SymbolTechnicals{})
	return nil
}

// MarshalJSON implements [json.Marshaler].
func (t SymbolTechnicals) MarshalJSON() ([]byte, error) {
	type alias SymbolTechnicals
	return marshalWithExtra(alias(t), t.Extra)
}

// TechnicalsResult is the answer of GET /v1/technicals: up to 19 indicators
// with history, zones and crossovers, for up to 3 symbols × 3 timeframes,
// plus a bias summary.
type TechnicalsResult struct {
	// Data carries one entry per requested symbol.
	Data []SymbolTechnicals `json:"data"`
	// Confluence is the multi-timeframe verdict when `intervals` was used.
	Confluence map[string]any `json:"confluence"`

	// Extra holds fields this SDK version does not map yet.
	Extra map[string]json.RawMessage `json:"-"`
}

// UnmarshalJSON implements [json.Unmarshaler].
func (r *TechnicalsResult) UnmarshalJSON(data []byte) error {
	type alias TechnicalsResult
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*r = TechnicalsResult(a)
	r.Extra = extraFields(data, TechnicalsResult{})
	return nil
}

// MarshalJSON implements [json.Marshaler].
func (r TechnicalsResult) MarshalJSON() ([]byte, error) {
	type alias TechnicalsResult
	return marshalWithExtra(alias(r), r.Extra)
}

// TechnicalsOptions are the parameters of [AnalysisService.Technicals].
// Pass Symbol for one pair or Symbols for up to [MaxTechnicalsSymbols];
// Interval for one timeframe or Intervals for up to three.
type TechnicalsOptions struct {
	// Symbol is one trading pair (alternative to Symbols).
	Symbol string
	// Symbols are up to 3 pairs in one call; the answer then carries one
	// entry per symbol in Data.
	Symbols []string
	// Interval is the timeframe (ignored when Intervals is given).
	Interval Interval
	// Intervals are up to 3 timeframes in one call, with a
	// multi-timeframe confluence verdict.
	Intervals []Interval
	// Indicators are the indicators to compute, at most 19 (default
	// "rsi,macd,ema,bollinger").
	Indicators []string
	// History is prior values per indicator in "series", 0–20 (default 5).
	History int
	// Periods are period overrides, marshaled to a JSON object in the
	// query, e.g. {"rsi": 7, "ema": [9, 21, 55]}.
	Periods map[string]any
}

// validate implements client-side validation before the request.
func (o *TechnicalsOptions) validate() error {
	if o == nil {
		return invalid("symbol", "symbol or symbols is required")
	}
	hasOne := strings.TrimSpace(o.Symbol) != ""
	hasMany := len(o.Symbols) > 0
	if !hasOne && !hasMany {
		return invalid("symbol", "symbol or symbols is required")
	}
	if hasOne && hasMany {
		return invalid("symbol", "pass either symbol or symbols, not both")
	}
	if hasMany {
		if err := ValidateSymbolList(o.Symbols, MaxTechnicalsSymbols); err != nil {
			return err
		}
	}
	if err := checkEnum("interval", string(o.Interval), o.Interval.Valid()); err != nil {
		return err
	}
	if len(o.Intervals) > 3 {
		return invalid("intervals", fmt.Sprintf("at most 3 intervals allowed, got %d", len(o.Intervals)))
	}
	for _, iv := range o.Intervals {
		if !iv.Valid() {
			return invalid("intervals", fmt.Sprintf("unknown interval %q", iv))
		}
	}
	if len(o.Indicators) > 19 {
		return invalid("indicators", fmt.Sprintf("at most 19 indicators allowed, got %d", len(o.Indicators)))
	}
	if o.History != 0 {
		if err := checkRange("history", o.History, 0, 20); err != nil {
			return err
		}
	}
	return nil
}

// Technicals returns up to 19 technical indicators with history, zones and
// crossovers, for up to 3 symbols × 3 timeframes, plus a bias summary.
//
// GET /v1/technicals — docs: https://traderspy.app/api ("Technical indicators")
func (s *AnalysisService) Technicals(ctx context.Context, opts *TechnicalsOptions) (*TechnicalsResult, *Response, error) {
	if err := opts.validate(); err != nil {
		return nil, nil, err
	}
	q := url.Values{}
	if opts.Symbol != "" {
		q.Set("symbol", NormalizeSymbol(opts.Symbol))
	}
	if len(opts.Symbols) > 0 {
		q.Set("symbols", strings.Join(NormalizeSymbols(opts.Symbols), ","))
	}
	if opts.Interval != "" {
		q.Set("interval", string(opts.Interval))
	}
	if len(opts.Intervals) > 0 {
		parts := make([]string, len(opts.Intervals))
		for i, iv := range opts.Intervals {
			parts[i] = string(iv)
		}
		q.Set("intervals", strings.Join(parts, ","))
	}
	if len(opts.Indicators) > 0 {
		q.Set("indicators", strings.Join(opts.Indicators, ","))
	}
	if opts.History != 0 {
		q.Set("history", strconv.Itoa(opts.History))
	}
	if len(opts.Periods) > 0 {
		encoded, err := json.Marshal(opts.Periods)
		if err != nil {
			return nil, nil, invalid("periods", fmt.Sprintf("cannot encode: %s", err))
		}
		q.Set("periods", string(encoded))
	}
	var out TechnicalsResult
	resp, err := s.client.get(ctx, "/v1/technicals", q, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// ─── screener ────────────────────────────────────────────────────────────

// ScreenerRow is one matching symbol of a screener run, with the metrics
// the scan computed for it. Fields not mapped in this SDK version are kept
// in Extra.
type ScreenerRow struct {
	// Symbol is the trading pair.
	Symbol string `json:"symbol"`
	// Price is the last price.
	Price float64 `json:"price"`
	// Change24hPct is the 24-hour change, in percent.
	Change24hPct *float64 `json:"change24hPct"`
	// Volume24hUSD is the 24-hour volume in USD.
	Volume24hUSD *float64 `json:"volume24hUsd"`
	// RSI14 is the 14-period RSI on the scanned interval.
	RSI14 *float64 `json:"rsi14"`
	// ADX14 is the 14-period ADX on the scanned interval.
	ADX14 *float64 `json:"adx14"`
	// ATRPct is the ATR as a percent of price.
	ATRPct *float64 `json:"atrPct"`
	// VolumeRatio is the bar's volume against its average.
	VolumeRatio *float64 `json:"volumeRatio"`
	// Trend is the trend-phase label (e.g. relative to the 200 average).
	Trend string `json:"trend"`
	// Bias is the directional verdict.
	Bias string `json:"bias"`
	// Squeeze flags a Bollinger squeeze.
	Squeeze bool `json:"squeeze"`

	// Extra holds fields this SDK version does not map yet.
	Extra map[string]json.RawMessage `json:"-"`
}

// UnmarshalJSON implements [json.Unmarshaler].
func (r *ScreenerRow) UnmarshalJSON(data []byte) error {
	type alias ScreenerRow
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*r = ScreenerRow(a)
	r.Extra = extraFields(data, ScreenerRow{})
	return nil
}

// MarshalJSON implements [json.Marshaler].
func (r ScreenerRow) MarshalJSON() ([]byte, error) {
	type alias ScreenerRow
	return marshalWithExtra(alias(r), r.Extra)
}

// ScreenerUniverse describes what the screener scanned.
type ScreenerUniverse struct {
	// Scanned is the number of perpetuals evaluated.
	Scanned int `json:"scanned"`

	// Extra holds fields this SDK version does not map yet.
	Extra map[string]json.RawMessage `json:"-"`
}

// UnmarshalJSON implements [json.Unmarshaler].
func (u *ScreenerUniverse) UnmarshalJSON(data []byte) error {
	type alias ScreenerUniverse
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*u = ScreenerUniverse(a)
	u.Extra = extraFields(data, ScreenerUniverse{})
	return nil
}

// MarshalJSON implements [json.Marshaler].
func (u ScreenerUniverse) MarshalJSON() ([]byte, error) {
	type alias ScreenerUniverse
	return marshalWithExtra(alias(u), u.Extra)
}

// ScreenerResult is the answer of POST /v1/screener.
type ScreenerResult struct {
	// Conditions describes the applied conditions in words.
	Conditions []string `json:"conditions"`
	// Interval is the timeframe the scan evaluated on.
	Interval string `json:"interval"`
	// Matched is how many perpetuals passed the conditions.
	Matched int `json:"matched"`
	// Universe describes the scanned set.
	Universe *ScreenerUniverse `json:"universe"`
	// Results are the matching rows.
	Results []ScreenerRow `json:"results"`
	// Warnings are notices about the scan (e.g. dropped conditions).
	Warnings []string `json:"warnings"`

	// Extra holds fields this SDK version does not map yet.
	Extra map[string]json.RawMessage `json:"-"`
}

// UnmarshalJSON implements [json.Unmarshaler].
func (r *ScreenerResult) UnmarshalJSON(data []byte) error {
	type alias ScreenerResult
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*r = ScreenerResult(a)
	r.Extra = extraFields(data, ScreenerResult{})
	return nil
}

// MarshalJSON implements [json.Marshaler].
func (r ScreenerResult) MarshalJSON() ([]byte, error) {
	type alias ScreenerResult
	return marshalWithExtra(alias(r), r.Extra)
}

// ScreenerOptions are the parameters of [AnalysisService.Screener].
// Without Symbols it scans the most-traded perpetuals (Universe controls
// how many); without Conditions the answer is a comparison table.
type ScreenerOptions struct {
	// Interval is the timeframe to evaluate on (default 4h).
	Interval Interval
	// Conditions are up to 3 AND-ed conditions. Optional: without them the
	// answer is a comparison table.
	Conditions []Condition
	// Symbols is an explicit list to scan, at most [MaxScreenerSymbols];
	// omit to scan the most-traded.
	Symbols []string
	// Universe is how many of the most-traded symbols to scan when Symbols
	// is omitted, 5–100 (default 50).
	Universe int
	// Assets selects which perpetuals the most-traded list holds: "all"
	// (default) or "crypto".
	Assets Assets
	// Limit is rows to return, 1–50 (default 20).
	Limit int
	// SortBy is the row ordering: "volume", "change24h" or "metric"
	// (default: "metric" with conditions, else "volume").
	SortBy ScreenerSortBy
	// SortOrder is the direction (default "desc").
	SortOrder SortOrder
}

// validate implements client-side validation before the request.
func (o *ScreenerOptions) validate() error {
	if o == nil {
		return nil
	}
	if err := checkEnum("interval", string(o.Interval), o.Interval.Valid()); err != nil {
		return err
	}
	if err := validateConditions(o.Conditions, false); err != nil {
		return err
	}
	if len(o.Symbols) > MaxScreenerSymbols {
		return invalid("symbols", fmt.Sprintf("at most %d symbols allowed, got %d", MaxScreenerSymbols, len(o.Symbols)))
	}
	for _, s := range o.Symbols {
		if strings.TrimSpace(s) == "" {
			return invalid("symbols", "empty symbol in list")
		}
	}
	if o.Universe != 0 {
		if err := checkRange("universe", o.Universe, 5, 100); err != nil {
			return err
		}
	}
	if err := checkEnum("assets", string(o.Assets), o.Assets.Valid()); err != nil {
		return err
	}
	if o.Limit != 0 {
		if err := checkRange("limit", o.Limit, 1, 50); err != nil {
			return err
		}
	}
	if err := checkEnum("sortBy", string(o.SortBy), o.SortBy.Valid()); err != nil {
		return err
	}
	return checkEnum("sortOrder", string(o.SortOrder), o.SortOrder.Valid())
}

// screenerBody is the POST body of /v1/screener.
type screenerBody struct {
	Interval   Interval       `json:"interval,omitempty"`
	Conditions []Condition    `json:"conditions,omitempty"`
	Symbols    []string       `json:"symbols,omitempty"`
	Universe   int            `json:"universe,omitempty"`
	Assets     Assets         `json:"assets,omitempty"`
	Limit      int            `json:"limit,omitempty"`
	SortBy     ScreenerSortBy `json:"sortBy,omitempty"`
	SortOrder  SortOrder      `json:"sortOrder,omitempty"`
}

// Screener scans the most-traded symbols (or an explicit list) for up to 3
// AND-ed conditions in one call — for example RSI oversold inside an
// uptrend.
//
// POST /v1/screener — docs: https://traderspy.app/api ("Screener")
func (s *AnalysisService) Screener(ctx context.Context, opts *ScreenerOptions) (*ScreenerResult, *Response, error) {
	if err := opts.validate(); err != nil {
		return nil, nil, err
	}
	var body screenerBody
	if opts != nil {
		body = screenerBody{
			Interval:   opts.Interval,
			Conditions: opts.Conditions,
			Symbols:    NormalizeSymbols(opts.Symbols),
			Universe:   opts.Universe,
			Assets:     opts.Assets,
			Limit:      opts.Limit,
			SortBy:     opts.SortBy,
			SortOrder:  opts.SortOrder,
		}
	}
	var out ScreenerResult
	resp, err := s.client.post(ctx, "/v1/screener", body, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// ─── backtest ────────────────────────────────────────────────────────────

// BacktestHorizon is the forward-return statistics at one horizon: what
// price did N bars after the condition fired, against the baseline.
// A nil field means there were too few samples to report.
type BacktestHorizon struct {
	// Bars is how many bars ahead the returns are measured.
	Bars int `json:"bars"`
	// AvgReturnPct is the average forward return, in percent.
	AvgReturnPct *float64 `json:"avgReturnPct"`
	// WinRatePct is the share of positive outcomes, in percent.
	WinRatePct *float64 `json:"winRatePct"`
	// EdgePct is the average return minus the baseline average of every
	// candle, in percent.
	EdgePct *float64 `json:"edgePct"`
	// WorstPct is the worst forward return seen, in percent.
	WorstPct *float64 `json:"worstPct"`

	// Extra holds fields this SDK version does not map yet.
	Extra map[string]json.RawMessage `json:"-"`
}

// UnmarshalJSON implements [json.Unmarshaler].
func (h *BacktestHorizon) UnmarshalJSON(data []byte) error {
	type alias BacktestHorizon
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*h = BacktestHorizon(a)
	h.Extra = extraFields(data, BacktestHorizon{})
	return nil
}

// MarshalJSON implements [json.Marshaler].
func (h BacktestHorizon) MarshalJSON() ([]byte, error) {
	type alias BacktestHorizon
	return marshalWithExtra(alias(h), h.Extra)
}

// RegimeStats are the outcome statistics inside one trend phase.
type RegimeStats struct {
	// Samples is how many occurrences fell in this phase.
	Samples int `json:"samples"`
	// EdgePct is the edge over the baseline in this phase, in percent.
	EdgePct *float64 `json:"edgePct"`
	// AvgReturnPct is the average forward return in this phase, in
	// percent.
	AvgReturnPct *float64 `json:"avgReturnPct"`
	// WinRatePct is the win rate in this phase, in percent.
	WinRatePct *float64 `json:"winRatePct"`

	// Extra holds fields this SDK version does not map yet.
	Extra map[string]json.RawMessage `json:"-"`
}

// UnmarshalJSON implements [json.Unmarshaler].
func (s *RegimeStats) UnmarshalJSON(data []byte) error {
	type alias RegimeStats
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*s = RegimeStats(a)
	s.Extra = extraFields(data, RegimeStats{})
	return nil
}

// MarshalJSON implements [json.Marshaler].
func (s RegimeStats) MarshalJSON() ([]byte, error) {
	type alias RegimeStats
	return marshalWithExtra(alias(s), s.Extra)
}

// RegimeHorizon splits one horizon by trend phase: outcomes when price was
// above vs below its 200-candle average.
type RegimeHorizon struct {
	// Hours is the horizon length in hours.
	Hours float64 `json:"hours"`
	// Above are the stats for the above-average phase.
	Above *RegimeStats `json:"above"`
	// Below are the stats for the below-average phase.
	Below *RegimeStats `json:"below"`

	// Extra holds fields this SDK version does not map yet.
	Extra map[string]json.RawMessage `json:"-"`
}

// UnmarshalJSON implements [json.Unmarshaler].
func (h *RegimeHorizon) UnmarshalJSON(data []byte) error {
	type alias RegimeHorizon
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*h = RegimeHorizon(a)
	h.Extra = extraFields(data, RegimeHorizon{})
	return nil
}

// MarshalJSON implements [json.Marshaler].
func (h RegimeHorizon) MarshalJSON() ([]byte, error) {
	type alias RegimeHorizon
	return marshalWithExtra(alias(h), h.Extra)
}

// BacktestRegimes is the trend-phase split of a backtest.
type BacktestRegimes struct {
	// Horizons are the per-horizon phase splits.
	Horizons []RegimeHorizon `json:"horizons"`

	// Extra holds fields this SDK version does not map yet.
	Extra map[string]json.RawMessage `json:"-"`
}

// UnmarshalJSON implements [json.Unmarshaler].
func (r *BacktestRegimes) UnmarshalJSON(data []byte) error {
	type alias BacktestRegimes
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*r = BacktestRegimes(a)
	r.Extra = extraFields(data, BacktestRegimes{})
	return nil
}

// MarshalJSON implements [json.Marshaler].
func (r BacktestRegimes) MarshalJSON() ([]byte, error) {
	type alias BacktestRegimes
	return marshalWithExtra(alias(r), r.Extra)
}

// BacktestResult is the answer of POST /v1/backtest: what price did after
// the condition fired on one symbol, against its own baseline.
type BacktestResult struct {
	// Symbol echoes the tested symbol, when reported.
	Symbol string `json:"symbol"`
	// Interval echoes the tested interval, when reported.
	Interval string `json:"interval"`
	// Occurrences is how many times the condition fired.
	Occurrences int `json:"occurrences"`
	// Bars is how many candles were tested.
	Bars int `json:"bars"`
	// Horizons are the forward-return statistics per horizon.
	Horizons []BacktestHorizon `json:"horizons"`
	// Baseline describes the all-candle average the edge is measured
	// against, when reported.
	Baseline map[string]any `json:"baseline"`
	// Regimes splits the outcome by trend phase (above/below the
	// 200-candle average).
	Regimes *BacktestRegimes `json:"regimes"`
	// Warnings are notices about the run.
	Warnings []string `json:"warnings"`

	// Extra holds fields this SDK version does not map yet.
	Extra map[string]json.RawMessage `json:"-"`
}

// UnmarshalJSON implements [json.Unmarshaler].
func (r *BacktestResult) UnmarshalJSON(data []byte) error {
	type alias BacktestResult
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*r = BacktestResult(a)
	r.Extra = extraFields(data, BacktestResult{})
	return nil
}

// MarshalJSON implements [json.Marshaler].
func (r BacktestResult) MarshalJSON() ([]byte, error) {
	type alias BacktestResult
	return marshalWithExtra(alias(r), r.Extra)
}

// BacktestOptions are the parameters of [AnalysisService.Backtest].
type BacktestOptions struct {
	// Symbol is the trading pair, required ("BTCUSDT" or a bare "BTC").
	Symbol string
	// Interval is the candle timeframe (default 4h).
	Interval Interval
	// Conditions are up to 3 AND-ed conditions, required.
	Conditions []Condition
	// Horizons are bars ahead to measure returns over, 1–4 entries each
	// 1–200 (default ≈ 4h / 1d / 3d for the interval).
	Horizons []int
}

// validate implements client-side validation before the request.
func (o *BacktestOptions) validate() error {
	if o == nil || strings.TrimSpace(o.Symbol) == "" {
		return invalid("symbol", "symbol is required")
	}
	if err := checkEnum("interval", string(o.Interval), o.Interval.Valid()); err != nil {
		return err
	}
	if err := validateConditions(o.Conditions, true); err != nil {
		return err
	}
	if len(o.Horizons) > 4 {
		return invalid("horizons", fmt.Sprintf("at most 4 horizons allowed, got %d", len(o.Horizons)))
	}
	for _, h := range o.Horizons {
		if err := checkRange("horizons[]", h, 1, 200); err != nil {
			return err
		}
	}
	return nil
}

// backtestBody is the POST body of /v1/backtest.
type backtestBody struct {
	Symbol     string      `json:"symbol"`
	Interval   Interval    `json:"interval,omitempty"`
	Conditions []Condition `json:"conditions"`
	Horizons   []int       `json:"horizons,omitempty"`
}

// Backtest measures what price did after a condition on one symbol:
// occurrences, forward returns, win rate and the edge over the baseline,
// also split by trend phase (above/below the 200-candle average).
//
// POST /v1/backtest — docs: https://traderspy.app/api ("Backtest a condition")
func (s *AnalysisService) Backtest(ctx context.Context, opts *BacktestOptions) (*BacktestResult, *Response, error) {
	if err := opts.validate(); err != nil {
		return nil, nil, err
	}
	body := backtestBody{
		Symbol:     NormalizeSymbol(opts.Symbol),
		Interval:   opts.Interval,
		Conditions: opts.Conditions,
		Horizons:   opts.Horizons,
	}
	var out BacktestResult
	resp, err := s.client.post(ctx, "/v1/backtest", body, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}
