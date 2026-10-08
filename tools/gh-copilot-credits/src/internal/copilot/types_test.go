package copilot

import (
	"strings"
	"testing"
)

func TestDecodeUserDataAndSelectPremiumQuota(t *testing.T) {
	raw := `{
		"copilot_plan":"business",
		"quota_reset_date_utc":"2026-10-01T00:00:00.000Z",
		"token_based_billing":true,
		"quota_snapshots":{
			"chat":{"quota_id":"chat","unlimited":true,"credits_used":0},
			"premium_interactions":{
				"quota_id":"premium_interactions",
				"timestamp_utc":"2026-09-26T13:35:02.302Z",
				"entitlement":20000,
				"quota_remaining":2272,
				"remaining":2272,
				"percent_remaining":11.36,
				"credits_used":17728,
				"overage_permitted":true,
				"overage_count":0,
				"unlimited":false,
				"token_based_billing":true
			}
		}
	}`

	data, err := DecodeUserData(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("DecodeUserData() error = %v", err)
	}
	quota, err := data.PremiumQuota()
	if err != nil {
		t.Fatalf("PremiumQuota() error = %v", err)
	}
	if quota.QuotaID != "premium_interactions" {
		t.Fatalf("QuotaID = %q, want premium_interactions", quota.QuotaID)
	}
	if quota.CreditsUsed == nil || *quota.CreditsUsed != 17728 {
		t.Fatalf("CreditsUsed = %v, want 17728", quota.CreditsUsed)
	}
	if got := quota.DerivedUsedCredits(); got != 17728 {
		t.Fatalf("DerivedUsedCredits() = %v, want 17728", got)
	}
	if data.CopilotPlan != "business" {
		t.Fatalf("CopilotPlan = %q, want business", data.CopilotPlan)
	}
}

func TestPremiumQuotaValidation(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "missing quota",
			raw:  `{"quota_snapshots":{"chat":{"quota_id":"chat"}}}`,
			want: "premium_interactions",
		},
		{
			name: "unlimited quota",
			raw:  `{"quota_snapshots":{"premium_interactions":{"quota_id":"premium_interactions","unlimited":true}}}`,
			want: "unlimited",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data, err := DecodeUserData(strings.NewReader(tc.raw))
			if err != nil {
				t.Fatalf("DecodeUserData() error = %v", err)
			}
			_, err = data.PremiumQuota()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("PremiumQuota() error = %v, want error containing %q", err, tc.want)
			}
		})
	}
}

func TestQuotaDerivedUsedHandlesOverage(t *testing.T) {
	quota := QuotaSnapshot{
		Entitlement:    20000,
		QuotaRemaining: 0,
		Remaining:      -125.5,
	}
	if got := quota.DerivedUsedCredits(); got != 20000 {
		t.Fatalf("DerivedUsedCredits() = %v, want 20000", got)
	}
	if got := quota.OverageCredits(); got != 125.5 {
		t.Fatalf("OverageCredits() = %v, want 125.5", got)
	}
}
