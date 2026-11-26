// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Igor Sazonov

package traderspy

import (
	"context"
	"encoding/json"
	"iter"
	"maps"
	"net/url"
	"strconv"
)

// SmartMoneyService groups the smart-money endpoints: leaderboards of top
// traders on Binance, Hyperliquid, Bybit and OKX, their profiles, position
// history, open/closed positions and aggregate statistics. Obtain it from
// [Client.SmartMoney].
//
// Docs: https://traderspy.app/api ("Top traders", "Elite leaderboard",
// "Trader profile", "Trader history", "Smart-money positions",
// "Smart-money statistics", "Tracked exchanges")
type SmartMoneyService struct {
	client *Client
}

// Trader is a ranked smart-money trader on a leaderboard.
// Fields not mapped in this SDK version are kept in Extra.
type Trader struct {
	// TopTraderID identifies the trader; pass it to
	// [SmartMoneyService.Trader] and [SmartMoneyService.TraderHistory].
	TopTraderID string `json:"topTraderId"`
	// TraderName is the display name when the exchange publishes one.
	TraderName string `json:"traderName"`
	// Source is the exchange the trader is on.
	Source string `json:"source"`
	// Rank is the position on the leaderboard.
	Rank int `json:"rank"`
	// ROI is the return on investment over the time range, in percent.
	ROI *float64 `json:"roi"`
	// PnL is the profit over the time range, in USD.
	PnL *float64 `json:"pnl"`
	// SmartScore is TraderSpy's cross-exchange trader score.
	SmartScore *float64 `json:"smartScore"`
	// WinRatePct is the share of profitable closed positions, in percent.
	WinRatePct *float64 `json:"winRatePct"`

	// Extra holds fields this SDK version does not map yet.
	Extra map[string]json.RawMessage `json:"-"`
}

// UnmarshalJSON implements [json.Unmarshaler].
func (t *Trader) UnmarshalJSON(data []byte) error {
	type alias Trader
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*t = Trader(a)
	t.Extra = extraFields(data, Trader{})
	return nil
}

// MarshalJSON implements [json.Marshaler].
func (t Trader) MarshalJSON() ([]byte, error) {
	type alias Trader
	return marshalWithExtra(alias(t), t.Extra)
}

// TraderList is one page (or the whole board) of ranked traders.
type TraderList struct {
	// Data are the traders in ranking order.
	Data []Trader `json:"data"`
	// Pagination describes the page, when the endpoint paginates.
	Pagination *Pagination `json:"pagination"`
}

// Position is a tracked smart-money position, open or closed.
//
// Size is in the position's base asset; on OKX it is in contracts and
// EntryPrice is absent, so a USD notional cannot always be computed.
// Fields not mapped in this SDK version are kept in Extra.
type Position struct {
	// Source is the exchange the position is on.
	Source string `json:"source"`
	// TopTraderID identifies the trader holding the position.
	TopTraderID string `json:"topTraderId"`
	// TraderName is the trader's display name when available.
	TraderName string `json:"traderName"`
	// Symbol is the trading pair (e.g. "BTCUSDT").
	Symbol string `json:"symbol"`
	// Side is "long" or "short".
	Side string `json:"side"`
	// Size is the position size (base asset, or contracts on OKX).
	Size float64 `json:"size"`
	// EntryPrice is the average entry price, when reported.
	EntryPrice *float64 `json:"entryPrice"`
	// Leverage is the position leverage, when reported.
	Leverage *float64 `json:"leverage"`
	// IsOpen reports whether the position is still open.
	IsOpen bool `json:"isOpen"`
	// LastEvent is the most recent change: opened, "increase" (added to),
	// "partial_close" (partly closed) or closed.
	LastEvent string `json:"lastEvent"`
	// SortDate is when the position last changed (Unix milliseconds).
	SortDate UnixMillis `json:"sortDate"`
	// PnL is the realized profit of a closed position, in USD.
	PnL *float64 `json:"pnl"`
	// UnrealizedPnL is the open position's floating profit, in USD.
	UnrealizedPnL *float64 `json:"unrealizedPnl"`

	// Extra holds fields this SDK version does not map yet.
	Extra map[string]json.RawMessage `json:"-"`
}

// UnmarshalJSON implements [json.Unmarshaler].
func (p *Position) UnmarshalJSON(data []byte) error {
	type alias Position
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*p = Position(a)
	p.Extra = extraFields(data, Position{})
	return nil
}

