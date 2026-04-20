package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

func (h *Handler) Sync(w http.ResponseWriter, r *http.Request) {
	since := queryStr(r, "since")

	var cutoff time.Time
	if since != "" {
		var err error
		cutoff, err = time.Parse(time.RFC3339, since)
		if err != nil {
			writeError(w, 400, "invalid since format, use RFC3339")
			return
		}
	} else {
		cutoff = time.Now().AddDate(-1, 0, 0)
	}

	cutoffStr := cutoff.Format(time.RFC3339)

	// Optional upper bound for date-windowed requests (used by backfill).
	until := queryStr(r, "until")
	var ceilingStr string
	if until != "" {
		ceiling, err := time.Parse(time.RFC3339, until)
		if err != nil {
			writeError(w, 400, "invalid until format, use RFC3339")
			return
		}
		ceilingStr = ceiling.Format(time.RFC3339)
	}

	result := map[string]any{
		"sync_version": time.Now().UTC().Format(time.RFC3339),
	}

	// Scans
	result["scans"] = h.syncScans(cutoffStr, ceilingStr)

	// Events
	result["events"] = h.syncEvents(cutoffStr, ceilingStr)

	// Signals (denormalized with event context)
	result["signals"] = h.syncSignals(cutoffStr, ceilingStr)

	// Articles
	result["articles"] = h.syncArticles(cutoffStr, ceilingStr)

	// Event-Article junction
	result["event_articles"] = h.syncEventArticles(cutoffStr, ceilingStr)

	// Cascade analyses
	result["cascade_analyses"] = h.syncCascadeAnalyses(cutoffStr, ceilingStr)

	// News articles (from SQLite)
	result["news_articles"] = h.syncNewsArticles(cutoffStr, ceilingStr)

	// Renko signals (always latest per ticker)
	result["renko_signals"] = h.syncRenkoSignals()

	// Renko reversals (last 30 days, both directions)
	result["renko_reversals"] = h.syncRenkoReversals(cutoffStr, ceilingStr)

	// Renko reversals stats (direction-split hit rates + latest run date)
	result["renko_reversals_stats"] = h.syncRenkoReversalsStats()

	// Commodities (always full set, small: 12 rows)
	result["commodities"] = h.syncCommodities()

	// Commodity price history (last 90 days daily closes — ~1k rows)
	result["commodity_price_history"] = h.syncCommodityPriceHistory()

	writeJSON(w, result)
}

func (h *Handler) syncCommodities() []map[string]any {
	if h.Neo4j == nil {
		return []map[string]any{}
	}

	records, err := h.Neo4j.Query(
		`MATCH (c:Commodity)
		RETURN c.name AS name, c.type AS type, c.unit AS unit, c.yahoo_symbol AS yahoo_symbol,
			c.current_price AS current_price, c.prev_month_price AS prev_month_price,
			c.prev_quarter_price AS prev_quarter_price,
			c.change_1m_pct AS change_1m_pct, c.change_3m_pct AS change_3m_pct,
			c.price_updated_at AS price_updated_at
		ORDER BY c.name`, nil)
	if err != nil || records == nil {
		return []map[string]any{}
	}
	return records
}

func (h *Handler) syncCommodityPriceHistory() []map[string]any {
	if h.PG == nil {
		return []map[string]any{}
	}

	rows, err := h.PG.DB.Query(`
		SELECT name, trade_date::text, close
		FROM commodity_prices
		WHERE trade_date >= (CURRENT_DATE - INTERVAL '90 days')
		ORDER BY name, trade_date`)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()

	var out []map[string]any
	for rows.Next() {
		var name, tradeDate string
		var closePrice float64
		if err := rows.Scan(&name, &tradeDate, &closePrice); err != nil {
			continue
		}
		out = append(out, map[string]any{
			"name":       name,
			"trade_date": tradeDate,
			"close":      closePrice,
		})
	}
	if out == nil {
		return []map[string]any{}
	}
	return out
}

