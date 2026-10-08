package forecast

import (
	"errors"
	"fmt"
	"time"
)

type Report struct {
	PeriodStart              string  `json:"period_start"`
	PeriodEnd                string  `json:"period_end"`
	AsOfDate                 string  `json:"as_of_date"`
	UsedCredits              float64 `json:"used_credits"`
	EntitlementCredits       float64 `json:"entitlement_credits"`
	RemainingCredits         float64 `json:"remaining_credits"`
	CalendarDaysElapsed      int     `json:"calendar_days_elapsed"`
	CalendarDaysTotal        int     `json:"calendar_days_total"`
	CalendarDaysRemaining    int     `json:"calendar_days_remaining"`
	WeekdaysElapsed          int     `json:"weekdays_elapsed"`
	WeekdaysTotal            int     `json:"weekdays_total"`
	WeekdaysRemaining        int     `json:"weekdays_remaining"`
	WeekendsElapsed          int     `json:"weekends_elapsed"`
	WeekendsTotal            int     `json:"weekends_total"`
	CalendarRate             float64 `json:"calendar_rate_credits_per_day"`
	CalendarProjected        float64 `json:"calendar_projected_total"`
	CalendarRequiredPace     float64 `json:"calendar_required_pace"`
	WeekdayRate              float64 `json:"weekday_rate_credits_per_workday"`
	WeekdayProjected         float64 `json:"weekday_projected_total"`
	WeekdayRequiredPace      float64 `json:"weekday_required_pace"`
	WeekdayProjectionAssumes string  `json:"weekday_projection_assumes"`
}

func Calculate(used, entitlement float64, resetUTC, now time.Time) (Report, error) {
	if entitlement <= 0 {
		return Report{}, errors.New("entitlement must be positive")
	}
	if resetUTC.IsZero() {
		return Report{}, errors.New("quota reset date is required")
	}
	location := now.Location()
	resetYear, resetMonth, resetDay := resetUTC.Date()
	resetDate := time.Date(resetYear, resetMonth, resetDay, 0, 0, 0, 0, location)
	periodStart := resetDate.AddDate(0, -1, 0)
	periodEnd := resetDate.AddDate(0, 0, -1)
	asOf := dateOnly(now)
	if asOf.Before(periodStart) || asOf.After(periodEnd) {
		return Report{}, fmt.Errorf("current date %s is outside billing period %s to %s", formatDate(asOf), formatDate(periodStart), formatDate(periodEnd))
	}

	calendarTotal := daysInclusive(periodStart, periodEnd)
	calendarElapsed := daysInclusive(periodStart, asOf)
	weekdaysTotal := countWeekdays(periodStart, periodEnd)
	weekdaysElapsed := countWeekdays(periodStart, asOf)
	weekendsTotal := calendarTotal - weekdaysTotal
	weekendsElapsed := calendarElapsed - weekdaysElapsed
	remainingCredits := entitlement - used
	calendarRemaining := calendarTotal - calendarElapsed
	weekdaysRemaining := weekdaysTotal - weekdaysElapsed

	report := Report{
		PeriodStart:              formatDate(periodStart),
		PeriodEnd:                formatDate(periodEnd),
		AsOfDate:                 formatDate(asOf),
		UsedCredits:              used,
		EntitlementCredits:       entitlement,
		RemainingCredits:         remainingCredits,
		CalendarDaysElapsed:      calendarElapsed,
		CalendarDaysTotal:        calendarTotal,
		CalendarDaysRemaining:    calendarRemaining,
		WeekdaysElapsed:          weekdaysElapsed,
		WeekdaysTotal:            weekdaysTotal,
		WeekdaysRemaining:        weekdaysRemaining,
		WeekendsElapsed:          weekendsElapsed,
		WeekendsTotal:            weekendsTotal,
		CalendarRate:             divide(used, calendarElapsed),
		CalendarProjected:        multiplyRate(used, calendarElapsed, calendarTotal),
		CalendarRequiredPace:     requiredPace(remainingCredits, calendarRemaining),
		WeekdayRate:              divide(used, weekdaysElapsed),
		WeekdayProjected:         multiplyRate(used, weekdaysElapsed, weekdaysTotal),
		WeekdayRequiredPace:      requiredPace(remainingCredits, weekdaysRemaining),
		WeekdayProjectionAssumes: "weekend usage is zero; usage is projected across actual local weekdays",
	}
	return report, nil
}

func dateOnly(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, value.Location())
}

func formatDate(value time.Time) string { return value.Format("2006-01-02") }

func countWeekdays(start, end time.Time) int {
	count := 0
	for current := start; !current.After(end); current = current.AddDate(0, 0, 1) {
		if current.Weekday() != time.Saturday && current.Weekday() != time.Sunday {
			count++
		}
	}
	return count
}

func daysInclusive(start, end time.Time) int {
	startUTC := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	endUTC := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, time.UTC)
	return int(endUTC.Sub(startUTC).Hours()/24) + 1
}

func divide(value float64, divisor int) float64 {
	if divisor <= 0 {
		return 0
	}
	return value / float64(divisor)
}

func multiplyRate(used float64, elapsed, total int) float64 {
	if elapsed <= 0 {
		return 0
	}
	return used / float64(elapsed) * float64(total)
}

func requiredPace(remaining float64, days int) float64 {
	if remaining <= 0 || days <= 0 {
		return 0
	}
	return remaining / float64(days)
}
