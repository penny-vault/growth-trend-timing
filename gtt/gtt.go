// Copyright 2021-2026
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gtt

import (
	"context"
	_ "embed"
	"fmt"
	"math"
	"time"

	"github.com/penny-vault/pvbt/asset"
	"github.com/penny-vault/pvbt/data"
	"github.com/penny-vault/pvbt/engine"
	"github.com/penny-vault/pvbt/portfolio"
	"github.com/penny-vault/pvbt/universe"
)

//go:embed README.md
var description string

// GrowthTrendTiming combines unemployment rate trend with price trend to
// determine equity exposure. Only goes defensive when BOTH unemployment is
// rising and price is below its moving average.
type GrowthTrendTiming struct {
	EquityUniverse     universe.Universe `pvbt:"equity" desc:"Equity asset to hold in risk-on mode" default:"SPY" suggest:"default=SPY"`
	CashTicker         string            `pvbt:"cash-ticker" desc:"Cash asset for defensive allocation" default:"BIL" suggest:"default=BIL"`
	UnemploymentTicker string            `pvbt:"unemployment-ticker" desc:"Unemployment rate data series" default:"UNRATE" suggest:"default=UNRATE"`
	PriceSMALength     int               `pvbt:"price-sma-length" desc:"Moving average period in months for price trend" default:"10" suggest:"default=10"`
}

func (s *GrowthTrendTiming) Name() string {
	return "Growth-Trend Timing"
}

func (s *GrowthTrendTiming) Setup(_ *engine.Engine) {}

func (s *GrowthTrendTiming) Describe() engine.StrategyDescription {
	return engine.StrategyDescription{
		ShortCode:   "gtt",
		Description: description,
		Source:      "https://www.philosophicaleconomics.com/2016/02/uetrend/",
		Version:     "1.0.0",
		VersionDate: time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
		Schedule:    "@monthend",
		Benchmark:   "VFINX",
	}
}

func (s *GrowthTrendTiming) Compute(ctx context.Context, eng *engine.Engine, strategyPortfolio portfolio.Portfolio, batch *portfolio.Batch) error {
	// 1. Fetch equity prices for the SMA calculation.
	equityDF, err := s.EquityUniverse.Window(ctx, portfolio.Months(s.PriceSMALength+1), data.MetricClose)
	if err != nil {
		return fmt.Errorf("failed to fetch equity prices: %w", err)
	}

	// 2. Fetch unemployment rate data for the trend calculation.
	//    Need 13 months of history to compute a 12-month rolling average plus the current value.
	unrate := eng.Asset(s.UnemploymentTicker)
	unemploymentUniverse := eng.Universe(unrate)

	unemploymentDF, err := unemploymentUniverse.Window(ctx, portfolio.Months(13), data.MetricClose)
	if err != nil {
		return fmt.Errorf("failed to fetch unemployment data: %w", err)
	}

	// 3. Downsample equity to monthly for SMA calculation.
	equityMonthly := equityDF.Downsample(data.Monthly).Last()

	if equityMonthly.Len() < s.PriceSMALength+1 {
		return nil
	}

	// Downsample unemployment to monthly.
	unemploymentMonthly := unemploymentDF.Downsample(data.Monthly).Last()

	if unemploymentMonthly.Len() < 13 {
		return nil
	}

	// 4. Check unemployment trend: is current rate above its 12-month average?
	unemploymentSMA := unemploymentMonthly.Rolling(12).Mean()
	unemploymentSMA = unemploymentSMA.Drop(math.NaN()).Last()
	currentUnemployment := unemploymentMonthly.Last()

	if unemploymentSMA.Len() == 0 || currentUnemployment.Len() == 0 {
		return nil
	}

	currentUnemploymentRate := currentUnemployment.Value(unrate, data.MetricClose)
	averageUnemploymentRate := unemploymentSMA.Value(unrate, data.MetricClose)
	unemploymentRising := currentUnemploymentRate > averageUnemploymentRate

	// 5. Check price trend: is equity price below its SMA?
	priceSMA := equityMonthly.Rolling(s.PriceSMALength).Mean()
	priceSMA = priceSMA.Drop(math.NaN()).Last()
	currentPrice := equityMonthly.Last()

	if priceSMA.Len() == 0 || currentPrice.Len() == 0 {
		return nil
	}

	equityAsset := currentPrice.AssetList()[0]
	currentPriceValue := currentPrice.Value(equityAsset, data.MetricClose)
	smaPriceValue := priceSMA.Value(equityAsset, data.MetricClose)
	priceBelowSMA := currentPriceValue < smaPriceValue

	// 6. Annotate decision inputs.
	batch.Annotate("unemployment-rate", fmt.Sprintf("%.1f%%", currentUnemploymentRate))
	batch.Annotate("unemployment-sma12", fmt.Sprintf("%.1f%%", averageUnemploymentRate))
	batch.Annotate("unemployment-rising", fmt.Sprintf("%t", unemploymentRising))
	batch.Annotate("price", fmt.Sprintf("%.2f", currentPriceValue))
	batch.Annotate(fmt.Sprintf("price-sma%d", s.PriceSMALength), fmt.Sprintf("%.2f", smaPriceValue))
	batch.Annotate("price-below-sma", fmt.Sprintf("%t", priceBelowSMA))

	// 7. Decision: go defensive only when BOTH signals are bearish.
	var selectedAsset asset.Asset

	var justification string

	cashAsset := eng.Asset(s.CashTicker)

	if unemploymentRising && priceBelowSMA {
		selectedAsset = cashAsset
		justification = fmt.Sprintf("defensive: UE %.1f%% > avg %.1f%% AND price %.2f < SMA %.2f",
			currentUnemploymentRate, averageUnemploymentRate, currentPriceValue, smaPriceValue)
	} else {
		selectedAsset = equityAsset
		justification = "risk-on: growth-trend favorable"
	}

	batch.Annotate("justification", justification)

	allocation := portfolio.Allocation{
		Date:          eng.CurrentDate(),
		Members:       map[asset.Asset]float64{selectedAsset: 1.0},
		Justification: justification,
	}

	if err := batch.RebalanceTo(ctx, allocation); err != nil {
		return fmt.Errorf("rebalance failed: %w", err)
	}

	return nil
}
