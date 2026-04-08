package handlers

import "net/http"

func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		writeError(w, 403, "admin access required")
		return
	}
	neo4jStats, err := h.Neo4j.Query(`
		MATCH (c:Company) WITH count(c) as companies
		MATCH (p:Plant) WITH companies, count(p) as plants
		MATCH (s:Sector) WITH companies, plants, count(s) as sectors
		MATCH (l:Location) WITH companies, plants, sectors, count(l) as locations
		MATCH (rm:RawMaterial) WITH companies, plants, sectors, locations, count(rm) as materials
		MATCH ()-[r1:COMPETES_WITH]->() WITH companies, plants, sectors, locations, materials, count(r1) as competes
		MATCH ()-[r2:SUPPLIES_TO]->() WITH companies, plants, sectors, locations, materials, competes, count(r2) as supplies
		MATCH ()-[r3:CONSUMES]->() WITH companies, plants, sectors, locations, materials, competes, supplies, count(r3) as consumes
		RETURN companies, plants, sectors, locations, materials, competes, supplies, consumes`, nil)
	if err != nil {
		writeError(w, 500, "internal error")
		return
	}

	result := map[string]any{
		"neo4j":         map[string]any{},
		"relationships": map[string]any{},
		"news":          map[string]any{},
		"recentScans":   []any{},
	}

	if len(neo4jStats) > 0 {
		s := neo4jStats[0]
		result["neo4j"] = map[string]any{
			"companies": s["companies"],
			"plants":    s["plants"],
			"sectors":   s["sectors"],
			"locations": s["locations"],
			"materials": s["materials"],
		}
		result["relationships"] = map[string]any{
			"competes": s["competes"],
			"supplies": s["supplies"],
			"consumes": s["consumes"],
		}
	}

	if h.SQLite != nil {
		newsStats := map[string]int{"total": 0, "high": 0, "medium": 0, "low": 0}
		rows, err := h.SQLite.DB.Query(`SELECT classification, count(*) FROM articles GROUP BY classification`)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var cls string
				var cnt int
				rows.Scan(&cls, &cnt)
				newsStats["total"] += cnt
				switch cls {
				case "HIGH":
					newsStats["high"] = cnt
				case "MEDIUM":
					newsStats["medium"] = cnt
				case "LOW":
					newsStats["low"] = cnt
				}
			}
		}
		result["news"] = newsStats

		var scans []map[string]any
		scanRows, err := h.SQLite.DB.Query(`SELECT id, scanned_at, total_fetched, new_articles, high_relevance, notifications_sent FROM scan_log ORDER BY scanned_at DESC LIMIT 5`)
		if err == nil {
			defer scanRows.Close()
			for scanRows.Next() {
				var id, totalFetched, newArticles, highRelevance, notificationsSent int
				var scannedAt string
				scanRows.Scan(&id, &scannedAt, &totalFetched, &newArticles, &highRelevance, &notificationsSent)
				scans = append(scans, map[string]any{
					"id": id, "scanned_at": scannedAt, "total_fetched": totalFetched,
					"new_articles": newArticles, "high_relevance": highRelevance, "notifications_sent": notificationsSent,
				})
			}
		}
		if scans == nil {
			scans = []map[string]any{}
		}
		result["recentScans"] = scans
	}

	writeJSON(w, result)
}
