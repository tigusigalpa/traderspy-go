// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Igor Sazonov

package traderspy

import (
	"context"
	"encoding/json"
	"iter"
	"net/url"
	"strconv"
	"time"
)

// SignalsService groups the AI-signals endpoints. Obtain it from
// [Client.Signals].
//
// Docs: https://traderspy.app/api ("List AI signals", "Signal statistics",
// "Get a signal")
type SignalsService struct {
	client *Client
}

// Signal is a published AI signal: a long/short setup on a crypto futures
// symbol with entry, targets and stop as percent distances from Price, and
// the outcome so far.
//
// Fields not mapped in this SDK version are kept in Extra.
type Signal struct {
	// ID is the signal identifier; pass it to [SignalsService.Get].
	ID string `json:"id"`
	// StrategyName is the AI strategy that produced the signal.
	StrategyName string `json:"strategyName"`
	// Importance is the signal's importance (high, medium, low).
	Importance Importance `json:"importance"`
	// Action is the direction: "buy" (long) or "sell" (short).
	Action SignalAction `json:"action"`
	// SignalStrength is the strategy's confidence score.
	SignalStrength float64 `json:"signalStrength"`
	// Coin is the symbol the signal is on (e.g. "BTCUSDT", or a bare "BTC").
	Coin string `json:"coin"`
	// Timeframe is the candle timeframe of the setup.
	Timeframe string `json:"timeframe"`
	// Price is the reference price when the signal was published; targets
	// and stop are percent distances from it (see [Signal.TargetPrices]).
	Price float64 `json:"price"`
	// Targets are the take-profit and stop levels.
	Targets []SignalTarget `json:"targets"`
	// ResolutionStatus is how the signal resolved so far (e.g. "open",
	// a target hit, a stop hit); empty while unknown.
	ResolutionStatus string `json:"resolutionStatus"`
	// CreatedAt is when the signal was published; pass the newest one you
	// have as ListSignalsOptions.Since to poll for new signals.
	CreatedAt Time `json:"createdAt"`
	// URL is the signal's page on traderspy.app.
	URL string `json:"url"`

	// Extra holds fields this SDK version does not map yet.
	Extra map[string]json.RawMessage `json:"-"`
}

// UnmarshalJSON implements [json.Unmarshaler] and preserves unknown fields
// in Extra.
func (s *Signal) UnmarshalJSON(data []byte) error {
	type alias Signal
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*s = Signal(a)
	s.Extra = extraFields(data, Signal{})
	return nil
}

// MarshalJSON implements [json.Marshaler], keeping Extra fields.
func (s Signal) MarshalJSON() ([]byte, error) {
	type alias Signal
	return marshalWithExtra(alias(s), s.Extra)
}

// SignalTarget is one level of a signal: a take-profit ("tp1", "tp2", "tp3")
// or the stop-loss ("sl"), given as a percent distance from the entry
// Price.
type SignalTarget struct {
	// Label is the human label of the level.
	Label string `json:"label"`
	// Type is "tp1"/"tp2"/"tp3" for take-profits and "sl" for the stop.
	Type string `json:"type"`
	// Pct is the distance from the signal's Price, in percent.
	Pct float64 `json:"pct"`
}

// TargetPrices converts the signal's targets and stop into absolute prices:
// a buy's targets sit above the entry and its stop below; a sell's the
// other way round. Keys are the target types ("tp1", "tp2", "sl", ...).
func (s *Signal) TargetPrices() map[string]float64 {
	levels := make(map[string]float64, len(s.Targets))
	up := s.Action != ActionSell
	for _, t := range s.Targets {
		pct := t.Pct
		if pct < 0 {
			pct = -pct
		}
		pct /= 100
		towardsProfit := t.Type != "sl"
		rises := towardsProfit == up
		if rises {
			levels[t.Type] = s.Price * (1 + pct)
		} else {
			levels[t.Type] = s.Price * (1 - pct)
		}
	}
	return levels
}

// SignalList is one page of AI signals.
type SignalList struct {
	// Data are the signals, newest first.
	Data []Signal `json:"data"`
	// Pagination describes the page.
	Pagination *Pagination `json:"pagination"`
}

