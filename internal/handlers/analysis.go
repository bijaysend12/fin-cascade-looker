package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
)

func (h *Handler) ListAnalysisScans(w http.ResponseWriter, r *http.Request) {
	if h.PG == nil {
		writeJSON(w, map[string]any{"scans": []any{}, "total": 0})
		return
	}

	limit := queryInt(r, "limit", 20)
	offset := queryInt(r, "offset", 0)

	var total int
	h.PG.DB.QueryRow("SELECT count(*) FROM scans").Scan(&total)

	rows, err := h.PG.DB.Query(`
		SELECT id, ran_at, articles_fetched, articles_new, high_count, medium_count, low_count, events_analyzed, email_sent
		FROM scans ORDER BY ran_at DESC LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		writeError(w, 500, "internal error")
		return
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
		scans = []map[string]any{}
	}

	writeJSON(w, map[string]any{"scans": scans, "total": total})
}

func (h *Handler) GetAnalysisScan(w http.ResponseWriter, r *http.Request) {
	if h.PG == nil {
		writeError(w, 503, "PostgreSQL not available")
		return
	}

	scanID := r.PathValue("id")

	var id, fetched, new_, high, med, low, events int
	var ranAt string
	var emailSent bool
	err := h.PG.DB.QueryRow(`
		SELECT id, ran_at, articles_fetched, articles_new, high_count, medium_count, low_count, events_analyzed, email_sent
		FROM scans WHERE id = $1
	`, scanID).Scan(&id, &ranAt, &fetched, &new_, &high, &med, &low, &events, &emailSent)
	if err != nil {
		writeError(w, 404, "scan not found")
		return
	}

	scan := map[string]any{
		"id": id, "ran_at": ranAt, "articles_fetched": fetched, "articles_new": new_,
		"high_count": high, "medium_count": med, "low_count": low,
		"events_analyzed": events, "email_sent": emailSent,
	}

	articleRows, _ := h.PG.DB.Query(`
		SELECT id, hash, title, source, url, pub_date, classification, event_type, reason
		FROM articles WHERE scan_id = $1 ORDER BY classification, created_at
	`, scanID)
	var articles []map[string]any
	if articleRows != nil {
		defer articleRows.Close()
		for articleRows.Next() {
			var aid int
			var hash, title, classification string
			var source, url, pubDate, eventType, reason *string
			articleRows.Scan(&aid, &hash, &title, &source, &url, &pubDate, &classification, &eventType, &reason)
			articles = append(articles, map[string]any{
				"id": aid, "hash": hash, "title": title, "source": ptrStr(source), "url": ptrStr(url),
				"pub_date": ptrStr(pubDate), "classification": classification,
				"event_type": ptrStr(eventType), "reason": ptrStr(reason),
			})
		}
	}
	if articles == nil {
		articles = []map[string]any{}
	}

	eventRows, _ := h.PG.DB.Query(`
		SELECT id, headline, event_type, subtype, severity, temporal, location, sectors, key_facts
		FROM events WHERE scan_id = $1
		ORDER BY CASE severity WHEN 'CRITICAL' THEN 0 WHEN 'HIGH' THEN 1 WHEN 'MEDIUM' THEN 2 WHEN 'LOW' THEN 3 ELSE 4 END, created_at
	`, scanID)
	var eventsList []map[string]any
	if eventRows != nil {
		defer eventRows.Close()
		for eventRows.Next() {
			var eid int
			var headline, eventType, severity string
			var subtype, temporal *string
			var location, sectors, keyFacts *string
			eventRows.Scan(&eid, &headline, &eventType, &subtype, &severity, &temporal, &location, &sectors, &keyFacts)

			event := map[string]any{
				"id": eid, "headline": headline, "event_type": eventType,
				"subtype": ptrStr(subtype), "severity": severity, "temporal": ptrStr(temporal),
				"location": parseJSON(location), "sectors": parseJSON(sectors), "key_facts": parseJSON(keyFacts),
			}

			sigRows, _ := h.PG.DB.Query(`
				SELECT ticker, signal, direction, impact_range, confidence, fundamentals, reason, reasoning_chain
				FROM signals WHERE event_id = $1 ORDER BY confidence DESC
			`, eid)
			var signals []map[string]any
			if sigRows != nil {
				defer sigRows.Close()
				for sigRows.Next() {
					var ticker, signal, direction string
					var impactRange, reason *string
					var confidence *int
					var fundamentals, reasoningChain *string
					sigRows.Scan(&ticker, &signal, &direction, &impactRange, &confidence, &fundamentals, &reason, &reasoningChain)
					signals = append(signals, map[string]any{
						"ticker": ticker, "signal": signal, "direction": direction,
						"impact_range": ptrStr(impactRange), "confidence": ptrInt(confidence),
						"fundamentals": parseJSON(fundamentals), "reason": ptrStr(reason),
						"reasoning_chain": parseJSON(reasoningChain),
					})
				}
			}
			if signals == nil {
				signals = []map[string]any{}
			}
			event["signals"] = signals

			var analysis map[string]any
			var directImpact, beneficiaries, demandFlow, supplyChain, sectorRipple, timeline *string
			err := h.PG.DB.QueryRow(`
				SELECT direct_impact, beneficiaries, demand_flow, supply_chain, sector_ripple, timeline
				FROM cascade_analysis WHERE event_id = $1
			`, eid).Scan(&directImpact, &beneficiaries, &demandFlow, &supplyChain, &sectorRipple, &timeline)
			if err == nil {
				analysis = map[string]any{
					"direct_impact": parseJSON(directImpact), "beneficiaries": parseJSON(beneficiaries),
					"demand_flow": parseJSON(demandFlow), "supply_chain": parseJSON(supplyChain),
					"sector_ripple": parseJSON(sectorRipple), "timeline": parseJSON(timeline),
				}
			}
			event["analysis"] = analysis

			artRows, _ := h.PG.DB.Query(`
				SELECT a.id, a.hash, a.title, a.source, a.url, a.pub_date, a.classification
				FROM articles a
				JOIN event_articles ea ON a.id = ea.article_id
				WHERE ea.event_id = $1
				ORDER BY a.classification, a.created_at
			`, eid)
			var eventArticles []map[string]any
			if artRows != nil {
				defer artRows.Close()
				for artRows.Next() {
					var aid int
					var hash, title, classification string
					var source, url, pubDate *string
					artRows.Scan(&aid, &hash, &title, &source, &url, &pubDate, &classification)
					eventArticles = append(eventArticles, map[string]any{
						"id": aid, "hash": hash, "title": title, "source": ptrStr(source),
						"url": ptrStr(url), "pub_date": ptrStr(pubDate), "classification": classification,
					})
				}
			}
			if eventArticles == nil {
				eventArticles = []map[string]any{}
			}
			event["articles"] = eventArticles

			eventsList = append(eventsList, event)
		}
	}
	if eventsList == nil {
		eventsList = []map[string]any{}
	}

	writeJSON(w, map[string]any{"scan": scan, "articles": articles, "events": eventsList})
}

func (h *Handler) ListSignals(w http.ResponseWriter, r *http.Request) {
	if h.PG == nil {
		writeJSON(w, map[string]any{"signals": []any{}, "total": 0})
		return
	}

	ticker := queryStr(r, "ticker")
	signal := queryStr(r, "signal")
	limit := queryInt(r, "limit", 50)
	offset := queryInt(r, "offset", 0)

	where := "1=1"
	args := []any{}
	argN := 1

	if ticker != "" {
		where += " AND s.ticker = $" + itoa(argN)
		args = append(args, ticker)
		argN++
	}
	if signal != "" {
		where += " AND s.signal = $" + itoa(argN)
		args = append(args, signal)
		argN++
	}

	var total int
	h.PG.DB.QueryRow("SELECT count(*) FROM signals s JOIN events e ON s.event_id = e.id WHERE "+where, args...).Scan(&total)

	query := `
		SELECT s.ticker, s.signal, s.direction, s.impact_range, s.confidence, s.fundamentals, s.reason, s.reasoning_chain,
			e.headline, e.event_type, e.severity, s.created_at
		FROM signals s JOIN events e ON s.event_id = e.id
		WHERE ` + where + `
		ORDER BY s.created_at DESC LIMIT $` + itoa(argN) + ` OFFSET $` + itoa(argN+1)
	args = append(args, limit, offset)

	rows, err := h.PG.DB.Query(query, args...)
	if err != nil {
		writeError(w, 500, "internal error")
		return
	}
	defer rows.Close()

	var signals []map[string]any
	for rows.Next() {
		var ticker, signal, direction string
		var impactRange, reason, headline, eventType, severity *string
		var confidence *int
		var fundamentals, reasoningChain *string
		var createdAt string
		rows.Scan(&ticker, &signal, &direction, &impactRange, &confidence, &fundamentals, &reason, &reasoningChain,
			&headline, &eventType, &severity, &createdAt)
		signals = append(signals, map[string]any{
			"ticker": ticker, "signal": signal, "direction": direction,
			"impact_range": ptrStr(impactRange), "confidence": ptrInt(confidence),
			"fundamentals": parseJSON(fundamentals), "reason": ptrStr(reason),
			"reasoning_chain": parseJSON(reasoningChain),
			"event_headline": ptrStr(headline), "event_type": ptrStr(eventType),
			"severity": ptrStr(severity), "created_at": createdAt,
		})
	}
	if signals == nil {
		signals = []map[string]any{}
	}

	writeJSON(w, map[string]any{"signals": signals, "total": total})
}

func (h *Handler) GetAnalysisStats(w http.ResponseWriter, r *http.Request) {
	if h.PG == nil {
		writeJSON(w, map[string]any{})
		return
	}

	result := map[string]any{}

	var totalScans, totalEvents, totalSignals int
	h.PG.DB.QueryRow("SELECT count(*) FROM scans").Scan(&totalScans)
	h.PG.DB.QueryRow("SELECT count(*) FROM events").Scan(&totalEvents)
	h.PG.DB.QueryRow("SELECT count(*) FROM signals").Scan(&totalSignals)

	result["total_scans"] = totalScans
	result["total_events"] = totalEvents
	result["total_signals"] = totalSignals

	bySignal := map[string]int{}
	rows, err := h.PG.DB.Query("SELECT signal, count(*) FROM signals GROUP BY signal")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var s string
			var cnt int
			rows.Scan(&s, &cnt)
			bySignal[s] = cnt
		}
	}
	result["by_signal"] = bySignal

	bySeverity := map[string]int{}
	rows2, err := h.PG.DB.Query("SELECT severity, count(*) FROM events GROUP BY severity")
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var s string
			var cnt int
			rows2.Scan(&s, &cnt)
			bySeverity[s] = cnt
		}
	}
	result["by_severity"] = bySeverity

	writeJSON(w, result)
}

func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func ptrInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

func parseJSON(s *string) any {
	if s == nil || *s == "" {
		return nil
	}
	var v any
	if err := json.Unmarshal([]byte(*s), &v); err != nil {
		return *s
	}
	return v
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
