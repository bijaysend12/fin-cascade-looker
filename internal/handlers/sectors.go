package handlers

import "net/http"

func (h *Handler) ListSectors(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		writeError(w, 403, "admin access required")
		return
	}
	records, err := h.Neo4j.Query(
		`MATCH (s:Sector)<-[:BELONGS_TO]-(c:Company)
		RETURN s.name as name, count(c) as companyCount
		ORDER BY companyCount DESC`, nil)
	if err != nil {
		writeError(w, 500, "internal error")
		return
	}
	if records == nil {
		records = []map[string]any{}
	}
	writeJSON(w, records)
}
