package output

import (
	"strings"
	"testing"

	"github.com/Alechan/ai-resources/tools/gh-copilot-credits/src/internal/copilot"
	"github.com/Alechan/ai-resources/tools/gh-copilot-credits/src/internal/forecast"
)

func TestRenderCurrentTable(t *testing.T) {
	report := CurrentReport{
		ObservedAtUTC:     "2026-09-26T13:35:02Z",
		APITimestampUTC:   "2026-09-26T13:35:02.302Z",
		CopilotPlan:       "business",
		QuotaResetDateUTC: "2026-10-01T00:00:00Z",
		Quota: copilot.QuotaSnapshot{
			QuotaID:          copilot.PremiumInteractionsQuotaID,
			Entitlement:      20000,
			QuotaRemaining:   2272,
			Remaining:        2272,
			PercentRemaining: 11.36,
			CreditsUsed:      floatPtr(17728),
		},
		Forecast: &forecast.Report{
			PeriodStart:              "2026-09-01",
			PeriodEnd:                "2026-09-30",
			AsOfDate:                 "2026-09-26",
			CalendarRate:             684.0,
			CalendarProjected:        20520,
			WeekdayRate:              933.0,
			WeekdayProjected:         20526,
			WeekdayProjectionAssumes: "weekend usage is zero",
		},
	}

	got := RenderCurrentTable(report)
	for _, want := range []string{
		"GitHub Copilot AI credits",
		"2026-09-26T13:35:02Z",
		"premium_interactions",
		"Reported Used",
		"Derived Used",
		"17728",
		"20000",
		"2272",
		"11.36%",
		"Burn-rate estimates",
		"Calendar-day",
		"Weekday",
		"Elapsed",
		"Projected total",
		"Required pace",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("RenderCurrentTable() does not contain %q\noutput:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Assumption:") {
		t.Errorf("RenderCurrentTable() contains an assumption line:\n%s", got)
	}
}

func TestRenderCurrentJSONExcludesRawAccountMetadata(t *testing.T) {
	report := CurrentReport{
		ObservedAtUTC: "2026-09-26T13:35:02Z",
		Quota: copilot.QuotaSnapshot{
			QuotaID:        copilot.PremiumInteractionsQuotaID,
			Entitlement:    20000,
			QuotaRemaining: 2272,
			Remaining:      2272,
		},
	}

	got, err := RenderCurrentJSON(report)
	if err != nil {
		t.Fatalf("RenderCurrentJSON() error = %v", err)
	}
	if !strings.Contains(got, `"quota_id"`) {
		t.Fatalf("JSON output does not contain quota_id: %s", got)
	}
	for _, forbidden := range []string{"organization_list", "analytics_tracking_id", "access_token"} {
		if strings.Contains(got, forbidden) {
			t.Errorf("JSON output contains forbidden field %q: %s", forbidden, got)
		}
	}
}

func floatPtr(value float64) *float64 { return &value }
