package copilot

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const PremiumInteractionsQuotaID = "premium_interactions"

// UserData contains only the account and quota fields needed by this tool.
// The endpoint returns additional account metadata; it is intentionally not
// represented here so callers cannot accidentally persist it.
type UserData struct {
	CopilotPlan       string                   `json:"copilot_plan"`
	QuotaResetDateUTC string                   `json:"quota_reset_date_utc"`
	TokenBasedBilling bool                     `json:"token_based_billing"`
	QuotaSnapshots    map[string]QuotaSnapshot `json:"quota_snapshots"`
}

type QuotaSnapshot struct {
	QuotaID           string   `json:"quota_id"`
	TimestampUTC      string   `json:"timestamp_utc"`
	Entitlement       float64  `json:"entitlement"`
	QuotaRemaining    float64  `json:"quota_remaining"`
	Remaining         float64  `json:"remaining"`
	PercentRemaining  float64  `json:"percent_remaining"`
	Unlimited         bool     `json:"unlimited"`
	OveragePermitted  bool     `json:"overage_permitted"`
	OverageCount      float64  `json:"overage_count"`
	HasQuota          bool     `json:"has_quota"`
	QuotaResetAt      float64  `json:"quota_reset_at"`
	TokenBasedBilling bool     `json:"token_based_billing"`
	CreditsUsed       *float64 `json:"credits_used"`
}

func DecodeUserData(reader io.Reader) (UserData, error) {
	var data UserData
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&data); err != nil {
		return UserData{}, fmt.Errorf("decode Copilot response: %w", err)
	}
	if data.QuotaSnapshots == nil {
		return UserData{}, errors.New("Copilot response has no quota_snapshots")
	}
	return data, nil
}

func (data UserData) PremiumQuota() (QuotaSnapshot, error) {
	quota, ok := data.QuotaSnapshots[PremiumInteractionsQuotaID]
	if !ok {
		return QuotaSnapshot{}, fmt.Errorf("Copilot response has no %q quota", PremiumInteractionsQuotaID)
	}
	if quota.Unlimited {
		return QuotaSnapshot{}, errors.New("premium_interactions quota is unlimited")
	}
	if quota.QuotaID == "" {
		quota.QuotaID = PremiumInteractionsQuotaID
	}
	if quota.QuotaID != PremiumInteractionsQuotaID {
		return QuotaSnapshot{}, fmt.Errorf("premium quota has unexpected quota_id %q", quota.QuotaID)
	}
	if quota.Entitlement <= 0 {
		return QuotaSnapshot{}, errors.New("premium_interactions quota has invalid entitlement")
	}
	return quota, nil
}

func (quota QuotaSnapshot) DerivedUsedCredits() float64 {
	return quota.Entitlement - quota.QuotaRemaining
}

func (quota QuotaSnapshot) OverageCredits() float64 {
	if quota.Remaining >= 0 {
		return 0
	}
	return -quota.Remaining
}
