package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/dto"

	"github.com/stretchr/testify/assert"
)

// A low-balance reminder goes out once, on the charge that takes a wallet or
// subscription balance below the threshold. A balance already below it earns
// nothing more until a top-up lifts it back above the line. The site switch
// and the user's switch both turn reminders off; a user who never chose gets
// them.
func TestQuotaReminderFiresOncePerDrop(t *testing.T) {
	previousEnabled, previousThreshold := common.QuotaRemindEnabled, common.QuotaRemindThreshold
	t.Cleanup(func() {
		common.QuotaRemindEnabled, common.QuotaRemindThreshold = previousEnabled, previousThreshold
	})
	common.QuotaRemindThreshold = 1000
	on, off := true, false
	// changes move the balance from start: negative ones are charges, which
	// may remind; positive ones are top-ups. want holds one entry per charge.
	for _, tc := range []struct {
		name        string
		siteEnabled bool
		setting     dto.UserSetting
		start       int64
		changes     []int64
		want        []bool
	}{
		{
			name:        "never chosen: one reminder per drop, again after a top-up",
			siteEnabled: true,
			start:       1500,
			changes:     []int64{-400, -200, -300, +1500, -1200},
			want:        []bool{false, true, false, true},
		},
		{
			name:        "switched on with its own threshold",
			siteEnabled: true,
			setting:     dto.UserSetting{QuotaWarningEnabled: &on, QuotaWarningThreshold: 5000},
			start:       6000,
			changes:     []int64{-500, -600, -2000},
			want:        []bool{false, true, false},
		},
		{
			name:        "a charge that lands exactly on the threshold does not remind",
			siteEnabled: true,
			start:       1200,
			changes:     []int64{-200, -1},
			want:        []bool{false, true},
		},
		{
			name:        "user switched off",
			siteEnabled: true,
			setting:     dto.UserSetting{QuotaWarningEnabled: &off},
			start:       1500,
			changes:     []int64{-600},
			want:        []bool{false},
		},
		{
			name:    "site switched off",
			setting: dto.UserSetting{QuotaWarningEnabled: &on},
			start:   1500,
			changes: []int64{-600},
			want:    []bool{false},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			common.QuotaRemindEnabled = tc.siteEnabled
			got := make([]bool, 0, len(tc.want))
			balance := tc.start
			for _, change := range tc.changes {
				if change < 0 {
					got = append(got, quotaReminderDue(tc.setting, balance, balance+change))
				}
				balance += change
			}
			assert.Equal(t, tc.want, got)
		})
	}
}
