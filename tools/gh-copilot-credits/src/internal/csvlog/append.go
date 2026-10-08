package csvlog

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"syscall"
	"time"

	"github.com/Alechan/ai-resources/tools/gh-copilot-credits/src/internal/copilot"
)

var header = []string{
	"schema_version",
	"observed_at_utc",
	"api_timestamp_utc",
	"source",
	"quota_id",
	"copilot_plan",
	"quota_reset_date_utc",
	"credits_used",
	"derived_used_credits",
	"entitlement_credits",
	"quota_remaining",
	"remaining_credits",
	"percent_remaining",
	"overage_permitted",
	"overage_count",
	"unlimited",
	"token_based_billing",
}

type Snapshot struct {
	ObservedAtUTC     string
	APITimestampUTC   string
	Source            string
	QuotaID           string
	CopilotPlan       string
	QuotaResetDateUTC string
	CreditsUsed       string
	DerivedUsed       float64
	Entitlement       float64
	QuotaRemaining    float64
	Remaining         float64
	PercentRemaining  float64
	OveragePermitted  bool
	OverageCount      float64
	Unlimited         bool
	TokenBasedBilling bool
}

func SnapshotFrom(data copilot.UserData, quota copilot.QuotaSnapshot, observedAt time.Time) Snapshot {
	creditsUsed := ""
	if quota.CreditsUsed != nil {
		creditsUsed = formatNumber(*quota.CreditsUsed)
	}
	return Snapshot{
		ObservedAtUTC:     observedAt.UTC().Format(time.RFC3339),
		APITimestampUTC:   quota.TimestampUTC,
		Source:            "gh-copilot-credits",
		QuotaID:           quota.QuotaID,
		CopilotPlan:       data.CopilotPlan,
		QuotaResetDateUTC: data.QuotaResetDateUTC,
		CreditsUsed:       creditsUsed,
		DerivedUsed:       quota.DerivedUsedCredits(),
		Entitlement:       quota.Entitlement,
		QuotaRemaining:    quota.QuotaRemaining,
		Remaining:         quota.Remaining,
		PercentRemaining:  quota.PercentRemaining,
		OveragePermitted:  quota.OveragePermitted,
		OverageCount:      quota.OverageCount,
		Unlimited:         quota.Unlimited,
		TokenBasedBilling: quota.TokenBasedBilling || data.TokenBasedBilling,
	}
}

func Append(path string, snapshot Snapshot) error {
	if path == "" {
		return errors.New("CSV path is required")
	}
	parent := filepath.Dir(path)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return fmt.Errorf("create CSV directory: %w", err)
	}

	lock, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("open CSV lock: %w", err)
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("lock CSV: %w", err)
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)

	existing, err := os.Open(path)
	if err == nil {
		rows, readErr := csv.NewReader(existing).Read()
		closeErr := existing.Close()
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return fmt.Errorf("read CSV header: %w", readErr)
		}
		if closeErr != nil {
			return fmt.Errorf("close CSV: %w", closeErr)
		}
		if !slices.Equal(rows, header) {
			return errors.New("CSV header does not match gh-copilot-credits schema")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("open existing CSV: %w", err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open CSV for append: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat CSV: %w", err)
	}
	writer := csv.NewWriter(file)
	if info.Size() == 0 {
		if err := writer.Write(header); err != nil {
			return fmt.Errorf("write CSV header: %w", err)
		}
	}
	if err := writer.Write(snapshot.row()); err != nil {
		return fmt.Errorf("write CSV row: %w", err)
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush CSV: %w", err)
	}
	return nil
}

func (snapshot Snapshot) row() []string {
	return []string{
		"1",
		snapshot.ObservedAtUTC,
		snapshot.APITimestampUTC,
		snapshot.Source,
		snapshot.QuotaID,
		snapshot.CopilotPlan,
		snapshot.QuotaResetDateUTC,
		snapshot.CreditsUsed,
		formatNumber(snapshot.DerivedUsed),
		formatNumber(snapshot.Entitlement),
		formatNumber(snapshot.QuotaRemaining),
		formatNumber(snapshot.Remaining),
		formatNumber(snapshot.PercentRemaining),
		strconv.FormatBool(snapshot.OveragePermitted),
		formatNumber(snapshot.OverageCount),
		strconv.FormatBool(snapshot.Unlimited),
		strconv.FormatBool(snapshot.TokenBasedBilling),
	}
}

func formatNumber(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
