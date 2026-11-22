package router

import (
	"path/filepath"
	"testing"

	"github.com/Im-oldxu/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func retirementDatabase(t *testing.T, tables ...any) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "settings.db")))
	require.NoError(t, err)
	connection, err := db.DB()
	require.NoError(t, err)
	previous := model.DB
	model.DB = db
	t.Cleanup(func() {
		model.DB = previous
		require.NoError(t, connection.Close())
	})
	require.NoError(t, db.AutoMigrate(tables...))
	return db
}

func TestRetiredOptionsRejectEntireSettingsWrite(t *testing.T) {
	db := retirementDatabase(t, &model.Option{})
	original := []model.Option{
		{Key: "ModelPrice", Value: " legacy { "},
		{Key: "SystemName", Value: "local-preview"},
	}
	require.NoError(t, db.Create(&original).Error)
	for _, key := range []string{"ModelPrice", "modelprice ", "channel_affinity_setting.rules", "billing_setting.billing_expr", "LogConsumeEnabled", "log_setting.max_range_days", "MaxTokenAutoGroups", "Chats", "DrawingEnabled", "TaskEnabled", "TaskPublicAddress", "QuotaForNewUser", "EpayKey", "StripeApiSecret", "payment_setting.pay_methods", "checkin_setting.enabled"} {
		require.ErrorIs(t, model.UpdateOption(key, "new"), model.ErrRetiredGatewayOption)
		require.ErrorIs(t, model.UpdateOptionsBulk(map[string]string{
			"SystemName": "changed", key: "new",
		}), model.ErrRetiredGatewayOption)
		require.ErrorIs(t, model.UpdateOptionsBulk(map[string]string{
			"ServerAddress": "https://example.test", key: "new",
		}), model.ErrRetiredGatewayOption)
	}
	var stored []model.Option
	require.NoError(t, db.Find(&stored).Error)
	assert.ElementsMatch(t, original, stored)
}

func TestArchivedRecordsSurviveTaskMaintenance(t *testing.T) {
	db := retirementDatabase(t, &model.SystemTask{}, &model.SystemTaskLock{})
	tasks := []model.SystemTask{
		{TaskID: "subscription-pending", Type: "sync_subscription_plans", Status: model.SystemTaskStatusPending},
		{TaskID: "subscription-done", Type: "sync_subscription_plans", Status: model.SystemTaskStatusSucceeded},
		{TaskID: "archived-first", Type: "log_cleanup", Status: model.SystemTaskStatusSucceeded},
		{TaskID: "archived-last", Type: "log_cleanup", Status: model.SystemTaskStatusRunning},
	}
	require.NoError(t, db.Create(&tasks).Error)
	pending, err := model.PendingSyncTaskTypes()
	require.NoError(t, err)
	assert.NotContains(t, pending, "sync_subscription_plans")
	lock := model.SystemTaskLock{Type: "log_cleanup", TaskID: "archived-last", LockedBy: "old-node", LockedUntil: 1}
	require.NoError(t, db.Create(&lock).Error)
	require.NoError(t, model.ExpireStaleSystemTaskLocks(2))
	_, err = model.DeleteFinishedSystemTasks(t.Context(), tasks[0].UpdatedAt+1)
	require.NoError(t, err)
	_, err = model.DeleteSystemTaskHistory(model.SystemTaskFilter{})
	require.NoError(t, err)
	var after []model.SystemTask
	require.NoError(t, db.Order("id").Find(&after).Error)
	assert.Equal(t, tasks, after)
	var storedLock model.SystemTaskLock
	require.NoError(t, db.First(&storedLock).Error)
	assert.Equal(t, lock, storedLock)
}
