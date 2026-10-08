package csvlog

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Alechan/ai-resources/tools/gh-copilot-credits/src/internal/copilot"
)

func TestAppendCreatesHeaderAndSnapshot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "usage.csv")
	snapshot := Snapshot{
		ObservedAtUTC:     "2026-09-26T13:35:02Z",
		APITimestampUTC:   "2026-09-26T13:35:02.302Z",
		Source:            "gh-copilot-credits",
		CopilotPlan:       "business",
		QuotaID:           copilot.PremiumInteractionsQuotaID,
		QuotaResetDateUTC: "2026-10-01T00:00:00Z",
		CreditsUsed:       "17728",
		DerivedUsed:       17728,
		Entitlement:       20000,
		QuotaRemaining:    2272,
		Remaining:         2272,
		PercentRemaining:  11.36,
		OveragePermitted:  true,
	}

	if err := Append(path, snapshot); err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	text := string(data)
	if strings.Count(text, "schema_version") != 1 {
		t.Fatalf("header count = %d, csv = %s", strings.Count(text, "schema_version"), text)
	}
	if strings.Contains(text, "organization_list") || strings.Contains(text, "access_token") {
		t.Fatalf("CSV contains excluded metadata: %s", text)
	}
	rows, err := csv.NewReader(strings.NewReader(text)).ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("row count = %d, want 2", len(rows))
	}
	if got, want := rows[1][0], "1"; got != want {
		t.Fatalf("schema version = %q, want %q", got, want)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("file permissions = %o, want 600", info.Mode().Perm())
	}
}

func TestAppendDoesNotDuplicateHeader(t *testing.T) {
	path := filepath.Join(t.TempDir(), "usage.csv")
	snapshot := Snapshot{
		ObservedAtUTC:   "2026-09-26T13:35:02Z",
		APITimestampUTC: "2026-09-26T13:35:02Z",
		QuotaID:         copilot.PremiumInteractionsQuotaID,
		CreditsUsed:     "10",
		DerivedUsed:     10,
		Entitlement:     20,
		QuotaRemaining:  10,
		Remaining:       10,
	}
	if err := Append(path, snapshot); err != nil {
		t.Fatal(err)
	}
	snapshot.ObservedAtUTC = "2026-09-26T19:35:02Z"
	if err := Append(path, snapshot); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(data), "schema_version"); got != 1 {
		t.Fatalf("header count = %d, want 1", got)
	}
}

func TestAppendRejectsUnexpectedHeader(t *testing.T) {
	path := filepath.Join(t.TempDir(), "usage.csv")
	if err := os.WriteFile(path, []byte("not,the,expected,header\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := Append(path, Snapshot{QuotaID: copilot.PremiumInteractionsQuotaID})
	if err == nil || !strings.Contains(err.Error(), "header") {
		t.Fatalf("Append() error = %v, want header error", err)
	}
}

func TestSnapshotFromQuota(t *testing.T) {
	data := copilot.UserData{
		CopilotPlan:       "business",
		QuotaResetDateUTC: "2026-10-01T00:00:00Z",
	}
	used := 17728.0
	quota := copilot.QuotaSnapshot{
		QuotaID:          copilot.PremiumInteractionsQuotaID,
		TimestampUTC:     "2026-09-26T13:35:02Z",
		Entitlement:      20000,
		QuotaRemaining:   2272,
		Remaining:        2272,
		PercentRemaining: 11.36,
		CreditsUsed:      &used,
		OveragePermitted: true,
	}
	snapshot := SnapshotFrom(data, quota, time.Date(2026, 9, 26, 13, 35, 2, 0, time.UTC))
	if snapshot.CreditsUsed != "17728" || snapshot.DerivedUsed != 17728 {
		t.Fatalf("snapshot usage = %q/%v", snapshot.CreditsUsed, snapshot.DerivedUsed)
	}
	if snapshot.CopilotPlan != "business" {
		t.Fatalf("plan = %q", snapshot.CopilotPlan)
	}
}
