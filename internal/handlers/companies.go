package handlers

import (
	"fmt"
	"net/http"
)

func (h *Handler) ListCompanies(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		writeError(w, 403, "admin access required")
		return
	}
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
		writeError(w, 500, "internal error")
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
		writeError(w, 500, "internal error")
		return
	}
	if records == nil {
		records = []map[string]any{}
	}

	writeJSON(w, map[string]any{"companies": records, "total": total})
}

func (h *Handler) GetCompany(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		writeError(w, 403, "admin access required")
		return
	}
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
		writeError(w, 500, "internal error")
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

func (h *Handler) GetCompanyFundamentals(w http.ResponseWriter, r *http.Request) {
	ticker := r.PathValue("ticker")
	if ticker == "" {
		writeError(w, 400, "ticker required")
		return
	}

	rows, err := h.Neo4j.Query(
		`MATCH (c:Company {ticker: $ticker})
		RETURN c.ticker as ticker, c.name as name,
		c.pe as pe, c.pb as pb, c.ev_to_ebitda as ev_to_ebitda,
		c.enterprise_value as enterprise_value, c.market_cap as market_cap,
		c.dividend_yield as dividend_yield,
		c.fifty_two_week_high as fifty_two_week_high, c.fifty_two_week_low as fifty_two_week_low,
		c.roce as roce, c.roe as roe,
		c.op_margin as op_margin, c.profit_margin as profit_margin,
		c.gross_margin as gross_margin, c.ebitda_margin as ebitda_margin,
		c.debt_to_equity as debt_to_equity, c.total_debt as total_debt,
		c.total_cash as total_cash, c.free_cashflow as free_cashflow,
		c.borrowings as borrowings, c.operating_cashflow as operating_cashflow,
		c.sales_growth_3yr as sales_growth_3yr, c.sales_growth_5yr as sales_growth_5yr,
		c.profit_growth_3yr as profit_growth_3yr, c.profit_growth_5yr as profit_growth_5yr,
		c.promoter_pct as promoter_pct, c.fii_pct as fii_pct,
		c.dii_pct as dii_pct, c.public_pct as public_pct,
		c.analyst_rec as analyst_rec, c.target_price as target_price,
		c.analyst_count as analyst_count, c.beta as beta,
		c.annual_revenue as annual_revenue, c.annual_profit as annual_profit,
		c.enriched_at as enriched_at, c.enrichment_source as enrichment_source`,
		map[string]any{"ticker": ticker})
	if err != nil {
		writeError(w, 500, "internal error")
		return
	}
	if len(rows) == 0 {
		writeError(w, 404, fmt.Sprintf("company %s not found", ticker))
		return
	}

	writeJSON(w, rows[0])
}

