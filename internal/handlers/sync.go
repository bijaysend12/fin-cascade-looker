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

	// Commodities (always full set, small: 12 rows)
	result["commodities"] = h.syncCommodities()

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
			ca.supply_chain, ca.sector_ripple, ca.timeline, ca.historical_pattern
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
		var directImpact, beneficiaries, demandFlow, supplyChain, sectorRipple, timeline, historicalPattern *string
		rows.Scan(&eventID, &directImpact, &beneficiaries, &demandFlow, &supplyChain, &sectorRipple, &timeline, &historicalPattern)
		cascades = append(cascades, map[string]any{
			"event_id":           eventID,
			"direct_impact":      rawJSON(directImpact),
			"beneficiaries":      rawJSON(beneficiaries),
			"demand_flow":        rawJSON(demandFlow),
			"supply_chain":       rawJSON(supplyChain),
			"sector_ripple":      rawJSON(sectorRipple),
			"timeline":           rawJSON(timeline),
			"historical_pattern": rawJSON(historicalPattern),
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

// rawJSON returns a json.RawMessage so JSONB columns stay as raw JSON in the response
// instead of being double-encoded as strings.
func rawJSON(s *string) any {
	if s == nil || *s == "" {
		return nil
	}
	return json.RawMessage(*s)
}

