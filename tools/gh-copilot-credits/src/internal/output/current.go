package output

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Alechan/ai-resources/tools/gh-copilot-credits/src/internal/copilot"
	"github.com/Alechan/ai-resources/tools/gh-copilot-credits/src/internal/forecast"
)

type CurrentReport struct {
	ObservedAtUTC     string                `json:"observed_at_utc"`
	APITimestampUTC   string                `json:"api_timestamp_utc"`
	CopilotPlan       string                `json:"copilot_plan,omitempty"`
	QuotaResetDateUTC string                `json:"quota_reset_date_utc,omitempty"`
	Quota             copilot.QuotaSnapshot `json:"quota"`
	Forecast          *forecast.Report      `json:"forecast,omitempty"`
}

func RenderCurrentTable(report CurrentReport) string {
	var builder strings.Builder
	fmt.Fprintln(&builder, "GitHub Copilot AI credits")
	fmt.Fprintln(&builder, "─────────────────────────")
	fmt.Fprintf(&builder, "Observed:        %s\n", report.ObservedAtUTC)
	fmt.Fprintf(&builder, "API timestamp:   %s\n", report.APITimestampUTC)
	fmt.Fprintf(&builder, "Plan:            %s\n", report.CopilotPlan)
	fmt.Fprintf(&builder, "Reset:           %s\n\n", report.QuotaResetDateUTC)
	fmt.Fprintln(&builder, "Quota                  Reported Used    Derived Used    Entitlement    Remaining    Remaining %")
	reportedUsed := report.Quota.DerivedUsedCredits()
	if report.Quota.CreditsUsed != nil {
		reportedUsed = *report.Quota.CreditsUsed
	}
	fmt.Fprintf(
		&builder,
		"%-22s %13s %13s %13s %12s %15s\n",
		report.Quota.QuotaID,
		formatNumber(reportedUsed),
		formatNumber(report.Quota.DerivedUsedCredits()),
		formatNumber(report.Quota.Entitlement),
		formatNumber(report.Quota.Remaining),
		formatPercent(report.Quota.PercentRemaining),
	)
	if report.Forecast != nil {
		fmt.Fprintln(&builder)
		fmt.Fprintln(&builder, "Burn-rate estimates")
		fmt.Fprintf(&builder, "Period:     %s to %s (as of %s)\n", report.Forecast.PeriodStart, report.Forecast.PeriodEnd, report.Forecast.AsOfDate)
		fmt.Fprintf(&builder, "%-22s %-25s %-25s\n", "Metric", "Calendar-day", "Weekday")
		fmt.Fprintf(&builder, "%-22s %-25s %-25s\n", "Elapsed", fmt.Sprintf("%d/%d days", report.Forecast.CalendarDaysElapsed, report.Forecast.CalendarDaysTotal), fmt.Sprintf("%d/%d weekdays", report.Forecast.WeekdaysElapsed, report.Forecast.WeekdaysTotal))
		fmt.Fprintf(&builder, "%-22s %-25s %-25s\n", "Remaining", fmt.Sprintf("%d days", report.Forecast.CalendarDaysRemaining), fmt.Sprintf("%d weekdays", report.Forecast.WeekdaysRemaining))
		fmt.Fprintf(&builder, "%-22s %-25s %-25s\n", "Weekend days", fmt.Sprintf("%d/%d", report.Forecast.WeekendsElapsed, report.Forecast.WeekendsTotal), "—")
		fmt.Fprintf(&builder, "%-22s %-25s %-25s\n", "Rate", fmt.Sprintf("%.2f credits/day", report.Forecast.CalendarRate), fmt.Sprintf("%.2f credits/workday", report.Forecast.WeekdayRate))
		fmt.Fprintf(&builder, "%-22s %-25s %-25s\n", "Projected total", fmt.Sprintf("%.2f credits", report.Forecast.CalendarProjected), fmt.Sprintf("%.2f credits", report.Forecast.WeekdayProjected))
		fmt.Fprintf(&builder, "%-22s %-25s %-25s\n", "Required pace", fmt.Sprintf("%.2f credits/day", report.Forecast.CalendarRequiredPace), fmt.Sprintf("%.2f credits/workday", report.Forecast.WeekdayRequiredPace))
	}
	return builder.String()
}

func RenderCurrentJSON(report CurrentReport) (string, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode current report: %w", err)
	}
	return string(data) + "\n", nil
}

func formatNumber(value float64) string {
	if value == float64(int64(value)) {
		return fmt.Sprintf("%d", int64(value))
	}
	return fmt.Sprintf("%.2f", value)
}

func formatPercent(value float64) string {
	return fmt.Sprintf("%.2f%%", value)
}
