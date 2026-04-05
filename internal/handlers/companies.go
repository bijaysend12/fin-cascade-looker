package handlers

import (
	"fmt"
	"net/http"
)

func (h *Handler) ListCompanies(w http.ResponseWriter, r *http.Request) {
	search := queryStr(r, "search")
	sector := queryStr(r, "sector")
	cap := queryStr(r, "cap")
	limit := queryInt(r, "limit", 20)
	offset := queryInt(r, "offset", 0)

	countQuery := `
		MATCH (c:Company)
		WHERE ($search = '' OR toLower(c.name) CONTAINS toLower($search) OR toLower(c.ticker) CONTAINS toLower($search))
		AND ($sector = '' OR c.sector = $sector)
		AND ($cap = '' OR c.marketCapCategory = $cap)
		RETURN count(c) as total`

	params := map[string]any{"search": search, "sector": sector, "cap": cap, "limit": limit, "offset": offset}

	countResult, err := h.Neo4j.Query(countQuery, params)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	total := 0
	if len(countResult) > 0 {
		if v, ok := countResult[0]["total"].(int64); ok {
			total = int(v)
		}
	}

	listQuery := `
		MATCH (c:Company)
		WHERE ($search = '' OR toLower(c.name) CONTAINS toLower($search) OR toLower(c.ticker) CONTAINS toLower($search))
		AND ($sector = '' OR c.sector = $sector)
		AND ($cap = '' OR c.marketCapCategory = $cap)
		RETURN c.ticker as ticker, c.name as name, c.sector as sector, c.industry as industry,
			c.marketCapCategory as marketCapCategory, c.debtToEquity as debtToEquity,
			c.hqCity as hqCity, c.hqState as hqState, c.exportPct as exportPct
		ORDER BY c.name SKIP toInteger($offset) LIMIT toInteger($limit)`

	records, err := h.Neo4j.Query(listQuery, params)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	if records == nil {
		records = []map[string]any{}
	}

	writeJSON(w, map[string]any{"companies": records, "total": total})
}

func (h *Handler) GetCompany(w http.ResponseWriter, r *http.Request) {
	ticker := r.PathValue("ticker")
	if ticker == "" {
		writeError(w, 400, "ticker required")
		return
	}

	params := map[string]any{"ticker": ticker}

	companyRows, err := h.Neo4j.Query(
		`MATCH (c:Company {ticker: $ticker}) RETURN c.ticker as ticker, c.name as name, c.sector as sector,
		c.industry as industry, c.hqCity as hqCity, c.hqState as hqState, c.debtToEquity as debtToEquity,
		c.exportPct as exportPct, c.marketCapCategory as marketCapCategory, c.description as description`, params)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	if len(companyRows) == 0 {
		writeError(w, 404, fmt.Sprintf("company %s not found", ticker))
		return
	}

	plants, _ := h.Neo4j.Query(
		`MATCH (c:Company {ticker: $ticker})-[:HAS_PLANT]->(p:Plant)
		RETURN p.name as name, p.city as city, p.state as state, p.type as type, p.capacity as capacity
		ORDER BY p.state, p.city`, params)

	competitors, _ := h.Neo4j.Query(
		`MATCH (c:Company {ticker: $ticker})-[:COMPETES_WITH]->(r:Company)
		RETURN r.ticker as ticker, r.name as name, r.sector as sector, r.marketCapCategory as marketCapCategory`, params)

	suppliers, _ := h.Neo4j.Query(
		`MATCH (sup:Company)-[s:SUPPLIES_TO]->(c:Company {ticker: $ticker})
		RETURN sup.ticker as ticker, sup.name as name, s.material as material`, params)

	customers, _ := h.Neo4j.Query(
		`MATCH (c:Company {ticker: $ticker})-[s:SUPPLIES_TO]->(cust:Company)
		RETURN cust.ticker as ticker, cust.name as name, s.material as material`, params)

	rawMaterials, _ := h.Neo4j.Query(
		`MATCH (c:Company {ticker: $ticker})-[r:CONSUMES]->(rm:RawMaterial)
		RETURN rm.name as name, r.imported as imported, r.sourceCountry as sourceCountry`, params)

	sectorInfo, _ := h.Neo4j.Query(
		`MATCH (c:Company {ticker: $ticker})-[:BELONGS_TO]->(s:Sector)
		RETURN s.name as name, s.upstreamDependencies as upstreamDependencies, s.downstreamSectors as downstreamSectors`, params)

	if plants == nil {
		plants = []map[string]any{}
	}
	if competitors == nil {
		competitors = []map[string]any{}
	}
	if suppliers == nil {
		suppliers = []map[string]any{}
	}
	if customers == nil {
		customers = []map[string]any{}
	}
	if rawMaterials == nil {
		rawMaterials = []map[string]any{}
	}

	result := map[string]any{
		"company":      companyRows[0],
		"plants":       plants,
		"competitors":  competitors,
		"suppliers":    suppliers,
		"customers":    customers,
		"rawMaterials": rawMaterials,
	}
	if len(sectorInfo) > 0 {
		result["sector"] = sectorInfo[0]
	}

	writeJSON(w, result)
}