func (h *Handler) syncScans(since, until string) []map[string]any {
	if h.PG == nil {
		return []map[string]any{}
	}

	q := `SELECT id, ran_at, articles_fetched, articles_new, high_count, medium_count, low_count, events_analyzed, email_sent
		FROM scans WHERE ran_at >= $1`
	args := []any{since}
	if until != "" {
		q += ` AND ran_at < $2`
		args = append(args, until)
	}
	q += ` ORDER BY ran_at DESC`
	rows, err := h.PG.DB.Query(q, args...)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()

	var scans []map[string]any
	for rows.Next() {
		var id, fetched, new_, high, med, low, events int
		var ranAt string
		var emailSent bool
		rows.Scan(&id, &ranAt, &fetched, &new_, &high, &med, &low, &events, &emailSent)
		scans = append(scans, map[string]any{
			"id": id, "ran_at": ranAt, "articles_fetched": fetched, "articles_new": new_,
			"high_count": high, "medium_count": med, "low_count": low,
			"events_analyzed": events, "email_sent": emailSent,
		})
	}
	if scans == nil {
		return []map[string]any{}
	}
	return scans
}

func (h *Handler) syncEvents(since, until string) []map[string]any {
	if h.PG == nil {
		return []map[string]any{}
	}

	q := `SELECT id, scan_id, headline, event_type, subtype, severity, temporal, location, sectors, key_facts, created_at
		FROM events WHERE created_at >= $1`
	args := []any{since}
	if until != "" {
		q += ` AND created_at < $2`
		args = append(args, until)
	}
	q += ` ORDER BY created_at DESC`
	rows, err := h.PG.DB.Query(q, args...)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()

	var events []map[string]any
	for rows.Next() {
		var id, scanID int
		var headline, eventType, severity, createdAt string
		var subtype, temporal, location, sectors, keyFacts *string
		rows.Scan(&id, &scanID, &headline, &eventType, &subtype, &severity, &temporal, &location, &sectors, &keyFacts, &createdAt)
		events = append(events, map[string]any{
			"id": id, "scan_id": scanID, "headline": headline, "event_type": eventType,
			"subtype": ptrStr(subtype), "severity": severity, "temporal": ptrStr(temporal),
			"location": rawJSON(location), "sectors": rawJSON(sectors), "key_facts": rawJSON(keyFacts),
			"created_at": createdAt,
		})
	}
	if events == nil {
		return []map[string]any{}
	}
	return events
}

func (h *Handler) syncSignals(since, until string) []map[string]any {
	if h.PG == nil {
		return []map[string]any{}
	}

	q := `SELECT s.id, s.event_id, s.ticker, s.signal, s.direction, s.impact_range, s.confidence,
			s.fundamentals, s.reason, s.reasoning_chain, s.created_at,
			e.headline, e.event_type, e.severity
		FROM signals s JOIN events e ON s.event_id = e.id
		WHERE s.created_at >= $1`
	args := []any{since}
	if until != "" {
		q += ` AND s.created_at < $2`
		args = append(args, until)
	}
	q += ` ORDER BY s.created_at DESC`
	rows, err := h.PG.DB.Query(q, args...)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()

	var signals []map[string]any
	for rows.Next() {
		var id, eventID int
		var ticker, signal, direction, createdAt string
		var impactRange, reason *string
		var confidence *int
		var fundamentals, reasoningChain *string
		var headline, eventType, severity *string
		rows.Scan(&id, &eventID, &ticker, &signal, &direction, &impactRange, &confidence,
			&fundamentals, &reason, &reasoningChain, &createdAt,
			&headline, &eventType, &severity)
		signals = append(signals, map[string]any{
			"id": id, "event_id": eventID, "ticker": ticker, "signal": signal, "direction": direction,
			"impact_range": ptrStr(impactRange), "confidence": ptrInt(confidence),
			"fundamentals": rawJSON(fundamentals), "reason": ptrStr(reason),
			"reasoning_chain": rawJSON(reasoningChain), "created_at": createdAt,
			"event_headline": ptrStr(headline), "event_type": ptrStr(eventType), "severity": ptrStr(severity),
		})
	}
	if signals == nil {
		return []map[string]any{}
	}
	return signals
}

