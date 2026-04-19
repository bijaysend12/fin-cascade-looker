package handlers

import (
	"net/http"
	"time"
)

func (h *Handler) StaticSync(w http.ResponseWriter, r *http.Request) {
	since := queryStr(r, "since")

	if since != "" {
		if _, err := time.Parse(time.RFC3339, since); err != nil {
			writeError(w, 400, "invalid since format, use RFC3339")
			return
		}
	}

	result := map[string]any{
		"sync_version": time.Now().UTC().Format(time.RFC3339),
	}

	result["companies"] = h.syncCompaniesStatic(since)
	result["sectors"] = h.syncSectorsStatic(since)
	result["graph_edges"] = h.syncGraphEdges()

	writeJSON(w, result)
}

func (h *Handler) syncCompaniesStatic(since string) []map[string]any {
	if h.Neo4j == nil {
		return []map[string]any{}
	}

	cypher := `MATCH (c:Company)`
	params := map[string]any{}
	if since != "" {
		cypher += ` WHERE c.enriched_at IS NULL OR c.enriched_at > $since`
		params["since"] = since
	}
	cypher += `
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
			c.nse_listed AS nse_listed
		ORDER BY c.ticker`

	records, err := h.Neo4j.Query(cypher, params)
	if err != nil || records == nil {
		return []map[string]any{}
	}
	return records
}

func (h *Handler) syncSectorsStatic(since string) []map[string]any {
	if h.Neo4j == nil {
		return []map[string]any{}
	}

	cypher := `MATCH (s:Sector)`
	params := map[string]any{}
	if since != "" {
		cypher += ` WHERE s.medians_computed_at IS NULL OR s.medians_computed_at > $since`
		params["since"] = since
	}
	cypher += `
		RETURN s.name AS name,
			s.median_pe AS median_pe, s.median_pb AS median_pb,
			s.median_roce AS median_roce, s.median_roe AS median_roe,
			s.median_de AS median_de, s.median_op_margin AS median_op_margin,
			s.medians_computed_at AS medians_computed_at,
			s.upstreamDependencies AS upstream_dependencies,
			s.downstreamSectors AS downstream_sectors
		ORDER BY s.name`

	records, err := h.Neo4j.Query(cypher, params)
	if err != nil || records == nil {
		return []map[string]any{}
	}
	return records
}

func (h *Handler) syncGraphEdges() []map[string]any {
	if h.Neo4j == nil {
		return []map[string]any{}
	}

	records, err := h.Neo4j.Query(
		`MATCH (a:Company)-[r]->(b)
		WHERE type(r) IN ['COMPETES_WITH','SUPPLIES_TO','CONSUMES','DEPENDS_ON_COMMODITY','SOURCES_API_FROM','DISTRIBUTES_FOR','HAS_PLANT']
		RETURN a.ticker AS source_ticker,
			coalesce(b.ticker, b.name) AS target_id,
			b.name AS target_name,
			type(r) AS edge_type,
			labels(b)[0] AS target_type,
			properties(r) AS props
		ORDER BY a.ticker, type(r)`, nil)
	if err != nil || records == nil {
		return []map[string]any{}
	}
	return records
}
