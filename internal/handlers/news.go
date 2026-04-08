package handlers

import (
	"fmt"
	"net/http"
)

func (h *Handler) ListNews(w http.ResponseWriter, r *http.Request) {
	if h.SQLite == nil {
		writeJSON(w, map[string]any{"articles": []any{}, "total": 0})
		return
	}

	classification := queryStr(r, "classification")
	eventType := queryStr(r, "type")
	limit := queryInt(r, "limit", 20)
	offset := queryInt(r, "offset", 0)

	where := "1=1"
	args := []any{}
	if classification != "" {
		where += " AND classification = ?"
		args = append(args, classification)
	}
	if eventType != "" {
		where += " AND event_type = ?"
		args = append(args, eventType)
	}

	var total int
	countQuery := fmt.Sprintf("SELECT count(*) FROM articles WHERE %s", where)
	h.SQLite.DB.QueryRow(countQuery, args...).Scan(&total)

	query := fmt.Sprintf("SELECT hash, title, link, source, classification, event_type, processed_at, notified FROM articles WHERE %s ORDER BY processed_at DESC LIMIT ? OFFSET ?", where)
	args = append(args, limit, offset)

	rows, err := h.SQLite.DB.Query(query, args...)
	if err != nil {
		writeError(w, 500, "internal error")
		return
	}
	defer rows.Close()

	var articles []map[string]any
	for rows.Next() {
		var hash string
		var title, link, source, cls, eventT, processedAt *string
		var notified int
		rows.Scan(&hash, &title, &link, &source, &cls, &eventT, &processedAt, &notified)
		articles = append(articles, map[string]any{
			"hash": hash, "title": deref(title), "link": deref(link), "source": deref(source),
			"classification": deref(cls), "event_type": deref(eventT), "processed_at": deref(processedAt), "notified": notified,
		})
	}
	if articles == nil {
		articles = []map[string]any{}
	}

	writeJSON(w, map[string]any{"articles": articles, "total": total})
}

func (h *Handler) GetNewsStats(w http.ResponseWriter, r *http.Request) {
	if h.SQLite == nil {
		writeJSON(w, map[string]any{"byClassification": map[string]int{}, "byEventType": map[string]int{}})
		return
	}

	byCls := map[string]int{}
	rows, err := h.SQLite.DB.Query("SELECT classification, count(*) FROM articles GROUP BY classification")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var cls string
			var cnt int
			rows.Scan(&cls, &cnt)
			byCls[cls] = cnt
		}
	}

	byType := map[string]int{}
	rows2, err := h.SQLite.DB.Query("SELECT event_type, count(*) FROM articles WHERE event_type IS NOT NULL AND event_type != '' GROUP BY event_type")
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var t string
			var cnt int
			rows2.Scan(&t, &cnt)
			byType[t] = cnt
		}
	}

	writeJSON(w, map[string]any{"byClassification": byCls, "byEventType": byType})
}

func (h *Handler) ListScans(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		writeError(w, 403, "admin access required")
		return
	}
	if h.SQLite == nil {
		writeJSON(w, []any{})
		return
	}

	limit := queryInt(r, "limit", 10)
	rows, err := h.SQLite.DB.Query("SELECT id, scanned_at, total_fetched, new_articles, high_relevance, notifications_sent FROM scan_log ORDER BY scanned_at DESC LIMIT ?", limit)
	if err != nil {
		writeJSON(w, []any{})
		return
	}
	defer rows.Close()

	var scans []map[string]any
	for rows.Next() {
		var id, totalFetched, newArticles, highRelevance, notifSent int
		var scannedAt string
		rows.Scan(&id, &scannedAt, &totalFetched, &newArticles, &highRelevance, &notifSent)
		scans = append(scans, map[string]any{
			"id": id, "scanned_at": scannedAt, "total_fetched": totalFetched,
			"new_articles": newArticles, "high_relevance": highRelevance, "notifications_sent": notifSent,
		})
	}
	if scans == nil {
		scans = []map[string]any{}
	}

	writeJSON(w, scans)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