func (h *Handler) syncArticles(since, until string) []map[string]any {
	if h.PG == nil {
		return []map[string]any{}
	}

	q := `SELECT id, scan_id, hash, title, source, url, pub_date, classification, event_type, reason, created_at
		FROM articles WHERE created_at >= $1`
	args := []any{since}
	if until != "" {
		q += ` AND created_at < $2`
		args = append(args, until)
	}
	q += ` ORDER BY created_at DESC`
	rows, err := h.PG.DB.Query(q, args...)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()

	var articles []map[string]any
	for rows.Next() {
		var id, scanID int
		var hash, title, classification, createdAt string
		var source, url, pubDate, eventType, reason *string
		rows.Scan(&id, &scanID, &hash, &title, &source, &url, &pubDate, &classification, &eventType, &reason, &createdAt)
		articles = append(articles, map[string]any{
			"id": id, "scan_id": scanID, "hash": hash, "title": title, "source": ptrStr(source),
			"url": ptrStr(url), "pub_date": ptrStr(pubDate), "classification": classification,
			"event_type": ptrStr(eventType), "reason": ptrStr(reason), "created_at": createdAt,
		})
	}
	if articles == nil {
		return []map[string]any{}
	}
	return articles
}

func (h *Handler) syncEventArticles(since, until string) []map[string]any {
	if h.PG == nil {
		return []map[string]any{}
	}

	q := `SELECT ea.event_id, ea.article_id
		FROM event_articles ea
		JOIN events e ON ea.event_id = e.id
		WHERE e.created_at >= $1`
	args := []any{since}
	if until != "" {
		q += ` AND e.created_at < $2`
		args = append(args, until)
	}
	rows, err := h.PG.DB.Query(q, args...)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()

	var junctions []map[string]any
	for rows.Next() {
		var eventID, articleID int
		rows.Scan(&eventID, &articleID)
		junctions = append(junctions, map[string]any{
			"event_id": eventID, "article_id": articleID,
		})
	}
	if junctions == nil {
		return []map[string]any{}
	}
	return junctions
}

func (h *Handler) syncCascadeAnalyses(since, until string) []map[string]any {
	if h.PG == nil {
		return []map[string]any{}
	}

	q := `SELECT ca.event_id, ca.direct_impact, ca.beneficiaries, ca.demand_flow,
			ca.supply_chain, ca.sector_ripple, ca.timeline, ca.historical_pattern,
			ca.commodity_context
		FROM cascade_analysis ca
		JOIN events e ON ca.event_id = e.id
		WHERE e.created_at >= $1`
	args := []any{since}
	if until != "" {
		q += ` AND e.created_at < $2`
		args = append(args, until)
	}
	rows, err := h.PG.DB.Query(q, args...)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()

	var cascades []map[string]any
	for rows.Next() {
		var eventID int
		var directImpact, beneficiaries, demandFlow, supplyChain, sectorRipple, timeline, historicalPattern, commodityContext *string
		rows.Scan(&eventID, &directImpact, &beneficiaries, &demandFlow, &supplyChain, &sectorRipple, &timeline, &historicalPattern, &commodityContext)
		cascades = append(cascades, map[string]any{
			"event_id":           eventID,
			"direct_impact":      rawJSON(directImpact),
			"beneficiaries":      rawJSON(beneficiaries),
			"demand_flow":        rawJSON(demandFlow),
			"supply_chain":       rawJSON(supplyChain),
			"sector_ripple":      rawJSON(sectorRipple),
			"timeline":           rawJSON(timeline),
			"historical_pattern": rawJSON(historicalPattern),
			"commodity_context":  rawJSON(commodityContext),
		})
	}
	if cascades == nil {
		return []map[string]any{}
	}
	return cascades
}

