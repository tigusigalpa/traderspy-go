// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Igor Sazonov

package traderspy

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

// MarketService groups the market-data endpoints: live prices, OHLCV
// candles, tracked symbols and derivatives (funding, open interest). Obtain
// it from [Client.Market].
//
// Docs: https://traderspy.app/api ("Live prices", "Candles (OHLCV)",
// "Tracked symbols", "Funding & open interest")
type MarketService struct {
	client *Client
}

// Price is the live snapshot of one futures symbol: last price, 24h
// high/low, volume and change. Prices are float64 as the API sends them;
// fields not mapped in this SDK version are kept in Extra.
type Price struct {
	// Symbol is the trading pair (e.g. "BTCUSDT").
	Symbol string `json:"symbol"`
	// Price is the last trade price.
	Price float64 `json:"price"`
	// High24h and Low24h are the 24-hour extremes.
	High24h *float64 `json:"high24h"`
	Low24h  *float64 `json:"low24h"`
	// Volume24h is the 24-hour traded volume.
	Volume24h *float64 `json:"volume24h"`
	// Volume24hUSD is the 24-hour volume in USD, when reported.
	Volume24hUSD *float64 `json:"volume24hUsd"`
	// Change24hPct is the 24-hour price change, in percent.
	Change24hPct *float64 `json:"change24hPct"`

	// Extra holds fields this SDK version does not map yet.
	Extra map[string]json.RawMessage `json:"-"`
}

// UnmarshalJSON implements [json.Unmarshaler].
func (p *Price) UnmarshalJSON(data []byte) error {
	type alias Price
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*p = Price(a)
	p.Extra = extraFields(data, Price{})
	return nil
}

// MarshalJSON implements [json.Marshaler].
func (p Price) MarshalJSON() ([]byte, error) {
	type alias Price
	return marshalWithExtra(alias(p), p.Extra)
}

// PriceList is the answer of GET /v1/market/prices.
type PriceList struct {
	// Data are the prices, one entry per requested symbol.
	Data []Price `json:"data"`
}

// UnmarshalJSON implements [json.Unmarshaler]. It accepts the "data" member
// either as an array of prices or as an object keyed by symbol; an object
// is converted to a slice sorted by symbol.
func (l *PriceList) UnmarshalJSON(data []byte) error {
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(data, &env); err != nil {
		return err
	}
	if len(env.Data) == 0 {
		return json.Unmarshal(data, &l.Data)
	}
	if env.Data[0] == '[' {
		return json.Unmarshal(env.Data, &l.Data)
	}
	var keyed map[string]Price
	if err := json.Unmarshal(env.Data, &keyed); err != nil {
		return err
	}
	l.Data = make([]Price, 0, len(keyed))
	for symbol, p := range keyed {
		if p.Symbol == "" {
			p.Symbol = symbol
		}
		l.Data = append(l.Data, p)
	}
	sort.Slice(l.Data, func(i, j int) bool { return l.Data[i].Symbol < l.Data[j].Symbol })
	return nil
}

// Candle is one OHLCV bar. The last candle of a Candles answer may still be
// open.
type Candle struct {
	// Time is the bar's open time.
	Time UnixMillis `json:"time"`
	// Open, High, Low, Close are the bar's prices.
	Open  float64 `json:"open"`
	High  float64 `json:"high"`
	Low   float64 `json:"low"`
	Close float64 `json:"close"`
	// Volume is the bar's traded volume.
	Volume float64 `json:"volume"`

	// Extra holds fields this SDK version does not map yet.
	Extra map[string]json.RawMessage `json:"-"`
}

// candleKeys are the JSON object forms accepted for a candle.
var candleAliases = map[string]string{
	"time": "time", "t": "time", "openTime": "time", "timestamp": "time",
	"open": "open", "o": "open",
	"high": "high", "h": "high",
	"low": "low", "l": "low",
	"close": "close", "c": "close",
	"volume": "volume", "v": "volume",
}

// UnmarshalJSON implements [json.Unmarshaler]. It accepts both the object
// form (with keys like "time" and "open", or "t" and "o") and
// the positional array form ([time, open, high, low, close, volume]).
func (c *Candle) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if data[0] == '[' {
		var row []json.RawMessage
		if err := json.Unmarshal(data, &row); err != nil {
			return err
		}
		if len(row) < 5 {
			return fmt.Errorf("traderspy: malformed candle array (want [time, open, high, low, close, volume])")
		}
		var candle Candle
		if err := json.Unmarshal(row[0], &candle.Time); err != nil {
			return err
		}
		for i, field := range []*float64{
			&candle.Open, &candle.High, &candle.Low, &candle.Close, &candle.Volume,
		} {
			if i+1 < len(row) {
				if err := decodeCandleNumber(row[i+1], field); err != nil {
					return err
				}
			}
		}
		*c = candle
		return nil
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	normalized := map[string]json.RawMessage{}
	var candle Candle
	for key, value := range raw {
		if field, ok := candleAliases[strings.ToLower(key)]; ok {
			normalized[field] = value
			continue
		}
		if candle.Extra == nil {
			candle.Extra = map[string]json.RawMessage{}
		}
		candle.Extra[key] = value
	}
	if v, ok := normalized["time"]; ok {
		if err := json.Unmarshal(v, &candle.Time); err != nil {
			return err
		}
	}
	for field, ptr := range map[string]*float64{
		"open": &candle.Open, "high": &candle.High, "low": &candle.Low,
		"close": &candle.Close, "volume": &candle.Volume,
	} {
		if v, ok := normalized[field]; ok {
			if err := decodeCandleNumber(v, ptr); err != nil {
				return err
			}
		}
	}
	*c = candle
	return nil
}

