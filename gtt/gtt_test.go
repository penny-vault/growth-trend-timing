package gtt_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/penny-vault/growth-trend-timing/gtt"
	"github.com/penny-vault/pvbt/asset"
	"github.com/penny-vault/pvbt/data"
	"github.com/penny-vault/pvbt/engine"
	"github.com/penny-vault/pvbt/portfolio"
)

var _ = Describe("GrowthTrendTiming", func() {
	var (
		ctx       context.Context
		snap      *data.SnapshotProvider
		nyc       *time.Location
		startDate time.Time
		endDate   time.Time
	)

	BeforeEach(func() {
		ctx = context.Background()

		var err error
		nyc, err = time.LoadLocation("America/New_York")
		Expect(err).NotTo(HaveOccurred())

		snap, err = data.NewSnapshotProvider("testdata/snapshot.db")
		Expect(err).NotTo(HaveOccurred())

		startDate = time.Date(2024, 6, 1, 0, 0, 0, 0, nyc)
		endDate = time.Date(2026, 3, 1, 0, 0, 0, 0, nyc)
	})

	AfterEach(func() {
		if snap != nil {
			snap.Close()
		}
	})

	runBacktest := func() portfolio.Portfolio {
		strategy := &gtt.GrowthTrendTiming{}
		acct := portfolio.New(
			portfolio.WithCash(100000, startDate),
			portfolio.WithAllMetrics(),
		)

		eng := engine.New(strategy,
			engine.WithDataProvider(snap),
			engine.WithAssetProvider(snap),
			engine.WithAccount(acct),
		)

		result, err := eng.Backtest(ctx, startDate, endDate)
		Expect(err).NotTo(HaveOccurred())
		return result
	}

	It("produces expected returns and risk metrics", func() {
		result := runBacktest()

		summary, err := result.Summary()
		Expect(err).NotTo(HaveOccurred())
		Expect(summary.TWRR).To(BeNumerically("~", 0.2259, 0.01))
		Expect(summary.MaxDrawdown).To(BeNumerically(">", -0.11), "max drawdown should be better than -11%")

		Expect(result.Value()).To(BeNumerically("~", 122592, 500))
	})

	It("outperforms on a risk-adjusted basis vs buy-and-hold drawdown", func() {
		result := runBacktest()

		summary, err := result.Summary()
		Expect(err).NotTo(HaveOccurred())

		// Strategy should have significantly lower drawdown than benchmark (-10% vs -19%)
		Expect(summary.MaxDrawdown).To(BeNumerically(">", -0.10))
	})

	It("switches to defensive allocation when unemployment rises and price is below SMA", func() {
		result := runBacktest()
		txns := result.Transactions()

		tickers := map[string]bool{}
		for _, tx := range txns {
			if tx.Type == asset.BuyTransaction || tx.Type == asset.SellTransaction {
				tickers[tx.Asset.Ticker] = true
			}
		}

		Expect(tickers).To(HaveKey("SPY"), "should hold SPY in risk-on mode")
		Expect(tickers).To(HaveKey("BIL"), "should hold BIL in defensive mode")
	})

	It("produces the expected trade sequence", func() {
		result := runBacktest()
		txns := result.Transactions()

		type trade struct {
			date   string
			txType asset.TransactionType
			ticker string
		}

		var trades []trade
		for _, tx := range txns {
			if tx.Type == asset.BuyTransaction || tx.Type == asset.SellTransaction {
				trades = append(trades, trade{
					date:   tx.Date.In(nyc).Format("2006-01-02"),
					txType: tx.Type,
					ticker: tx.Asset.Ticker,
				})
			}
		}

		expected := []trade{
			{"2024-06-28", asset.BuyTransaction, "SPY"},
			{"2024-09-30", asset.BuyTransaction, "SPY"},
			{"2025-03-31", asset.SellTransaction, "SPY"},
			{"2025-03-31", asset.BuyTransaction, "BIL"},
			{"2025-04-30", asset.BuyTransaction, "BIL"},
			{"2025-05-30", asset.SellTransaction, "BIL"},
			{"2025-05-30", asset.BuyTransaction, "SPY"},
			{"2025-09-30", asset.BuyTransaction, "SPY"},
		}

		Expect(trades).To(HaveLen(len(expected)))
		for ii, exp := range expected {
			Expect(trades[ii].date).To(Equal(exp.date), "trade %d date", ii)
			Expect(trades[ii].txType).To(Equal(exp.txType), "trade %d type", ii)
			Expect(trades[ii].ticker).To(Equal(exp.ticker), "trade %d ticker", ii)
		}
	})
})