func (h *Handler) syncNewsArticles(since, until string) []map[string]any {
	if h.SQLite == nil {
		return []map[string]any{}
	}

	// SQLite processed_at uses space separator ("2026-04-13 22:41:14"),
	// but since/until params are RFC3339 with T separator. Normalize for correct comparison.
	sqliteSince := strings.Replace(since, "T", " ", 1)
	if idx := strings.Index(sqliteSince, "Z"); idx != -1 {
		sqliteSince = sqliteSince[:idx]
	}
	q := `SELECT hash, title, link, source, classification, event_type, processed_at, notified
		FROM articles WHERE processed_at >= ?`
	args := []any{sqliteSince}
	if until != "" {
		sqliteUntil := strings.Replace(until, "T", " ", 1)
		if idx := strings.Index(sqliteUntil, "Z"); idx != -1 {
			sqliteUntil = sqliteUntil[:idx]
		}
		q += ` AND processed_at < ?`
		args = append(args, sqliteUntil)
	}
	q += ` ORDER BY processed_at DESC`
	rows, err := h.SQLite.DB.Query(q, args...)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()

	var articles []map[string]any
	for rows.Next() {
		var hash string
		var title, link, source, cls, eventT, processedAt *string
		var notified int
		rows.Scan(&hash, &title, &link, &source, &cls, &eventT, &processedAt, &notified)
		// Append Z to processed_at so clients parse it as UTC (SQLite stores without timezone)
		pa := deref(processedAt)
		if pa != "" && !strings.HasSuffix(pa, "Z") && !strings.Contains(pa, "+") {
			pa = strings.Replace(pa, " ", "T", 1) + "Z"
		}
		articles = append(articles, map[string]any{
			"hash": hash, "title": deref(title), "url": deref(link), "source": deref(source),
			"classification": deref(cls), "event_type": deref(eventT),
			"processed_at": pa, "notified": notified,
		})
	}
	if articles == nil {
		return []map[string]any{}
	}
	return articles
}

func (h *Handler) syncRenkoSignals() []map[string]any {
	if h.PG == nil {
		return []map[string]any{}
	}

	rows, err := h.PG.DB.Query(`
		WITH latest AS (
			SELECT DISTINCT ON (ticker) ticker, trade_date, brick_size, brick_count, direction, trend, consecutive, anchor_price, atr_14
			FROM renko_signals
			ORDER BY ticker, trade_date DESC
		)
		SELECT ticker, trade_date, brick_size, brick_count, direction, trend, consecutive, anchor_price, atr_14
		FROM latest ORDER BY trend, ticker
	`)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()

	var signals []map[string]any
	for rows.Next() {
		var ticker, tradeDate, direction, trend string
		var brickSize, anchorPrice float64
		var brickCount, consecutive int
		var atr *float64
		rows.Scan(&ticker, &tradeDate, &brickSize, &brickCount, &direction, &trend, &consecutive, &anchorPrice, &atr)
		entry := map[string]any{
			"ticker": ticker, "trade_date": tradeDate, "brick_size": brickSize,
			"brick_count": brickCount, "direction": direction, "trend": trend,
			"consecutive": consecutive, "anchor_price": anchorPrice,
		}
		if atr != nil {
			entry["atr_14"] = *atr
		}
		signals = append(signals, entry)
	}
	if signals == nil {
		return []map[string]any{}
	}
	return signals
}