// SignalDetail is a signal in full: everything [Signal] carries plus
// indicator values, the AI review, the resolution details and the live
// price. Fields not mapped in this SDK version are kept in Extra.
type SignalDetail struct {
	Signal

	// Indicators are the indicator values behind the signal; the shape
	// depends on the producing strategy.
	Indicators map[string]any `json:"indicators"`
	// AIReview is the strategy's narrative review of the setup.
	AIReview string `json:"aiReview"`
	// LivePrice is the current market price of the coin, when the answer
	// includes it.
	LivePrice *float64 `json:"livePrice"`
	// Resolution describes how the signal played out once resolved.
	Resolution *SignalResolution `json:"resolution"`
}

// UnmarshalJSON implements [json.Unmarshaler]. An embedded type with its
// own UnmarshalJSON would swallow the whole object, so the signal fields
// are decoded through a type without methods.
func (d *SignalDetail) UnmarshalJSON(data []byte) error {
	type plainSignal Signal
	var detail struct {
		plainSignal
		Indicators map[string]any    `json:"indicators"`
		AIReview   string            `json:"aiReview"`
		LivePrice  *float64          `json:"livePrice"`
		Resolution *SignalResolution `json:"resolution"`
	}
	if err := json.Unmarshal(data, &detail); err != nil {
		return err
	}
	d.Signal = Signal(detail.plainSignal)
	d.Indicators = detail.Indicators
	d.AIReview = detail.AIReview
	d.LivePrice = detail.LivePrice
	d.Resolution = detail.Resolution
	d.Extra = extraFields(data, SignalDetail{})
	return nil
}

// MarshalJSON implements [json.Marshaler]. The embedded Signal marshals to
// its own object; the detail fields are merged over it, keeping Extra.
func (d SignalDetail) MarshalJSON() ([]byte, error) {
	type plainSignal Signal
	out := struct {
		plainSignal
		Indicators *map[string]any   `json:"indicators,omitempty"`
		AIReview   string            `json:"aiReview,omitempty"`
		LivePrice  *float64          `json:"livePrice,omitempty"`
		Resolution *SignalResolution `json:"resolution,omitempty"`
	}{plainSignal: plainSignal(d.Signal), AIReview: d.AIReview, LivePrice: d.LivePrice, Resolution: d.Resolution}
	if d.Indicators != nil {
		out.Indicators = &d.Indicators
	}
	return marshalWithExtra(out, d.Extra)
}

// SignalResolution is the outcome block of a resolved signal. Fields not
// mapped in this SDK version are kept in Extra.
type SignalResolution struct {
	// Status mirrors Signal.ResolutionStatus.
	Status string `json:"status"`
	// ResolvedAt is when the signal hit its final target or stop.
	ResolvedAt Time `json:"resolvedAt"`
	// ResultPct is the realized move in percent.
	ResultPct *float64 `json:"resultPct"`

	// Extra holds fields this SDK version does not map yet.
	Extra map[string]json.RawMessage `json:"-"`
}

// UnmarshalJSON implements [json.Unmarshaler].
func (r *SignalResolution) UnmarshalJSON(data []byte) error {
	type alias SignalResolution
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*r = SignalResolution(a)
	r.Extra = extraFields(data, SignalResolution{})
	return nil
}

// MarshalJSON implements [json.Marshaler].
func (r SignalResolution) MarshalJSON() ([]byte, error) {
	type alias SignalResolution
	return marshalWithExtra(alias(r), r.Extra)
}

// SignalStats is the outcome statistics of published signals over a period:
// totals, targets hit, stops and win rate, overall and per strategy.
// Fields not mapped in this SDK version are kept in Extra.
type SignalStats struct {
	// Period echoes the requested window ("24h", ...).
	Period string `json:"period"`
	// TotalSignals is the number of signals published in the window.
	TotalSignals int `json:"totalSignals"`
	// Resolved is the number of signals with a final outcome.
	Resolved int `json:"resolved"`
	// TargetsHit counts signals that reached a target.
	TargetsHit int `json:"targetsHit"`
	// StopsHit counts signals stopped out.
	StopsHit int `json:"stopsHit"`
	// WinRatePct is the win rate of resolved signals, in percent.
	WinRatePct *float64 `json:"winRatePct"`
	// Strategies breaks the statistics down per AI strategy.
	Strategies []StrategyStats `json:"strategies"`

	// Extra holds fields this SDK version does not map yet.
	Extra map[string]json.RawMessage `json:"-"`
}

