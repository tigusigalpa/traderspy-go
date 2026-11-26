// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Igor Sazonov

package traderspy

// Interval is a candle timeframe used by market data, technicals, the
// screener and backtests. Note it is a different enumeration than
// [TimeRange] (trader performance windows), [Period] (statistics windows)
// and [Timeframe] (signal timeframes); the API treats them as separate
// parameter domains even where values look alike.
type Interval string

// Candle intervals.
const (
	Interval1m  Interval = "1m"
	Interval5m  Interval = "5m"
	Interval15m Interval = "15m"
	Interval1h  Interval = "1h"
	Interval4h  Interval = "4h"
	Interval1d  Interval = "1d"
)

// Valid reports whether i is one of the known candle intervals.
func (i Interval) Valid() bool {
	switch i {
	case Interval1m, Interval5m, Interval15m, Interval1h, Interval4h, Interval1d:
		return true
	}
	return false
}

// Timeframe is the candle timeframe of an AI signal (the `timeframe` filter
// of GET /v1/signals). A narrower set than [Interval].
type Timeframe string

// Signal timeframes.
const (
	Timeframe15m Timeframe = "15m"
	Timeframe1h  Timeframe = "1h"
	Timeframe4h  Timeframe = "4h"
	Timeframe1d  Timeframe = "1d"
)

// Valid reports whether t is one of the known signal timeframes.
func (t Timeframe) Valid() bool {
	switch t {
	case Timeframe15m, Timeframe1h, Timeframe4h, Timeframe1d:
		return true
	}
	return false
}

// TimeRange is the performance window of a smart-money leaderboard
// (the `timeRange` parameter of the traders endpoints).
type TimeRange string

// Trader performance windows.
const (
	TimeRange24h TimeRange = "24h"
	TimeRange3D  TimeRange = "3D"
	TimeRange7D  TimeRange = "7D"
	TimeRange30D TimeRange = "30D"
)

// Valid reports whether t is one of the known time ranges.
func (t TimeRange) Valid() bool {
	switch t {
	case TimeRange24h, TimeRange3D, TimeRange7D, TimeRange30D:
		return true
	}
	return false
}

// Period is the statistics window of the signals and smart-money stats
// endpoints (the `period` parameter).
type Period string

// Statistics windows.
const (
	Period4h  Period = "4h"
	Period8h  Period = "8h"
	Period24h Period = "24h"
	Period7d  Period = "7d"
)

// Valid reports whether p is one of the known statistics periods.
func (p Period) Valid() bool {
	switch p {
	case Period4h, Period8h, Period24h, Period7d:
		return true
	}
	return false
}

// Importance filters AI signals by importance (the `importance` parameter
// of GET /v1/signals). "high" returns only high-importance signals,
// "medium" returns high + medium.
type Importance string

// Importance levels.
const (
	ImportanceAll    Importance = "all"
	ImportanceHigh   Importance = "high"
	ImportanceMedium Importance = "medium"
	ImportanceLow    Importance = "low"
)

// Valid reports whether i is one of the known importance levels.
func (i Importance) Valid() bool {
	switch i {
	case ImportanceAll, ImportanceHigh, ImportanceMedium, ImportanceLow:
		return true
	}
	return false
}

// SignalAction is the direction of an AI signal: "buy" (long) or
// "sell" (short).
type SignalAction string

// Signal actions.
const (
	ActionBuy  SignalAction = "buy"
	ActionSell SignalAction = "sell"
)

// Source is one of the exchanges smart-money tracking covers, or "all"
// where a filter accepts every exchange. Path parameters never accept
// "all"; use [Source.PathValid] to check.
type Source string

// Tracked exchanges and the cross-exchange value.
const (
	SourceAll         Source = "all"
	SourceBinance     Source = "binance"
	SourceHyperliquid Source = "hyperliquid"
	SourceBybit       Source = "bybit"
	SourceOKX         Source = "okx"
)

// Valid reports whether s is a known source filter value (including "all").
func (s Source) Valid() bool {
	switch s {
	case SourceAll, SourceBinance, SourceHyperliquid, SourceBybit, SourceOKX:
		return true
	}
	return false
}

// PathValid reports whether s can be used as a path parameter, where the
// API requires a concrete exchange ("all" is not allowed).
func (s Source) PathValid() bool {
	switch s {
	case SourceBinance, SourceHyperliquid, SourceBybit, SourceOKX:
		return true
	}
	return false
}

// RankingType selects the leaderboard snapshot of a traders endpoint:
// ranked by return on investment or by absolute profit.
type RankingType string

// Ranking types.
const (
	RankingROI RankingType = "ROI"
	RankingPNL RankingType = "PNL"
)

// Valid reports whether r is one of the known ranking types.
func (r RankingType) Valid() bool {
	switch r {
	case RankingROI, RankingPNL:
		return true
	}
	return false
}

// TraderSortBy is the sort metric of the top-traders leaderboard. The empty
// value means "the ranking type" (the API default).
type TraderSortBy string

// Leaderboard sort metrics.
const (
	SortByROI   TraderSortBy = "ROI"
	SortByPNL   TraderSortBy = "PNL"
	SortByScore TraderSortBy = "SCORE"
)