func (h *Handler) syncRenkoReversals(since, until string) []map[string]any {
	if h.PG == nil {
		return []map[string]any{}
	}

	q := `SELECT ticker, reversal_date, run_date, direction,
			streak_length, streak_duration_days, brick_size, atr_14, atr_trend,
			move_pct_from_extreme, current_price, anchor_price,
			rsi_14, macd_cross_direction, ma_50d_slope, price_vs_ma_200d_pct, candlestick_pattern,
			volume_ratio_reversal_day, volume_ratio_streak_window, peer_reversals_same_day,
			promoter_sell_value_cr_90d, promoter_buy_value_cr_90d, insider_sell_buy_ratio_90d,
			bulk_deals_sell_qty_90d, bulk_deals_buy_qty_90d, institutional_flag,
			opus_classification, opus_score, opus_reasoning,
			codex_classification, codex_score, codex_reasoning,
			debate_outcome, composite_score, debate_summary,
			roce, debt_to_equity, promoter_pledge_pct, pe, market_cap, sector, industry,
			piotroski_score, altman_z_score, interest_coverage_ratio, current_ratio,
			recent_cascade_sentiment, recent_cascade_count, sector_move_pct,
			outcome, user_thesis, user_risk, verdict_label, created_at
		FROM renko_reversals
		WHERE reversal_date >= (NOW() - INTERVAL '30 days')::date
			AND created_at >= $1`
	args := []any{since}
	if until != "" {
		q += ` AND created_at < $2`
		args = append(args, until)
	}
	q += ` ORDER BY reversal_date DESC, run_date DESC, ticker`
	rows, err := h.PG.DB.Query(q, args...)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()

	var out []map[string]any
	for rows.Next() {
		var ticker, direction, opusClass, opusReason, createdAt string
		var reversalDate, runDate string
		var streakLength, streakDurationDays int
		var brickSize, atr14, movePct, currentPrice, anchorPrice *float64
		var atrTrend, macdCross, candlestick *string
		var rsi14, ma50Slope, priceVs200 *float64
		var volRatioDay, volRatioStreak *float64
		var peerReversals *int
		var promoterSell, promoterBuy, insiderRatio *float64
		var bulkSell, bulkBuy *int64
		var instFlag *string
		var opusScore float64
		var codexClass, codexReason, debateOutcome, debateSummary *string
		var codexScore *float64
		var compositeScore float64
		var roce, debtEq, pledge, pe, marketCap, sectorMove *float64
		var sector, industry *string
		var piotroski *int
		var altmanZ, intCov, currentRatio *float64
		var cascadeSentiment *string
		var cascadeCount *int
		var outcome, userThesis, userRisk, verdictLabel *string
		if err := rows.Scan(&ticker, &reversalDate, &runDate, &direction,
			&streakLength, &streakDurationDays, &brickSize, &atr14, &atrTrend,
			&movePct, &currentPrice, &anchorPrice,
			&rsi14, &macdCross, &ma50Slope, &priceVs200, &candlestick,
			&volRatioDay, &volRatioStreak, &peerReversals,
			&promoterSell, &promoterBuy, &insiderRatio,
			&bulkSell, &bulkBuy, &instFlag,
			&opusClass, &opusScore, &opusReason,
			&codexClass, &codexScore, &codexReason,
			&debateOutcome, &compositeScore, &debateSummary,
			&roce, &debtEq, &pledge, &pe, &marketCap, &sector, &industry,
			&piotroski, &altmanZ, &intCov, &currentRatio,
			&cascadeSentiment, &cascadeCount, &sectorMove,
			&outcome, &userThesis, &userRisk, &verdictLabel, &createdAt); err != nil {
			continue
		}
		entry := map[string]any{
			"ticker":                     ticker,
			"reversal_date":              reversalDate,
			"run_date":                   runDate,
			"direction":                  direction,
			"streak_length":              streakLength,
			"streak_duration_days":       streakDurationDays,
			"opus_classification":        opusClass,
			"opus_score":                 opusScore,
			"opus_reasoning":             opusReason,
			"composite_score":            compositeScore,
			"created_at":                 createdAt,
		}
		putFloat(entry, "brick_size", brickSize)
		putFloat(entry, "atr_14", atr14)
		putStr(entry, "atr_trend", atrTrend)
		putFloat(entry, "move_pct_from_extreme", movePct)
		putFloat(entry, "current_price", currentPrice)
		putFloat(entry, "anchor_price", anchorPrice)
		putFloat(entry, "rsi_14", rsi14)
		putStr(entry, "macd_cross_direction", macdCross)
		putFloat(entry, "ma_50d_slope", ma50Slope)
		putFloat(entry, "price_vs_ma_200d_pct", priceVs200)
		putStr(entry, "candlestick_pattern", candlestick)
		putFloat(entry, "volume_ratio_reversal_day", volRatioDay)
		putFloat(entry, "volume_ratio_streak_window", volRatioStreak)
		putInt(entry, "peer_reversals_same_day", peerReversals)
		putFloat(entry, "promoter_sell_value_cr_90d", promoterSell)
		putFloat(entry, "promoter_buy_value_cr_90d", promoterBuy)
		putFloat(entry, "insider_sell_buy_ratio_90d", insiderRatio)
		putInt64(entry, "bulk_deals_sell_qty_90d", bulkSell)
		putInt64(entry, "bulk_deals_buy_qty_90d", bulkBuy)
		putStr(entry, "institutional_flag", instFlag)
		putStr(entry, "codex_classification", codexClass)
		putFloat(entry, "codex_score", codexScore)
		putStr(entry, "codex_reasoning", codexReason)
		putStr(entry, "debate_outcome", debateOutcome)
		putStr(entry, "debate_summary", debateSummary)
		putFloat(entry, "roce", roce)
		putFloat(entry, "debt_to_equity", debtEq)
		putFloat(entry, "promoter_pledge_pct", pledge)
		putFloat(entry, "pe", pe)
		putFloat(entry, "market_cap", marketCap)
		putStr(entry, "sector", sector)
		putStr(entry, "industry", industry)
		putInt(entry, "piotroski_score", piotroski)
		putFloat(entry, "altman_z_score", altmanZ)
		putFloat(entry, "interest_coverage_ratio", intCov)
		putFloat(entry, "current_ratio", currentRatio)
		putStr(entry, "recent_cascade_sentiment", cascadeSentiment)
		putInt(entry, "recent_cascade_count", cascadeCount)
		putFloat(entry, "sector_move_pct", sectorMove)
		putStr(entry, "outcome", outcome)
		putStr(entry, "user_thesis", userThesis)
		putStr(entry, "user_risk", userRisk)
		putStr(entry, "verdict_label", verdictLabel)
		out = append(out, entry)
	}
	if out == nil {
		return []map[string]any{}
	}
	return out
}