func decodeCandleNumber(data []byte, dst *float64) error {
	err := json.Unmarshal(data, dst)
	if err == nil {
		return nil
	}
	var value string
	if json.Unmarshal(data, &value) == nil {
		if n, parseErr := strconv.ParseFloat(value, 64); parseErr == nil && !math.IsNaN(n) && !math.IsInf(n, 0) {
			*dst = n
			return nil
		}
	}
	return err
}

// MarshalJSON implements [json.Marshaler].
func (c Candle) MarshalJSON() ([]byte, error) {
	type alias Candle
	return marshalWithExtra(alias(c), c.Extra)
}

// CandleList is the answer of GET /v1/market/candles.
type CandleList struct {
	// Data are the candles, oldest first; the last one may still be open.
	Data []Candle `json:"data"`
	// Symbol echoes the requested symbol, when reported.
	Symbol string `json:"symbol"`
	// Interval echoes the requested interval, when reported.
	Interval string `json:"interval"`
}

// UnmarshalJSON implements [json.Unmarshaler]. The candles are read from
// "data", or "candles" when "data" is absent.
func (l *CandleList) UnmarshalJSON(data []byte) error {
	type alias CandleList
	var a struct {
		alias
		Candles []Candle `json:"candles"`
	}
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*l = CandleList(a.alias)
	if l.Data == nil {
		l.Data = a.Candles
	}
	return nil
}

// SymbolList is the answer of GET /v1/market/symbols.
type SymbolList struct {
	// Symbols are the tracked futures symbols (e.g. "BTCUSDT").
	Symbols []string `json:"-"`

	// Extra holds fields this SDK version does not map yet.
	Extra map[string]json.RawMessage `json:"-"`
}

// UnmarshalJSON implements [json.Unmarshaler]. Symbols are read from a
// "data" or "symbols" member holding either plain strings or objects with a
// "symbol"/"name" field.
func (l *SymbolList) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	var list json.RawMessage
	for _, key := range []string{"data", "symbols"} {
		if v, ok := raw[key]; ok && len(v) > 0 && v[0] == '[' {
			list = v
			delete(raw, key)
			break
		}
	}
	if list == nil {
		*l = SymbolList{Extra: raw}
		return nil
	}
	var items []json.RawMessage
	if err := json.Unmarshal(list, &items); err != nil {
		return err
	}
	l.Symbols = make([]string, 0, len(items))
	for _, item := range items {
		var s *string
		if err := json.Unmarshal(item, &s); err == nil {
			if s != nil {
				l.Symbols = append(l.Symbols, *s)
			}
			continue
		}
		var obj struct {
			Symbol string `json:"symbol"`
			Name   string `json:"name"`
		}
		if err := json.Unmarshal(item, &obj); err == nil && (obj.Symbol != "" || obj.Name != "") {
			if obj.Symbol != "" {
				l.Symbols = append(l.Symbols, obj.Symbol)
			} else {
				l.Symbols = append(l.Symbols, obj.Name)
			}
		}
	}
	l.Extra = raw
	if len(l.Extra) == 0 {
		l.Extra = nil
	}
	return nil
}

// MarshalJSON implements [json.Marshaler] using the "data" envelope.
func (l SymbolList) MarshalJSON() ([]byte, error) {
	return marshalWithExtra(struct {
		Data []string `json:"data"`
	}{Data: l.Symbols}, l.Extra)
}

// Derivative is the derivatives snapshot of one Binance USDⓈ-M perpetual:
// funding rate, open interest, positioning and plain-language notes.
// Fields not mapped in this SDK version are kept in Extra.
type Derivative struct {
	// Symbol is the perpetual.
	Symbol string `json:"symbol"`
	// FundingRate is the current funding rate (e.g. 0.0001 = 0.01%).
	FundingRate *float64 `json:"fundingRate"`
	// NextFundingTime is when funding next settles, when reported.
	NextFundingTime *UnixMillis `json:"nextFundingTime"`
	// OpenInterest is the open interest in contracts/base asset.
	OpenInterest *float64 `json:"openInterest"`
	// OpenInterestUSD is the open interest in USD, when reported.
	OpenInterestUSD *float64 `json:"openInterestUsd"`
	// LongShortRatio is the account long/short ratio, when reported.
	LongShortRatio *float64 `json:"longShortRatio"`
	// LongPct and ShortPct are the positioning shares, in percent.
	LongPct  *float64 `json:"longPct"`
	ShortPct *float64 `json:"shortPct"`
	// Note is the plain-language summary of the snapshot.
	Note string `json:"note"`

	// Extra holds fields this SDK version does not map yet.
	Extra map[string]json.RawMessage `json:"-"`
}