// Valid reports whether s is one of the known leaderboard sort metrics.
func (s TraderSortBy) Valid() bool {
	switch s {
	case SortByROI, SortByPNL, SortByScore:
		return true
	}
	return false
}

// Order is the sort direction of the top-traders leaderboard (uppercase in
// the API). The screener uses the lowercase [SortOrder] instead.
type Order string

// Leaderboard sort directions.
const (
	OrderDesc Order = "DESC"
	OrderAsc  Order = "ASC"
)

// Valid reports whether o is one of the known leaderboard directions.
func (o Order) Valid() bool {
	switch o {
	case OrderDesc, OrderAsc:
		return true
	}
	return false
}

// PositionStatus filters smart-money positions (the `status` parameter of
// GET /v1/smart-money/positions).
type PositionStatus string

// Position status filters.
const (
	StatusAll    PositionStatus = "all"
	StatusOpen   PositionStatus = "open"
	StatusClosed PositionStatus = "closed"
)

// Valid reports whether s is one of the known status filters.
func (s PositionStatus) Valid() bool {
	switch s {
	case StatusAll, StatusOpen, StatusClosed:
		return true
	}
	return false
}

// Assets selects which perpetuals the screener's most-traded list holds:
// "all", or "crypto" to leave out stock, commodity and index perpetuals.
type Assets string

// Screener asset universes.
const (
	AssetsAll    Assets = "all"
	AssetsCrypto Assets = "crypto"
)

// Valid reports whether a is one of the known asset universes.
func (a Assets) Valid() bool {
	switch a {
	case AssetsAll, AssetsCrypto:
		return true
	}
	return false
}

// ScreenerSortBy is the row ordering of a screener result. The API default
// is "metric" when conditions are given and "volume" otherwise.
type ScreenerSortBy string

// Screener sort fields.
const (
	ScreenerSortVolume    ScreenerSortBy = "volume"
	ScreenerSortChange24h ScreenerSortBy = "change24h"
	ScreenerSortMetric    ScreenerSortBy = "metric"
)

// Valid reports whether s is one of the known screener sort fields.
func (s ScreenerSortBy) Valid() bool {
	switch s {
	case ScreenerSortVolume, ScreenerSortChange24h, ScreenerSortMetric:
		return true
	}
	return false
}

// SortOrder is the sort direction of a screener result (lowercase in the
// API). The traders leaderboard uses the uppercase [Order] instead.
type SortOrder string

// Screener sort directions.
const (
	SortOrderDesc SortOrder = "desc"
	SortOrderAsc  SortOrder = "asc"
)

// Valid reports whether s is one of the known screener directions.
func (s SortOrder) Valid() bool {
	switch s {
	case SortOrderDesc, SortOrderAsc:
		return true
	}
	return false
}

// Metric is a technical metric the screener and backtest conditions can be
// built on.
type Metric string

// Screener/backtest condition metrics.
const (
	MetricRSI           Metric = "rsi"
	MetricStochastic    Metric = "stochastic"
	MetricCCI           Metric = "cci"
	MetricMFI           Metric = "mfi"
	MetricWilliamsR     Metric = "williamsR"
	MetricADX           Metric = "adx"
	MetricROC           Metric = "roc"
	MetricMACDHistogram Metric = "macdHistogram"
	MetricATRPct        Metric = "atrPct"
	MetricVolumeRatio   Metric = "volumeRatio"
	MetricBBPercentB    Metric = "bbPercentB"
	MetricBBWidthPct    Metric = "bbWidthPct"
	MetricPriceVsEMA    Metric = "priceVsEma"
	MetricEMASpread     Metric = "emaSpread"
	MetricSupertrend    Metric = "supertrend"
	MetricChangePct     Metric = "changePct"
	MetricPrice         Metric = "price"
)

// Valid reports whether m is one of the known condition metrics.
func (m Metric) Valid() bool {
	switch m {
	case MetricRSI, MetricStochastic, MetricCCI, MetricMFI, MetricWilliamsR,
		MetricADX, MetricROC, MetricMACDHistogram, MetricATRPct, MetricVolumeRatio,
		MetricBBPercentB, MetricBBWidthPct, MetricPriceVsEMA, MetricEMASpread,
		MetricSupertrend, MetricChangePct, MetricPrice:
		return true
	}
	return false
}

// Op is the comparison operator of a screener/backtest condition.
// "lt"/"gt" compare the latest bar; "crossAbove"/"crossBelow" are one-bar
// events.
type Op string

// Condition operators.
const (
	OpLT         Op = "lt"
	OpGT         Op = "gt"
	OpCrossAbove Op = "crossAbove"
	OpCrossBelow Op = "crossBelow"
)

// Valid reports whether o is one of the known condition operators.
func (o Op) Valid() bool {
	switch o {
	case OpLT, OpGT, OpCrossAbove, OpCrossBelow:
		return true
	}
	return false
}

// PositionSide is the direction of a smart-money position.
type PositionSide string

// Position directions.
const (
	SideLong  PositionSide = "long"
	SideShort PositionSide = "short"
)