func (h *Handler) syncRenkoReversalsStats() map[string]any {
	stats := map[string]any{
		"buy":             emptyDirectionStats(),
		"sell":            emptyDirectionStats(),
		"latest_run_date": nil,
	}
	if h.PG == nil {
		return stats
	}

	rows, err := h.PG.DB.Query(`
		SELECT direction,
			COUNT(*) FILTER (WHERE reversal_date >= (NOW() - INTERVAL '30 days')::date) AS cnt,
			COUNT(*) FILTER (
				WHERE reversal_date >= (NOW() - INTERVAL '30 days')::date
				  AND outcome IN ('green','correct')
			) AS correct
		FROM renko_reversals
		WHERE direction IN ('up','down')
		GROUP BY direction
	`)
	if err != nil {
		return stats
	}
	defer rows.Close()
	for rows.Next() {
		var dir string
		var cnt, correct int
		if err := rows.Scan(&dir, &cnt, &correct); err != nil {
			continue
		}
		block := map[string]any{
			"last_30d_count":   cnt,
			"last_30d_correct": correct,
		}
		if cnt > 0 {
			block["last_30d_hit_rate"] = float64(correct) / float64(cnt)
		} else {
			block["last_30d_hit_rate"] = nil
		}
		switch dir {
		case "up":
			stats["buy"] = block
		case "down":
			stats["sell"] = block
		}
	}

	var latest *string
	if err := h.PG.DB.QueryRow(`SELECT MAX(run_date)::text FROM renko_reversals`).Scan(&latest); err == nil && latest != nil {
		stats["latest_run_date"] = *latest
	}
	return stats
}

func emptyDirectionStats() map[string]any {
	return map[string]any{
		"last_30d_count":    0,
		"last_30d_correct":  0,
		"last_30d_hit_rate": nil,
	}
}

func putFloat(m map[string]any, k string, v *float64) {
	if v != nil {
		m[k] = *v
	}
}

func putInt(m map[string]any, k string, v *int) {
	if v != nil {
		m[k] = *v
	}
}

func putInt64(m map[string]any, k string, v *int64) {
	if v != nil {
		m[k] = *v
	}
}

func putStr(m map[string]any, k string, v *string) {
	if v != nil && *v != "" {
		m[k] = *v
	}
}

// rawJSON returns a json.RawMessage so JSONB columns stay as raw JSON in the response
// instead of being double-encoded as strings.
func rawJSON(s *string) any {
	if s == nil || *s == "" {
		return nil
	}
	return json.RawMessage(*s)
}