// UnmarshalJSON implements [json.Unmarshaler].
func (d *Derivative) UnmarshalJSON(data []byte) error {
	type alias Derivative
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*d = Derivative(a)
	d.Extra = extraFields(data, Derivative{})
	return nil
}

// MarshalJSON implements [json.Marshaler].
func (d Derivative) MarshalJSON() ([]byte, error) {
	type alias Derivative
	return marshalWithExtra(alias(d), d.Extra)
}

// DerivativesList is the answer of GET /v1/market/derivatives.
type DerivativesList struct {
	// Data are the derivative snapshots, one per requested symbol.
	Data []Derivative `json:"data"`
}

// UnmarshalJSON implements [json.Unmarshaler], accepting "data" as an
// array or as an object keyed by symbol.
func (l *DerivativesList) UnmarshalJSON(data []byte) error {
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(data, &env); err != nil {
		return err
	}
	if len(env.Data) == 0 {
		return json.Unmarshal(data, &l.Data)
	}
	if env.Data[0] == '[' {
		return json.Unmarshal(env.Data, &l.Data)
	}
	var keyed map[string]Derivative
	if err := json.Unmarshal(env.Data, &keyed); err != nil {
		return err
	}
	l.Data = make([]Derivative, 0, len(keyed))
	for symbol, d := range keyed {
		if d.Symbol == "" {
			d.Symbol = symbol
		}
		l.Data = append(l.Data, d)
	}
	sort.Slice(l.Data, func(i, j int) bool { return l.Data[i].Symbol < l.Data[j].Symbol })
	return nil
}

// Prices returns the live price of up to [MaxPriceSymbols] futures symbols:
// last price, 24h high/low, volume and change. A bare coin ("BTC") reads as
// "BTCUSDT"; the SDK normalizes symbols before sending.
//
// GET /v1/market/prices — docs: https://traderspy.app/api ("Live prices")
func (s *MarketService) Prices(ctx context.Context, symbols []string) (*PriceList, *Response, error) {
	if err := ValidateSymbolList(symbols, MaxPriceSymbols); err != nil {
		return nil, nil, err
	}
	q := url.Values{"symbols": {strings.Join(NormalizeSymbols(symbols), ",")}}
	var out PriceList
	resp, err := s.client.get(ctx, "/v1/market/prices", q, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// CandlesOptions are the parameters of [MarketService.Candles].
type CandlesOptions struct {
	// Symbol is the trading pair, required ("BTCUSDT" or a bare "BTC").
	Symbol string
	// Interval is the candle timeframe (default 1h).
	Interval Interval
	// Limit is the number of newest candles, 1–500 (default 100).
	Limit int
}

// validate implements client-side validation before the request.
func (o *CandlesOptions) validate() error {
	if o == nil || strings.TrimSpace(o.Symbol) == "" {
		return invalid("symbol", "symbol is required")
	}
	if err := checkEnum("interval", string(o.Interval), o.Interval.Valid()); err != nil {
		return err
	}
	if o.Limit != 0 {
		if err := checkRange("limit", o.Limit, 1, 500); err != nil {
			return err
		}
	}
	return nil
}

// Candles returns OHLCV candles of a futures symbol, oldest first; the last
// one may still be open.
//
// GET /v1/market/candles — docs: https://traderspy.app/api ("Candles (OHLCV)")
func (s *MarketService) Candles(ctx context.Context, opts *CandlesOptions) (*CandleList, *Response, error) {
	if err := opts.validate(); err != nil {
		return nil, nil, err
	}
	q := url.Values{"symbol": {NormalizeSymbol(opts.Symbol)}}
	if opts.Interval != "" {
		q.Set("interval", string(opts.Interval))
	}
	if opts.Limit != 0 {
		q.Set("limit", strconv.Itoa(opts.Limit))
	}
	var out CandleList
	resp, err := s.client.get(ctx, "/v1/market/candles", q, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// Symbols returns every futures symbol TraderSpy keeps live candles for.
// It takes no parameters.
//
// GET /v1/market/symbols — docs: https://traderspy.app/api ("Tracked symbols")
func (s *MarketService) Symbols(ctx context.Context) (*SymbolList, *Response, error) {
	var out SymbolList
	resp, err := s.client.get(ctx, "/v1/market/symbols", nil, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// Derivatives returns funding rate, open interest and positioning for up to
// [MaxDerivativeSymbols] Binance USDⓈ-M perpetuals, with plain-language
// notes.
//
// GET /v1/market/derivatives —
// docs: https://traderspy.app/api ("Funding & open interest")
func (s *MarketService) Derivatives(ctx context.Context, symbols []string) (*DerivativesList, *Response, error) {
	if err := ValidateSymbolList(symbols, MaxDerivativeSymbols); err != nil {
		return nil, nil, err
	}
	q := url.Values{"symbols": {strings.Join(NormalizeSymbols(symbols), ",")}}
	var out DerivativesList
	resp, err := s.client.get(ctx, "/v1/market/derivatives", q, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}