// MarshalJSON implements [json.Marshaler].
func (p Position) MarshalJSON() ([]byte, error) {
	type alias Position
	return marshalWithExtra(alias(p), p.Extra)
}

// PositionList is one page of smart-money positions.
type PositionList struct {
	// Data are the positions, most recently changed first.
	Data []Position `json:"data"`
	// Pagination describes the page.
	Pagination *Pagination `json:"pagination"`
}

// TraderProfile is one trader's metrics and latest positions.
// Fields not mapped in this SDK version are kept in Extra.
type TraderProfile struct {
	Trader

	// Positions are the trader's latest positions.
	Positions []Position `json:"positions"`
}

// UnmarshalJSON implements [json.Unmarshaler]. The trader metrics may sit
// at the top level or under a "trader" member, and positions under
// "positions" on either level; an embedded type with its own UnmarshalJSON
// would swallow the whole object, so the parts are decoded explicitly.
func (p *TraderProfile) UnmarshalJSON(data []byte) error {
	var probe struct {
		Trader    json.RawMessage `json:"trader"`
		Positions []Position      `json:"positions"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return err
	}
	traderData := data
	nestedTrader := len(probe.Trader) > 0 && string(probe.Trader) != "null"
	if nestedTrader {
		traderData = probe.Trader
		if probe.Positions == nil {
			var nested struct {
				Positions []Position `json:"positions"`
			}
			if err := json.Unmarshal(probe.Trader, &nested); err != nil {
				return err
			}
			probe.Positions = nested.Positions
		}
	}
	type plainTrader Trader
	var t plainTrader
	if err := json.Unmarshal(traderData, &t); err != nil {
		return err
	}
	t.Extra = extraFields(traderData, TraderProfile{})
	if nestedTrader {
		outer := extraFields(data, TraderProfile{})
		delete(outer, "trader")
		if t.Extra == nil && len(outer) > 0 {
			t.Extra = outer
		} else {
			maps.Copy(t.Extra, outer)
		}
	}
	*p = TraderProfile{Trader: Trader(t), Positions: probe.Positions}
	return nil
}

// MarshalJSON implements [json.Marshaler]. The embedded Trader marshals to
// its own object; Positions is merged over it, keeping Extra.
func (p TraderProfile) MarshalJSON() ([]byte, error) {
	type plainTrader Trader
	out := struct {
		plainTrader
		Positions *[]Position `json:"positions,omitempty"`
	}{plainTrader: plainTrader(p.Trader)}
	if p.Positions != nil {
		out.Positions = &p.Positions
	}
	return marshalWithExtra(out, p.Extra)
}

// SmartMoneyStats is the aggregate smart-money statistics over a period:
// open positions, the long/short split, and the PnL and win rate of closed
// positions. Fields not mapped in this SDK version are kept in Extra.
type SmartMoneyStats struct {
	// Source echoes the requested exchange filter.
	Source string `json:"source"`
	// Period echoes the requested window.
	Period string `json:"period"`
	// OpenPositions is the number of currently open tracked positions.
	OpenPositions int `json:"openPositions"`
	// LongCount and ShortCount split open positions by direction.
	LongCount  int `json:"longCount"`
	ShortCount int `json:"shortCount"`
	// LongPct and ShortPct are the directional shares, in percent.
	LongPct  *float64 `json:"longPct"`
	ShortPct *float64 `json:"shortPct"`
	// ClosedPositions is the number closed inside the window.
	ClosedPositions int `json:"closedPositions"`
	// TotalPnL is the realized profit of positions closed in the window,
	// in USD.
	TotalPnL *float64 `json:"totalPnl"`
	// WinRatePct is the win rate of closed positions, in percent.
	WinRatePct *float64 `json:"winRatePct"`

	// Extra holds fields this SDK version does not map yet.
	Extra map[string]json.RawMessage `json:"-"`
}

// UnmarshalJSON implements [json.Unmarshaler].
func (s *SmartMoneyStats) UnmarshalJSON(data []byte) error {
	type alias SmartMoneyStats
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*s = SmartMoneyStats(a)
	s.Extra = extraFields(data, SmartMoneyStats{})
	return nil
}

// MarshalJSON implements [json.Marshaler].
func (s SmartMoneyStats) MarshalJSON() ([]byte, error) {
	type alias SmartMoneyStats
	return marshalWithExtra(alias(s), s.Extra)
}

// Exchange is one of the exchanges smart-money tracking covers.
// Fields not mapped in this SDK version are kept in Extra.
type Exchange struct {
	// ID is the source value to filter by ("binance", "hyperliquid",
	// "bybit", "okx").
	ID string `json:"id"`
	// Name is the display name.
	Name string `json:"name"`
	// Traders is the number of tracked traders, when reported.
	Traders int `json:"traders"`

	// Extra holds fields this SDK version does not map yet.
	Extra map[string]json.RawMessage `json:"-"`
}

// UnmarshalJSON implements [json.Unmarshaler].
func (e *Exchange) UnmarshalJSON(data []byte) error {
	type alias Exchange
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*e = Exchange(a)
	e.Extra = extraFields(data, Exchange{})
	return nil
}

// MarshalJSON implements [json.Marshaler].
func (e Exchange) MarshalJSON() ([]byte, error) {
	type alias Exchange
	return marshalWithExtra(alias(e), e.Extra)
}

// ExchangeList is the list of tracked exchanges.
type ExchangeList struct {
	// Data are the exchanges.
	Data []Exchange `json:"data"`
}

// TopTradersOptions filters [SmartMoneyService.TopTraders]. Zero fields are
// not sent, so the API defaults apply.
type TopTradersOptions struct {
	// Source is the exchange to read, or "all" (default).
	Source Source
	// TimeRange is the performance window (default 30D).
	TimeRange TimeRange
	// RankingType is the leaderboard snapshot (default ROI).
	RankingType RankingType
	// SortBy is the sort metric (default: the ranking type).
	SortBy TraderSortBy
	// Order is the sort direction (default DESC).
	Order Order
	// Limit is traders to return, 1–200 (default 20).
	Limit int
}

// validate implements client-side validation before the request.
func (o *TopTradersOptions) validate() error {
	if o == nil {
		return nil
	}
	if err := checkEnum("source", string(o.Source), o.Source.Valid()); err != nil {
		return err
	}
	if err := checkEnum("timeRange", string(o.TimeRange), o.TimeRange.Valid()); err != nil {
		return err
	}
	if err := checkEnum("rankingType", string(o.RankingType), o.RankingType.Valid()); err != nil {
		return err
	}
	if err := checkEnum("sortBy", string(o.SortBy), o.SortBy.Valid()); err != nil {
		return err
	}
	if err := checkEnum("order", string(o.Order), o.Order.Valid()); err != nil {
		return err
	}
	if o.Limit != 0 {
		if err := checkRange("limit", o.Limit, 1, 200); err != nil {
			return err
		}
	}
	return nil
}

// query encodes the options into URL query values.
func (o *TopTradersOptions) query() url.Values {
	q := url.Values{}
	if o == nil {
		return q
	}
	if o.Source != "" {
		q.Set("source", string(o.Source))
	}
	if o.TimeRange != "" {
		q.Set("timeRange", string(o.TimeRange))
	}
	if o.RankingType != "" {
		q.Set("rankingType", string(o.RankingType))
	}
	if o.SortBy != "" {
		q.Set("sortBy", string(o.SortBy))
	}
	if o.Order != "" {
		q.Set("order", string(o.Order))
	}
	if o.Limit != 0 {
		q.Set("limit", strconv.Itoa(o.Limit))
	}
	return q
}

// TopTraders returns ranked smart-money traders across exchanges, with ROI,
// PnL and SmartScore.
//
// GET /v1/smart-money/traders — docs: https://traderspy.app/api ("Top traders")
func (s *SmartMoneyService) TopTraders(ctx context.Context, opts *TopTradersOptions) (*TraderList, *Response, error) {
	if err := opts.validate(); err != nil {
		return nil, nil, err
	}
	var out TraderList
	resp, err := s.client.get(ctx, "/v1/smart-money/traders", opts.query(), &out)
	if err != nil {
		return nil, resp, err
	}
	resp.Pagination = out.Pagination
	return &out, resp, nil
}

// EliteTraders returns the cross-exchange elite leaderboard by SmartScore,
// recalculated daily. It takes no parameters.
//
// GET /v1/smart-money/traders/elite — docs: https://traderspy.app/api
// ("Elite leaderboard")
func (s *SmartMoneyService) EliteTraders(ctx context.Context) (*TraderList, *Response, error) {
	var out TraderList
	resp, err := s.client.get(ctx, "/v1/smart-money/traders/elite", nil, &out)
	if err != nil {
		return nil, resp, err
	}
	resp.Pagination = out.Pagination
	return &out, resp, nil
}

// TraderOptions filters [SmartMoneyService.Trader]. Zero fields are not
// sent, so the API defaults apply.
type TraderOptions struct {
	// TimeRange is the performance window (default 30D).
	TimeRange TimeRange
	// RankingType is the leaderboard snapshot (default ROI).
	RankingType RankingType
}

// validate implements client-side validation before the request.
func (o *TraderOptions) validate() error {
	if o == nil {
		return nil
	}
	if err := checkEnum("timeRange", string(o.TimeRange), o.TimeRange.Valid()); err != nil {
		return err
	}
	return checkEnum("rankingType", string(o.RankingType), o.RankingType.Valid())
}

// Trader returns one trader's metrics and latest positions. The source is
// required and must be a concrete exchange ("all" is not allowed);
// traderID is the TopTraderID of a leaderboard entry.
//
// GET /v1/smart-money/traders/{source}/{traderId} —
// docs: https://traderspy.app/api ("Trader profile")
func (s *SmartMoneyService) Trader(ctx context.Context, source Source, traderID string, opts *TraderOptions) (*TraderProfile, *Response, error) {
	if !source.PathValid() {
		return nil, nil, invalid("source", `must be one of "binance", "hyperliquid", "bybit", "okx"`)
	}
	if traderID == "" {
		return nil, nil, invalid("traderId", "trader id is required")
	}
	if err := opts.validate(); err != nil {
		return nil, nil, err
	}
	q := url.Values{}
	if opts != nil {
		if opts.TimeRange != "" {
			q.Set("timeRange", string(opts.TimeRange))
		}
		if opts.RankingType != "" {
			q.Set("rankingType", string(opts.RankingType))
		}
	}
	path := "/v1/smart-money/traders/" + url.PathEscape(string(source)) + "/" + url.PathEscape(traderID)
	var out TraderProfile
	resp, err := s.client.getData(ctx, path, q, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// TraderHistoryOptions filters [SmartMoneyService.TraderHistory]. Zero
// fields are not sent, so the API defaults apply.
type TraderHistoryOptions struct {
	// Page is the page number, 1–500 (default 1).
	Page int
	// Limit is positions per page, 1–50 (default 20).
	Limit int
}

// validate implements client-side validation before the request.
func (o *TraderHistoryOptions) validate() error {
	if o == nil {
		return nil
	}
	if o.Page != 0 {
		if err := checkRange("page", o.Page, 1, 500); err != nil {
			return err
		}
	}
	if o.Limit != 0 {
		if err := checkRange("limit", o.Limit, 1, 50); err != nil {
			return err
		}
	}
	return nil
}

// TraderHistory returns a trader's closed positions, newest first.
//
// GET /v1/smart-money/traders/{source}/{traderId}/history —
// docs: https://traderspy.app/api ("Trader history")
func (s *SmartMoneyService) TraderHistory(ctx context.Context, source Source, traderID string, opts *TraderHistoryOptions) (*PositionList, *Response, error) {
	if !source.PathValid() {
		return nil, nil, invalid("source", `must be one of "binance", "hyperliquid", "bybit", "okx"`)
	}
	if traderID == "" {
		return nil, nil, invalid("traderId", "trader id is required")
	}
	if err := opts.validate(); err != nil {
		return nil, nil, err
	}
	q := url.Values{}
	if opts != nil {
		if opts.Page != 0 {
			q.Set("page", strconv.Itoa(opts.Page))
		}
		if opts.Limit != 0 {
			q.Set("limit", strconv.Itoa(opts.Limit))
		}
	}
	path := "/v1/smart-money/traders/" + url.PathEscape(string(source)) + "/" + url.PathEscape(traderID) + "/history"
	var out PositionList
	resp, err := s.client.get(ctx, path, q, &out)
	if err != nil {
		return nil, resp, err
	}
	resp.Pagination = out.Pagination
	return &out, resp, nil
}

// AllTraderHistory returns an iterator over every closed position of a
// trader, fetching pages lazily; see [SignalsService.All] for the usage
// pattern.
func (s *SmartMoneyService) AllTraderHistory(ctx context.Context, source Source, traderID string, opts *TraderHistoryOptions) iter.Seq2[*Position, error] {
	return func(yield func(*Position, error) bool) {
		o := TraderHistoryOptions{Page: 1}
		if opts != nil {
			o = *opts
		}
		if o.Page == 0 {
			o.Page = 1
		}
		if o.Limit == 0 {
			o.Limit = 50 // the API's own page maximum keeps the call count low
		}
		for {
			page, _, err := s.TraderHistory(ctx, source, traderID, &o)
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
			o.Page++
		}
	}
}

// PositionsOptions filters [SmartMoneyService.Positions]. Zero fields are
// not sent, so the API defaults apply.
type PositionsOptions struct {
	// Status filters open/closed positions (default all).
	Status PositionStatus
	// Source filters by exchange (default all).
	Source Source
	// Symbol returns only positions on this coin ("BTC", "ETHUSDT").
	Symbol string
	// Limit is positions per page, 1–50 (default 20).
	Limit int
	// Offset is positions to skip, 0–10000 (default 0).
	Offset int
}

// validate implements client-side validation before the request.
func (o *PositionsOptions) validate() error {
	if o == nil {
		return nil
	}
	if err := checkEnum("status", string(o.Status), o.Status.Valid()); err != nil {
		return err
	}
	if err := checkEnum("source", string(o.Source), o.Source.Valid()); err != nil {
		return err
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
	return nil
}

// Positions returns open and closed positions of the tracked traders,
// most recently changed first.
//
// GET /v1/smart-money/positions —
// docs: https://traderspy.app/api ("Smart-money positions")
func (s *SmartMoneyService) Positions(ctx context.Context, opts *PositionsOptions) (*PositionList, *Response, error) {
	if err := opts.validate(); err != nil {
		return nil, nil, err
	}
	q := url.Values{}
	if opts != nil {
		if opts.Status != "" {
			q.Set("status", string(opts.Status))
		}
		if opts.Source != "" {
			q.Set("source", string(opts.Source))
		}
		if opts.Symbol != "" {
			q.Set("symbol", opts.Symbol)
		}
		if opts.Limit != 0 {
			q.Set("limit", strconv.Itoa(opts.Limit))
		}
		if opts.Offset != 0 {
			q.Set("offset", strconv.Itoa(opts.Offset))
		}
	}
	var out PositionList
	resp, err := s.client.get(ctx, "/v1/smart-money/positions", q, &out)
	if err != nil {
		return nil, resp, err
	}
	resp.Pagination = out.Pagination
	return &out, resp, nil
}

// AllPositions returns an iterator over every position matching the
// filters, fetching offset pages lazily; see [SignalsService.All] for the
// usage pattern.
func (s *SmartMoneyService) AllPositions(ctx context.Context, opts *PositionsOptions) iter.Seq2[*Position, error] {
	return func(yield func(*Position, error) bool) {
		o := PositionsOptions{}
		if opts != nil {
			o = *opts
		}
		if o.Limit == 0 {
			o.Limit = 50 // the API's own page maximum keeps the call count low
		}
		for {
			page, _, err := s.Positions(ctx, &o)
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

// SmartMoneyStatsOptions filters [SmartMoneyService.Stats]. Zero fields are
// not sent, so the API defaults apply.
type SmartMoneyStatsOptions struct {
	// Source is the exchange to read, or "all" (default).
	Source Source
	// Period is the window for the closed-position statistics.
	Period Period
}

// validate implements client-side validation before the request.
func (o *SmartMoneyStatsOptions) validate() error {
	if o == nil {
		return nil
	}
	if err := checkEnum("source", string(o.Source), o.Source.Valid()); err != nil {
		return err
	}
	return checkEnum("period", string(o.Period), o.Period.Valid())
}

// Stats returns aggregate smart-money statistics: open positions, the
// long/short split, and the PnL and win rate of closed positions over the
// period.
//
// GET /v1/smart-money/stats —
// docs: https://traderspy.app/api ("Smart-money statistics")
func (s *SmartMoneyService) Stats(ctx context.Context, opts *SmartMoneyStatsOptions) (*SmartMoneyStats, *Response, error) {
	if err := opts.validate(); err != nil {
		return nil, nil, err
	}
	q := url.Values{}
	if opts != nil {
		if opts.Source != "" {
			q.Set("source", string(opts.Source))
		}
		if opts.Period != "" {
			q.Set("period", string(opts.Period))
		}
	}
	var out SmartMoneyStats
	resp, err := s.client.getData(ctx, "/v1/smart-money/stats", q, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// Exchanges returns the exchanges smart-money tracking covers. It takes no
// parameters.
//
// GET /v1/smart-money/exchanges — docs: https://traderspy.app/api
// ("Tracked exchanges")
func (s *SmartMoneyService) Exchanges(ctx context.Context) (*ExchangeList, *Response, error) {
	var out ExchangeList
	resp, err := s.client.get(ctx, "/v1/smart-money/exchanges", nil, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}
