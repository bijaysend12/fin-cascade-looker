package handlers

import (
	"fmt"
	"math"
	"net/http"
)

// Financial-sector tickers where forensic signals don't apply.
// Banking uses NIM/GNPA/CASA instead; Ambit G&C methodology is
// calibrated for non-financial companies.
var forensicExcludedSectors = map[string]bool{
	"Banking":             true,
	"Financial Services":  true,
	"Insurance & NBFC":    true,
}

const forensicCoverageFloor = 0.60

type forensicCheck struct {
	Field     string `json:"field"`
	Label     string `json:"label"`
	Threshold string `json:"threshold"`
	Passed    bool   `json:"passed"`
	Value     any    `json:"value"`
}

func (h *Handler) GetCompanyForensic(w http.ResponseWriter, r *http.Request) {
	ticker := r.PathValue("ticker")
	if ticker == "" {
		writeError(w, 400, "ticker required")
		return
	}

	rows, err := h.Neo4j.Query(
		`MATCH (c:Company {ticker: $ticker})
		RETURN c.ticker AS ticker, c.name AS name, c.sector AS sector,
			c.fcf_to_pat_ratio AS fcf_to_pat_ratio,
			c.cwip_to_sales_ratio AS cwip_to_sales_ratio,
			c.tax_rate_vs_normal AS tax_rate_vs_normal,
			c.other_income_share AS other_income_share,
			c.interest_coverage_weak AS interest_coverage_weak,
			c.other_liabilities_share AS other_liabilities_share,
			c.cash_conversion_stretched AS cash_conversion_stretched,
			c.reserves_to_equity AS reserves_to_equity,
			c.dividend_payout_stable AS dividend_payout_stable,
			c.roce_growth_gap AS roce_growth_gap,
			c.debtor_days_concern AS debtor_days_concern,
			c.inventory_days_concern AS inventory_days_concern,
			c.ttm_sales_momentum AS ttm_sales_momentum,
			c.forensic_signals_computed_at AS computed_at`,
		map[string]any{"ticker": ticker})
	if err != nil {
		writeError(w, 500, "internal error")
		return
	}
	if len(rows) == 0 {
		writeError(w, 404, fmt.Sprintf("company %s not found", ticker))
		return
	}

	row := rows[0]
	sector, _ := row["sector"].(string)
	computedAt := row["computed_at"]

	if forensicExcludedSectors[sector] {
		writeJSON(w, map[string]any{
			"ticker":      ticker,
			"score":       nil,
			"reason":      "not_applicable",
			"sector":      sector,
			"computed_at": computedAt,
		})
		return
	}

	checks, missing := evaluateForensicChecks(row)

	totalPossible := len(checks) + len(missing)
	applicable := len(checks)
	coverage := float64(applicable) / float64(totalPossible)

	if coverage < forensicCoverageFloor {
		writeJSON(w, map[string]any{
			"ticker":          ticker,
			"score":           nil,
			"reason":          "insufficient_data",
			"sector":          sector,
			"applicable":      applicable,
			"total_possible":  totalPossible,
			"coverage":        coverage,
			"coverage_floor":  forensicCoverageFloor,
			"missing":         missing,
			"computed_at":     computedAt,
		})
		return
	}

	passed := 0
	for _, c := range checks {
		if c.Passed {
			passed++
		}
	}
	score := int(math.Round(float64(passed) / float64(applicable) * 100))

	writeJSON(w, map[string]any{
		"ticker":          ticker,
		"sector":          sector,
		"score":           score,
		"passed":          passed,
		"applicable":      applicable,
		"total_possible":  totalPossible,
		"coverage":        coverage,
		"coverage_floor":  forensicCoverageFloor,
		"checks":          checks,
		"missing":         missing,
		"computed_at":     computedAt,
	})
}

func evaluateForensicChecks(row map[string]any) (checks []forensicCheck, missing []string) {
	addNumeric := func(field, label, threshold string, pass func(float64) bool) {
		if v, ok := asFloat(row[field]); ok {
			checks = append(checks, forensicCheck{
				Field: field, Label: label, Threshold: threshold,
				Passed: pass(v), Value: v,
			})
		} else {
			missing = append(missing, field)
		}
	}
	addBool := func(field, label, threshold string, pass func(bool) bool) {
		if v, ok := row[field].(bool); ok {
			checks = append(checks, forensicCheck{
				Field: field, Label: label, Threshold: threshold,
				Passed: pass(v), Value: v,
			})
		} else {
			missing = append(missing, field)
		}
	}

	addNumeric("fcf_to_pat_ratio", "Cash conversion (FCF/PAT)", ">= 0.5",
		func(v float64) bool { return v >= 0.5 })
	addNumeric("cwip_to_sales_ratio", "CWIP vs sales", "<= 0.3",
		func(v float64) bool { return v <= 0.3 })
	addNumeric("tax_rate_vs_normal", "Tax rate vs India norm", ">= 0.6",
		func(v float64) bool { return v >= 0.6 })
	addNumeric("other_income_share", "Other income share of PBT", "<= 0.3",
		func(v float64) bool { return v <= 0.3 })
	addBool("interest_coverage_weak", "Interest coverage", "not weak",
		func(v bool) bool { return !v })
	addNumeric("other_liabilities_share", "Other liabilities share", "<= 0.4",
		func(v float64) bool { return v <= 0.4 })
	addBool("cash_conversion_stretched", "Cash conversion cycle", "not stretched",
		func(v bool) bool { return !v })
	addNumeric("reserves_to_equity", "Reserves vs equity capital", ">= 1.0",
		func(v float64) bool { return v >= 1.0 })
	addBool("dividend_payout_stable", "Dividend payout stability", "stable (10–80%)",
		func(v bool) bool { return v })
	addNumeric("roce_growth_gap", "ROCE vs sales growth gap", ">= 0",
		func(v float64) bool { return v >= 0 })
	addBool("debtor_days_concern", "Debtor days", "not elevated",
		func(v bool) bool { return !v })
	addBool("inventory_days_concern", "Inventory days", "not elevated",
		func(v bool) bool { return !v })
	addNumeric("ttm_sales_momentum", "TTM sales momentum", ">= -0.05",
		func(v float64) bool { return v >= -0.05 })

	return checks, missing
}

func asFloat(v any) (float64, bool) {
	if v == nil {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int64:
		return float64(n), true
	case int:
		return float64(n), true
	}
	return 0, false
}
