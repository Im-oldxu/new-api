package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateOptionValueRejectsSystemTaskRetentionBelowOneDay(t *testing.T) {
	for _, value := range []string{"", "0", "-1", "1.5", "invalid"} {
		t.Run(value, func(t *testing.T) {
			assert.Error(t, validateOptionValue("system_task_setting.retention_days", value))
		})
	}
	require.NoError(t, validateOptionValue("system_task_setting.retention_days", "1"))
	require.NoError(t, validateOptionValue("system_task_setting.retention_days", "30"))
}