// UnmarshalJSON implements [json.Unmarshaler].
func (s *SignalStats) UnmarshalJSON(data []byte) error {
	type alias SignalStats
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*s = SignalStats(a)
	s.Extra = extraFields(data, SignalStats{})
	return nil
}

// MarshalJSON implements [json.Marshaler].
func (s SignalStats) MarshalJSON() ([]byte, error) {
	type alias SignalStats
	return marshalWithExtra(alias(s), s.Extra)
}

// StrategyStats are the signal statistics of one AI strategy.
// Fields not mapped in this SDK version are kept in Extra.
type StrategyStats struct {
	// StrategyName is the AI strategy.
	StrategyName string `json:"strategyName"`
	// TotalSignals is its number of signals in the window.
	TotalSignals int `json:"totalSignals"`
	// Resolved is its number of signals with a final outcome.
	Resolved int `json:"resolved"`
	// TargetsHit counts its signals that reached a target.
	TargetsHit int `json:"targetsHit"`
	// StopsHit counts its signals stopped out.
	StopsHit int `json:"stopsHit"`
	// WinRatePct is its win rate, in percent.
	WinRatePct *float64 `json:"winRatePct"`

	// Extra holds fields this SDK version does not map yet.
	Extra map[string]json.RawMessage `json:"-"`
}

// UnmarshalJSON implements [json.Unmarshaler].
func (s *StrategyStats) UnmarshalJSON(data []byte) error {
	type alias StrategyStats
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*s = StrategyStats(a)
	s.Extra = extraFields(data, StrategyStats{})
	return nil
}

// MarshalJSON implements [json.Marshaler].
func (s StrategyStats) MarshalJSON() ([]byte, error) {
	type alias StrategyStats
	return marshalWithExtra(alias(s), s.Extra)
}

// ListSignalsOptions filters [SignalsService.List]. Zero fields are not
// sent, so the API defaults apply.
type ListSignalsOptions struct {
	// Limit is signals per page, 1–50 (API default 20).
	Limit int
	// Offset is signals to skip, 0–10000 (API default 0).
	Offset int
	// Importance filters by level; "high" is high only, "medium" is
	// high + medium.
	Importance Importance
	// Coin returns only signals on this coin ("BTC", "ETHUSDT").
	Coin string
	// Strategy returns only signals of the AI strategy whose name contains
	// these words, in order.
	Strategy string
	// Timeframe returns only signals on this candle timeframe.
	Timeframe Timeframe
	// Since returns only signals published after this time. To poll for
	// new signals, pass the newest CreatedAt you already have.
	Since time.Time
}

// validate implements client-side validation before the request.
func (o *ListSignalsOptions) validate() error {
	if o == nil {
		return nil
	}
	if o.Limit != 0 {
		if err := checkRange("limit", o.Limit, 1, 50); err != nil {
			return err
		}
	}
	if o.Offset != 0 {
		if err := checkRange("offset", o.Offset, 0, 10000); err != nil {
			return err
		}
	}
	if err := checkEnum("importance", string(o.Importance), o.Importance.Valid()); err != nil {
		return err
	}
	if err := checkEnum("timeframe", string(o.Timeframe), o.Timeframe.Valid()); err != nil {
		return err
	}
	return nil
}

// query encodes the options into URL query values.
func (o *ListSignalsOptions) query() url.Values {
	q := url.Values{}
	if o == nil {
		return q
	}
	if o.Limit != 0 {
		q.Set("limit", strconv.Itoa(o.Limit))
	}
	if o.Offset != 0 {
		q.Set("offset", strconv.Itoa(o.Offset))
	}
	if o.Importance != "" {
		q.Set("importance", string(o.Importance))
	}
	if o.Coin != "" {
		q.Set("coin", o.Coin)
	}
	if o.Strategy != "" {
		q.Set("strategy", o.Strategy)
	}
	if o.Timeframe != "" {
		q.Set("timeframe", string(o.Timeframe))
	}
	if !o.Since.IsZero() {
		q.Set("since", o.Since.UTC().Format(time.RFC3339))
	}
	return q
}

