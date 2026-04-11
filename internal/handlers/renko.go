package handlers

import (
	"database/sql"
	"net/http"
)

func (h *Handler) GetRenko(w http.ResponseWriter, r *http.Request) {
	if h.PG == nil {
		writeError(w, 503, "PostgreSQL not available")
		return
	}

	ticker := r.PathValue("ticker")
	days := queryInt(r, "days", 60)

	// Fetch daily prices
	priceRows, err := h.PG.DB.Query(`
		SELECT trade_date, open, high, low, close, volume, atr_14
		FROM daily_prices
		WHERE ticker = $1
		ORDER BY trade_date DESC
		LIMIT $2
	`, ticker, days)
	if err != nil {
		writeError(w, 500, "internal error")
		return
	}
	defer priceRows.Close()

	var prices []map[string]any
	for priceRows.Next() {
		var tradeDate string
		var open, high, low, close_ float64
		var volume int64
		var atr sql.NullFloat64
		priceRows.Scan(&tradeDate, &open, &high, &low, &close_, &volume, &atr)
		p := map[string]any{
			"trade_date": tradeDate, "open": open, "high": high,
			"low": low, "close": close_, "volume": volume,
		}
		if atr.Valid {
			p["atr_14"] = atr.Float64
		}
		prices = append(prices, p)
	}
	if prices == nil {
		prices = []map[string]any{}
	}

	// Fetch renko signals
	renkoRows, err := h.PG.DB.Query(`
		SELECT trade_date, brick_size, brick_count, direction, trend, consecutive, anchor_price, atr_14
		FROM renko_signals
		WHERE ticker = $1
		ORDER BY trade_date DESC
		LIMIT $2
	`, ticker, days)
	if err != nil {
		writeError(w, 500, "internal error")
		return
	}
	defer renkoRows.Close()

	var renko []map[string]any
	for renkoRows.Next() {
		var tradeDate, direction, trend string
		var brickSize, anchorPrice float64
		var brickCount, consecutive int
		var atr sql.NullFloat64
		renkoRows.Scan(&tradeDate, &brickSize, &brickCount, &direction, &trend, &consecutive, &anchorPrice, &atr)
		entry := map[string]any{
			"trade_date": tradeDate, "brick_size": brickSize, "brick_count": brickCount,
			"direction": direction, "trend": trend, "consecutive": consecutive,
			"anchor_price": anchorPrice,
		}
		if atr.Valid {
			entry["atr_14"] = atr.Float64
		}
		renko = append(renko, entry)
	}
	if renko == nil {
		renko = []map[string]any{}
	}

	writeJSON(w, map[string]any{"ticker": ticker, "prices": prices, "renko": renko})
}

func (h *Handler) ListRenkoSignals(w http.ResponseWriter, r *http.Request) {
	if h.PG == nil {
		writeJSON(w, map[string]any{"signals": []any{}, "total": 0})
		return
	}

	trend := queryStr(r, "trend")
	direction := queryStr(r, "direction")
	limit := queryInt(r, "limit", 50)
	offset := queryInt(r, "offset", 0)

	// Get latest state per ticker first, then filter
	latestCTE := `WITH latest AS (
		SELECT DISTINCT ON (ticker) ticker, trade_date, brick_size, brick_count, direction, trend, consecutive, anchor_price, atr_14
		FROM renko_signals
		ORDER BY ticker, trade_date DESC
	)`

	where := "1=1"
	args := []any{}
	argN := 1

	if trend != "" {
		where += " AND trend = $" + itoa(argN)
		args = append(args, trend)
		argN++
	}
	if direction != "" {
		where += " AND direction = $" + itoa(argN)
		args = append(args, direction)
		argN++
	}

	var total int
	h.PG.DB.QueryRow(latestCTE+`
		SELECT count(*) FROM latest WHERE `+where,
		args...).Scan(&total)

	fullQuery := latestCTE + `
		SELECT ticker, trade_date, brick_size, brick_count, direction, trend, consecutive, anchor_price, atr_14
		FROM latest WHERE ` + where + `
		ORDER BY trend, ticker LIMIT $` + itoa(argN) + ` OFFSET $` + itoa(argN+1)
	args = append(args, limit, offset)

	rows, err := h.PG.DB.Query(fullQuery, args...)
	if err != nil {
		writeError(w, 500, "internal error")
		return
	}
	defer rows.Close()

	var signals []map[string]any
	for rows.Next() {
		var ticker, tradeDate, direction, trend string
		var brickSize, anchorPrice float64
		var brickCount, consecutive int
		var atr sql.NullFloat64
		rows.Scan(&ticker, &tradeDate, &brickSize, &brickCount, &direction, &trend, &consecutive, &anchorPrice, &atr)
		entry := map[string]any{
			"ticker": ticker, "trade_date": tradeDate, "brick_size": brickSize,
			"brick_count": brickCount, "direction": direction, "trend": trend,
			"consecutive": consecutive, "anchor_price": anchorPrice,
		}
		if atr.Valid {
			entry["atr_14"] = atr.Float64
		}
		signals = append(signals, entry)
	}
	if signals == nil {
		signals = []map[string]any{}
	}

	writeJSON(w, map[string]any{"signals": signals, "total": total})
}

func (h *Handler) GetRenkoStats(w http.ResponseWriter, r *http.Request) {
	if h.PG == nil {
		writeJSON(w, map[string]any{})
		return
	}

	var totalTickers int
	h.PG.DB.QueryRow(`SELECT count(DISTINCT ticker) FROM renko_signals`).Scan(&totalTickers)

	byTrend := map[string]int{}
	rows, err := h.PG.DB.Query(`
		SELECT trend, count(*) FROM (
			SELECT DISTINCT ON (ticker) ticker, trend FROM renko_signals ORDER BY ticker, trade_date DESC
		) sub GROUP BY trend
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var trend string
			var cnt int
			rows.Scan(&trend, &cnt)
			byTrend[trend] = cnt
		}
	}

	byDirection := map[string]int{}
	rows2, err := h.PG.DB.Query(`
		SELECT direction, count(*) FROM (
			SELECT DISTINCT ON (ticker) ticker, direction FROM renko_signals ORDER BY ticker, trade_date DESC
		) sub GROUP BY direction
	`)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var dir string
			var cnt int
			rows2.Scan(&dir, &cnt)
			byDirection[dir] = cnt
		}
	}

	writeJSON(w, map[string]any{
		"total_tickers": totalTickers,
		"by_trend":      byTrend,
		"by_direction":  byDirection,
	})
}