func (h *Handler) GetCompanyGraph(w http.ResponseWriter, r *http.Request) {
	ticker := r.PathValue("ticker")
	if ticker == "" {
		writeError(w, 400, "ticker required")
		return
	}

	params := map[string]any{"ticker": ticker}

	company, err := h.Neo4j.Query(
		`MATCH (c:Company {ticker: $ticker}) RETURN c.ticker as ticker, c.name as name`, params)
	if err != nil || len(company) == 0 {
		writeError(w, 404, "company not found")
		return
	}

	nodes := []map[string]any{{
		"id": ticker, "label": company[0]["name"], "type": "Company", "isCenter": true,
	}}
	links := []map[string]any{}

	plants, _ := h.Neo4j.Query(
		`MATCH (c:Company {ticker: $ticker})-[:HAS_PLANT]->(p:Plant)
		RETURN p.name as name, p.city as city, p.state as state, p.type as type`, params)
	for _, p := range plants {
		id := fmt.Sprintf("plant_%s", p["name"])
		nodes = append(nodes, map[string]any{"id": id, "label": p["name"], "type": "Plant", "properties": p})
		links = append(links, map[string]any{"source": ticker, "target": id, "type": "HAS_PLANT"})
	}

	competitors, _ := h.Neo4j.Query(
		`MATCH (c:Company {ticker: $ticker})-[:COMPETES_WITH]->(r:Company)
		RETURN r.ticker as ticker, r.name as name`, params)
	for _, c := range competitors {
		id := fmt.Sprintf("%s", c["ticker"])
		nodes = append(nodes, map[string]any{"id": id, "label": c["name"], "type": "Competitor"})
		links = append(links, map[string]any{"source": ticker, "target": id, "type": "COMPETES_WITH"})
	}

	suppliers, _ := h.Neo4j.Query(
		`MATCH (sup:Company)-[s:SUPPLIES_TO]->(c:Company {ticker: $ticker})
		RETURN sup.ticker as ticker, sup.name as name, s.material as material`, params)
	for _, s := range suppliers {
		id := fmt.Sprintf("%s", s["ticker"])
		nodes = append(nodes, map[string]any{"id": id, "label": s["name"], "type": "Supplier", "properties": map[string]any{"material": s["material"]}})
		links = append(links, map[string]any{"source": id, "target": ticker, "type": "SUPPLIES_TO"})
	}

	customers, _ := h.Neo4j.Query(
		`MATCH (c:Company {ticker: $ticker})-[s:SUPPLIES_TO]->(cust:Company)
		RETURN cust.ticker as ticker, cust.name as name, s.material as material`, params)
	for _, c := range customers {
		id := fmt.Sprintf("%s", c["ticker"])
		nodes = append(nodes, map[string]any{"id": id, "label": c["name"], "type": "Customer", "properties": map[string]any{"material": c["material"]}})
		links = append(links, map[string]any{"source": ticker, "target": id, "type": "SUPPLIES_TO"})
	}

	sector, _ := h.Neo4j.Query(
		`MATCH (c:Company {ticker: $ticker})-[:BELONGS_TO]->(s:Sector) RETURN s.name as name`, params)
	for _, s := range sector {
		id := fmt.Sprintf("sector_%s", s["name"])
		nodes = append(nodes, map[string]any{"id": id, "label": s["name"], "type": "Sector"})
		links = append(links, map[string]any{"source": ticker, "target": id, "type": "BELONGS_TO"})
	}

	rawMaterials, _ := h.Neo4j.Query(
		`MATCH (c:Company {ticker: $ticker})-[:CONSUMES]->(rm:RawMaterial) RETURN rm.name as name`, params)
	for _, rm := range rawMaterials {
		id := fmt.Sprintf("mat_%s", rm["name"])
		nodes = append(nodes, map[string]any{"id": id, "label": rm["name"], "type": "RawMaterial"})
		links = append(links, map[string]any{"source": ticker, "target": id, "type": "CONSUMES"})
	}

	// Deduplicate nodes by id
	seen := map[string]bool{}
	var uniqueNodes []map[string]any
	for _, n := range nodes {
		id := fmt.Sprintf("%s", n["id"])
		if !seen[id] {
			seen[id] = true
			uniqueNodes = append(uniqueNodes, n)
		}
	}

	writeJSON(w, map[string]any{"nodes": uniqueNodes, "links": links})
}
