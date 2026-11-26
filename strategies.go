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

// StrategiesService groups the endpoints that read your own TraderSpy alert
// strategies and the alerts they sent. They need a key allowed to read your
// strategies (an option in the key's settings); otherwise the API answers
// 403 "strategies_not_allowed". Obtain it from [Client.Strategies].
//
// Docs: https://traderspy.app/api ("Your strategies", "Strategy alerts")
type StrategiesService struct {
	client *Client
}

// Strategy is an alert strategy built on TraderSpy: its rules in words, the
// timeframe and coins it watches, whether it runs and how many alerts it
// has sent. Fields not mapped in this SDK version are kept in Extra.
type Strategy struct {
	// ID is the strategy identifier; pass it to
	// [StrategyAlertsOptions.StrategyID].
	ID string `json:"id"`
	// Name is the strategy's display name.
	Name string `json:"name"`
	// Rules describes the alert conditions in words.
	Rules string `json:"rules"`
	// Timeframe is the candle timeframe the strategy watches.
	Timeframe string `json:"timeframe"`
	// Coins are the symbols the strategy watches.
	Coins []string `json:"coins"`
	// Enabled reports whether the strategy is running.
	Enabled bool `json:"enabled"`
	// AlertsSent is how many alerts the strategy has sent, when reported.
	AlertsSent int `json:"alertsSent"`
	// CreatedAt is when the strategy was created.
	CreatedAt Time `json:"createdAt"`

	// Extra holds fields this SDK version does not map yet.
	Extra map[string]json.RawMessage `json:"-"`
}

// UnmarshalJSON implements [json.Unmarshaler].
func (s *Strategy) UnmarshalJSON(data []byte) error {
	type alias Strategy
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*s = Strategy(a)
	s.Extra = extraFields(data, Strategy{})
	return nil
}

// MarshalJSON implements [json.Marshaler].
func (s Strategy) MarshalJSON() ([]byte, error) {
	type alias Strategy
	return marshalWithExtra(alias(s), s.Extra)
}

// StrategyList is the answer of GET /v1/strategies.
type StrategyList struct {
	// Data are the strategies, newest first.
	Data []Strategy `json:"data"`
	// Pagination describes the page, when reported.
	Pagination *Pagination `json:"pagination"`
}

// StrategyAlert is one alert a strategy sent: the coin, timeframe, price
// and the rules that matched. Fields not mapped in this SDK version are
// kept in Extra.
type StrategyAlert struct {
	// ID is the alert identifier.
	ID string `json:"id"`
	// StrategyID is the strategy that sent the alert.
	StrategyID string `json:"strategyId"`
	// Coin is the symbol the alert is on.
	Coin string `json:"coin"`
	// Timeframe is the candle timeframe that matched.
	Timeframe string `json:"timeframe"`
	// Price is the price when the alert fired.
	Price float64 `json:"price"`
	// Rules are the rules that matched, when reported.
	Rules []string `json:"rules"`
	// CreatedAt is when the alert was sent; pass the newest one you have
	// as StrategyAlertsOptions.Since to poll for new alerts.
	CreatedAt Time `json:"createdAt"`

	// Extra holds fields this SDK version does not map yet.
	Extra map[string]json.RawMessage `json:"-"`
}

// UnmarshalJSON implements [json.Unmarshaler].
func (a *StrategyAlert) UnmarshalJSON(data []byte) error {
	type alias StrategyAlert
	var b alias
	if err := json.Unmarshal(data, &b); err != nil {
		return err
	}
	*a = StrategyAlert(b)
	a.Extra = extraFields(data, StrategyAlert{})
	return nil
}

// MarshalJSON implements [json.Marshaler].
func (a StrategyAlert) MarshalJSON() ([]byte, error) {
	type alias StrategyAlert
	return marshalWithExtra(alias(a), a.Extra)
}

// StrategyAlertList is one page of strategy alerts.
type StrategyAlertList struct {
	// Data are the alerts, newest first.
	Data []StrategyAlert `json:"data"`
	// Pagination describes the page.
	Pagination *Pagination `json:"pagination"`
}

// List returns the strategies you built on TraderSpy, newest first: their
// rules in words, timeframe, coins, whether they run, and the alerts sent
// so far.
//
// GET /v1/strategies — docs: https://traderspy.app/api ("Your strategies")
func (s *StrategiesService) List(ctx context.Context) (*StrategyList, *Response, error) {
	var out StrategyList
	resp, err := s.client.get(ctx, "/v1/strategies", nil, &out)
	if err != nil {
		return nil, resp, err
	}
	resp.Pagination = out.Pagination
	return &out, resp, nil
}

// StrategyAlertsOptions filters [StrategiesService.Alerts]. Zero fields are
// not sent, so the API defaults apply.
type StrategyAlertsOptions struct {
	// Limit is alerts per page, 1–100 (default 20).
	Limit int
	// Offset is alerts to skip, 0–10000 (default 0).
	Offset int
	// Since returns only alerts sent after this time. To poll for new
	// alerts, pass the newest CreatedAt you already have.
	Since time.Time
	// StrategyID returns only the alerts of this strategy (an ID from
	// [StrategiesService.List]).
	StrategyID string
	// Coin returns only alerts on this coin ("BTC", "ETHUSDT").
	Coin string
}

// validate implements client-side validation before the request.
func (o *StrategyAlertsOptions) validate() error {
	if o == nil {
		return nil
	}
	if o.Limit != 0 {
		if err := checkRange("limit", o.Limit, 1, 100); err != nil {
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

// Alerts returns the alerts your strategies sent, newest first: coin,
// timeframe, price and the rules that matched.
//
// GET /v1/strategies/alerts — docs: https://traderspy.app/api ("Strategy alerts")
func (s *StrategiesService) Alerts(ctx context.Context, opts *StrategyAlertsOptions) (*StrategyAlertList, *Response, error) {
	if err := opts.validate(); err != nil {
		return nil, nil, err
	}
	q := url.Values{}
	if opts != nil {
		if opts.Limit != 0 {
			q.Set("limit", strconv.Itoa(opts.Limit))
		}
		if opts.Offset != 0 {
			q.Set("offset", strconv.Itoa(opts.Offset))
		}
		if !opts.Since.IsZero() {
			q.Set("since", opts.Since.UTC().Format(time.RFC3339))
		}
		if opts.StrategyID != "" {
			q.Set("strategyId", opts.StrategyID)
		}
		if opts.Coin != "" {
			q.Set("coin", opts.Coin)
		}
	}
	var out StrategyAlertList
	resp, err := s.client.get(ctx, "/v1/strategies/alerts", q, &out)
	if err != nil {
		return nil, resp, err
	}
	resp.Pagination = out.Pagination
	return &out, resp, nil
}

// AllAlerts returns an iterator over every strategy alert matching the
// filters, fetching offset pages lazily; see [SignalsService.All] for the
// usage pattern.
func (s *StrategiesService) AllAlerts(ctx context.Context, opts *StrategyAlertsOptions) iter.Seq2[*StrategyAlert, error] {
	return func(yield func(*StrategyAlert, error) bool) {
		o := StrategyAlertsOptions{}
		if opts != nil {
			o = *opts
		}
		if o.Limit == 0 {
			o.Limit = 100 // the API's own page maximum keeps the call count low
		}
		for {
			page, _, err := s.Alerts(ctx, &o)
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
