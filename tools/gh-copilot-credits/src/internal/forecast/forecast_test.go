package forecast

import (
	"testing"
	"time"
)

func TestCalculateUsesCalendarAndLocalWeekdayCounts(t *testing.T) {
	location := time.FixedZone("local", 2*60*60)
	reset := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 9, 26, 12, 0, 0, 0, location)

	got, err := Calculate(18023, 20000, reset, now)
	if err != nil {
		t.Fatalf("Calculate() error = %v", err)
	}
	if got.CalendarDaysElapsed != 26 || got.CalendarDaysTotal != 30 {
		t.Fatalf("calendar days = %d/%d, want 26/30", got.CalendarDaysElapsed, got.CalendarDaysTotal)
	}
	if got.WeekdaysElapsed != 19 || got.WeekdaysTotal != 22 {
		t.Fatalf("weekdays = %d/%d, want 19/22", got.WeekdaysElapsed, got.WeekdaysTotal)
	}
	if got.WeekendsElapsed != 7 || got.WeekendsTotal != 8 {
		t.Fatalf("weekends = %d/%d, want 7/8", got.WeekendsElapsed, got.WeekendsTotal)
	}
	if got.CalendarRate <= 0 || got.WeekdayRate <= 0 {
		t.Fatalf("rates = %v/%v, want positive rates", got.CalendarRate, got.WeekdayRate)
	}
	if got.CalendarProjected <= 0 || got.WeekdayProjected <= 0 {
		t.Fatalf("projections = %v/%v, want positive projections", got.CalendarProjected, got.WeekdayProjected)
	}
	if got.CalendarRequiredPace <= 0 || got.WeekdayRequiredPace <= 0 {
		t.Fatalf("required paces = %v/%v, want positive paces", got.CalendarRequiredPace, got.WeekdayRequiredPace)
	}
}

func TestCalculateAssumesNoWeekendUsageForWeekdayProjection(t *testing.T) {
	reset := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	got, err := Calculate(100, 1000, reset, now)
	if err != nil {
		t.Fatal(err)
	}
	if got.WeekdaysRemaining != 0 {
		t.Fatalf("WeekdaysRemaining = %d, want 0", got.WeekdaysRemaining)
	}
	if got.WeekdayRequiredPace != 0 {
		t.Fatalf("WeekdayRequiredPace = %v, want 0", got.WeekdayRequiredPace)
	}
}

func TestCalculateRejectsInvalidOrOutOfPeriodDates(t *testing.T) {
	reset := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	for name, now := range map[string]time.Time{
		"before period": time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC),
		"after reset":   time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Calculate(1, 10, reset, now); err == nil {
				t.Fatal("Calculate() error = nil, want error")
			}
		})
	}
}