func (h *Handler) GetCompanyStatic(w http.ResponseWriter, r *http.Request) {
	ticker := r.PathValue("ticker")
	if ticker == "" {
		writeError(w, 400, "ticker required")
		return
	}

	rows, err := h.Neo4j.Query(
		`MATCH (c:Company {ticker: $ticker})
		RETURN c.ticker AS ticker, c.name AS name, c.sector AS sector, c.industry AS industry,
			c.marketCapCategory AS market_cap_category, c.hqCity AS hq_city, c.hqState AS hq_state,
			c.description AS description, c.exportPct AS export_pct,

			c.pe AS pe, c.pb AS pb, c.market_cap AS market_cap,
			c.dividend_yield AS dividend_yield, c.ev_to_ebitda AS ev_to_ebitda,
			c.enterprise_value AS enterprise_value,
			c.fifty_two_week_high AS fifty_two_week_high,
			c.fifty_two_week_low AS fifty_two_week_low,

			c.roce AS roce, c.roe AS roe,
			c.op_margin AS op_margin, c.profit_margin AS profit_margin,
			c.gross_margin AS gross_margin, c.ebitda_margin AS ebitda_margin,

			c.debt_to_equity AS debt_to_equity, c.total_debt AS total_debt,
			c.total_cash AS total_cash, c.free_cashflow AS free_cashflow,
			c.borrowings AS borrowings, c.operating_cashflow AS operating_cashflow,

			c.sales_growth_3yr AS sales_growth_3yr, c.sales_growth_5yr AS sales_growth_5yr,
			c.profit_growth_3yr AS profit_growth_3yr, c.profit_growth_5yr AS profit_growth_5yr,

			c.promoter_pct AS promoter_pct, c.fii_pct AS fii_pct,
			c.dii_pct AS dii_pct, c.public_pct AS public_pct,
			c.promoter_pledge_pct AS promoter_pledge_pct,

			c.annual_revenue AS annual_revenue, c.annual_profit AS annual_profit,

			c.analyst_rec AS analyst_rec, c.target_price AS target_price,
			c.analyst_count AS analyst_count, c.beta AS beta,

			c.about AS about, c.description_short AS description_short,
			c.description_long AS description_long,
			c.pros AS pros_json, c.cons AS cons_json,
			c.key_segments_json AS key_segments_json,

			c.nim AS nim, c.gnpa AS gnpa, c.nnpa AS nnpa, c.casa_ratio AS casa_ratio,
			c.car AS car, c.pcr AS pcr, c.cost_to_income AS cost_to_income,
			c.revenue_us_pct AS revenue_us_pct, c.revenue_bfsi_pct AS revenue_bfsi_pct,
			c.attrition_rate AS attrition_rate,
			c.us_revenue_pct AS us_revenue_pct, c.rd_pct AS rd_pct,
			c.fda_status AS fda_status, c.anda_count AS anda_count,
			c.plf AS plf, c.installed_capacity_mw AS installed_capacity_mw,
			c.fuel_type AS fuel_type,
			c.vnb_margin AS vnb_margin, c.embedded_value AS embedded_value,
			c.persistency_13m AS persistency_13m, c.combined_ratio AS combined_ratio,
			c.solvency_ratio AS solvency_ratio,
			c.volume_growth_pct AS volume_growth_pct, c.rural_revenue_pct AS rural_revenue_pct,
			c.market_share_pct AS market_share_pct,
			c.monthly_sales_units AS monthly_sales_units, c.ev_mix_pct AS ev_mix_pct,
			c.capacity_utilization_pct AS capacity_utilization_pct,
			c.grm_usd_bbl AS grm_usd_bbl, c.crude_sourcing_mix AS crude_sourcing_mix,
			c.ebitda_per_tonne AS ebitda_per_tonne,
			c.installed_capacity_mtpa AS installed_capacity_mtpa,
			c.dgtr_status AS dgtr_status, c.downstream_applications AS downstream_applications,
			c.china_plus_one AS china_plus_one,
			c.order_book_cr AS order_book_cr, c.indigenous_pct AS indigenous_pct,
			c.defence_export_cr AS defence_export_cr,

			c.enriched_at AS enriched_at,
			c.enrichment_version AS enrichment_version,
			c.enrichment_status AS enrichment_status,
			c.enrichment_source AS enrichment_source,
			c.nse_listed AS nse_listed`,
		map[string]any{"ticker": ticker})
	if err != nil {
		writeError(w, 500, "internal error")
		return
	}
	if len(rows) == 0 {
		writeError(w, 404, fmt.Sprintf("company %s not found", ticker))
		return
	}

	edges, _ := h.Neo4j.Query(
		`MATCH (a:Company {ticker: $ticker})-[r]->(b)
		WHERE type(r) IN ['COMPETES_WITH','SUPPLIES_TO','CONSUMES','DEPENDS_ON_COMMODITY','SOURCES_API_FROM','DISTRIBUTES_FOR','HAS_PLANT']
		RETURN a.ticker AS source_ticker,
			coalesce(b.ticker, b.name) AS target_id,
			b.name AS target_name,
			type(r) AS edge_type,
			labels(b)[0] AS target_type,
			properties(r) AS props`,
		map[string]any{"ticker": ticker})
	if edges == nil {
		edges = []map[string]any{}
	}

	result := rows[0]
	result["graph_edges"] = edges

	writeJSON(w, result)
}

func (h *Handler) GetCompanyGraph(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		writeError(w, 403, "admin access required")
		return
	}
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
