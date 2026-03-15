# Growth-Trend Timing

The **Growth-Trend Timing** (GTT) strategy was originally proposed by [Philosophical Economics](https://www.philosophicaleconomics.com/2016/02/uetrend/) and later incorporated into Wouter Keller's Lethargic Asset Allocation (LAA). It combines an economic recession indicator (unemployment rate trend) with a price trend filter (10-month SMA) to determine when to be invested in equities versus cash.

The key insight is that price-based trend following works well during recessions but generates too many false signals outside of them. By only activating the price trend filter when the economy signals recession, the strategy avoids most whipsaw trades.

## Rules

1. On the last trading day of the month, check two conditions:
   - **Unemployment trend**: Is the current unemployment rate above its 12-month moving average? If yes, a recession may be starting.
   - **Price trend**: Is the equity index (SPY) below its 10-month simple moving average?
2. **Both conditions must be true** to go defensive:
   - If unemployment is rising AND price is below SMA: allocate 100% to cash (BIL)
   - Otherwise: allocate 100% to equities (SPY)
3. Hold until the close of the following month.

The strategy signals a bear market less than 15% of the time but captures most major downturns.

## Parameters

- **Equity Ticker**: The equity asset to hold (default: SPY)
- **Cash Ticker**: The cash/bond asset for defensive allocation (default: BIL)
- **Unemployment Ticker**: The unemployment rate data series (default: UNRATE)
- **Price SMA Length**: Moving average period in months for the price trend (default: 10)

## References

- Philosophical Economics (2016). "In Search of the Perfect Recession Indicator."
- Keller, W.J. (2019). "Growth-Trend Timing and 60-40 Variations: Lethargic Asset Allocation (LAA)." SSRN 3498092.