// List returns a page of published AI signals, newest first: entry,
// targets, stop and the outcome so far.
//
// GET /v1/signals — docs: https://traderspy.app/api ("List AI signals")
func (s *SignalsService) List(ctx context.Context, opts *ListSignalsOptions) (*SignalList, *Response, error) {
	if err := opts.validate(); err != nil {
		return nil, nil, err
	}
	var out SignalList
	resp, err := s.client.get(ctx, "/v1/signals", opts.query(), &out)
	if err != nil {
		return nil, resp, err
	}
	resp.Pagination = out.Pagination
	return &out, resp, nil
}

// All returns a Go 1.23 iterator that yields every signal matching the
// filters, fetching offset pages lazily until pagination ends, yield
// returns false, or ctx is done (yielded as the last value together with a
// non-nil error).
//
//	for signal, err := range client.Signals.All(ctx, &traderspy.ListSignalsOptions{
//		Importance: traderspy.ImportanceHigh,
//	}) {
//		if err != nil {
//			log.Fatal(err)
//		}
//		fmt.Println(signal.Coin, signal.StrategyName)
//	}
func (s *SignalsService) All(ctx context.Context, opts *ListSignalsOptions) iter.Seq2[*Signal, error] {
	return func(yield func(*Signal, error) bool) {
		o := ListSignalsOptions{}
		if opts != nil {
			o = *opts
		}
		if o.Limit == 0 {
			o.Limit = 50 // the API's own page maximum keeps the call count low
		}
		for {
			page, _, err := s.List(ctx, &o)
			if err != nil {
				yield(nil, err)
				return
			}
			if page == nil || len(page.Data) == 0 {
				return
			}
			for i := range page.Data {
				if !yield(&page.Data[i], nil) {
					return
				}
			}
			if page.Pagination == nil || !page.Pagination.HasMore {
				return
			}
			o.Offset += len(page.Data)
		}
	}
}

// Get returns one signal in full: indicator values, the AI review, entry,
// resolution and the live price.
//
// GET /v1/signals/{id} — docs: https://traderspy.app/api ("Get a signal")
func (s *SignalsService) Get(ctx context.Context, id string) (*SignalDetail, *Response, error) {
	if id == "" {
		return nil, nil, invalid("id", "signal id is required")
	}
	var out SignalDetail
	resp, err := s.client.getData(ctx, "/v1/signals/"+url.PathEscape(id), nil, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// SignalStatsOptions filters [SignalsService.Stats]. Zero fields are not
// sent, so the API defaults apply.
type SignalStatsOptions struct {
	// Period is the window of the statistics: 4h, 8h, 24h (default) or 7d.
	Period Period
	// Strategy limits the statistics to one AI strategy, matched by part
	// of its name.
	Strategy string
}

// validate implements client-side validation before the request.
func (o *SignalStatsOptions) validate() error {
	if o == nil {
		return nil
	}
	return checkEnum("period", string(o.Period), o.Period.Valid())
}

// Stats returns outcome statistics of published signals over a period:
// totals, targets hit, stops and win rate, overall and per strategy.
//
// GET /v1/signals/stats — docs: https://traderspy.app/api ("Signal statistics")
func (s *SignalsService) Stats(ctx context.Context, opts *SignalStatsOptions) (*SignalStats, *Response, error) {
	if err := opts.validate(); err != nil {
		return nil, nil, err
	}
	q := url.Values{}
	if opts != nil {
		if opts.Period != "" {
			q.Set("period", string(opts.Period))
		}
		if opts.Strategy != "" {
			q.Set("strategy", opts.Strategy)
		}
	}
	var out SignalStats
	resp, err := s.client.getData(ctx, "/v1/signals/stats", q, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}
