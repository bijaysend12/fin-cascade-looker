package handlers

import "net/http"

func (h *Handler) ListSectors(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		writeError(w, 403, "admin access required")
		return
	}
	records, err := h.Neo4j.Query(
		`MATCH (s:Sector)<-[:BELONGS_TO]-(c:Company)
		RETURN s.name as name, count(c) as companyCount,
			s.median_pe as median_pe, s.median_pb as median_pb,
			s.median_roce as median_roce, s.median_roe as median_roe,
			s.median_de as median_de, s.median_op_margin as median_op_margin,
			s.medians_computed_at as medians_computed_at
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
