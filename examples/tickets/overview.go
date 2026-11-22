package tickets

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/Im-oldxu/new-api/common"
	"github.com/Im-oldxu/new-api/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// The existing attachment column can hold at most one MiB.
type Settings struct {
	ID                 int `json:"-" gorm:"primaryKey"`
	AttachmentLimitKiB int `json:"attachment_limit_kib" gorm:"column:attachment_limit_kib;not null"`
}

func (Settings) TableName() string { return "example_ticket_settings" }

func attachmentLimit(c *gin.Context) (int, bool) {
	var settings Settings
	err := model.DB.WithContext(c.Request.Context()).First(&settings, 1).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return maxAttachment / 1024, true
	}
	if err != nil || settings.AttachmentLimitKiB < 1 || settings.AttachmentLimitKiB > maxAttachment/1024 {
		fail(c, 500, "Failed to load")
		return 0, false
	}
	return settings.AttachmentLimitKiB, true
}

func getSettings(c *gin.Context) {
	if limit, ok := attachmentLimit(c); ok {
		common.ApiSuccess(c, Settings{AttachmentLimitKiB: limit})
	}
}

func saveSettings(c *gin.Context) {
	var settings Settings
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 2048)
	if err := common.DecodeJsonBody(c.Request.Body, &settings); err != nil || settings.AttachmentLimitKiB < 1 || settings.AttachmentLimitKiB > maxAttachment/1024 {
		fail(c, 400, "Invalid parameters")
		return
	}
	settings.ID = 1
	err := model.DB.WithContext(c.Request.Context()).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "id"}}, DoUpdates: clause.AssignmentColumns([]string{"attachment_limit_kib"})}).Create(&settings).Error
	if err != nil {
		fail(c, 500, "Update failed")
		return
	}
	common.ApiSuccess(c, settings)
}

func overview(all bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		days, err := strconv.Atoi(c.DefaultQuery("days", "7"))
		if err != nil || (days != 7 && days != 30) {
			fail(c, 400, "Invalid parameters")
			return
		}
		today := time.Now().UTC().Truncate(24 * time.Hour)
		start := today.AddDate(0, 0, 1-days).Unix()
		var groups []struct {
			Status string
			Count  int64
		}
		if err := scope(c, all).Select("status, count(*) as count").Group("status").Scan(&groups).Error; err != nil {
			fail(c, 500, "Failed to load")
			return
		}
		counts := map[string]int64{pending: 0, processing: 0, completed: 0}
		var total int64
		for _, g := range groups {
			counts[g.Status] = g.Count
			total += g.Count
		}
		var buckets []struct {
			Day   int64
			Count int64
		}
		const day = "created_at - (created_at % 86400)"
		if err := scope(c, all).Where("created_at >= ? AND created_at < ?", start, today.AddDate(0, 0, 1).Unix()).Select(day + " AS day, count(*) AS count").Group(day).Order("day").Scan(&buckets).Error; err != nil {
			fail(c, 500, "Failed to load")
			return
		}
		trend := make([]int64, days)
		for _, b := range buckets {
			trend[(b.Day-start)/86400] = b.Count
		}
		var recent []Ticket
		if err := scope(c, all).Omit("attachment", "description").Order("updated_at DESC, id DESC").Limit(5).Find(&recent).Error; err != nil {
			fail(c, 500, "Failed to load")
			return
		}
		common.ApiSuccess(c, gin.H{"total": total, "counts": counts, "start": start, "trend": trend, "recent": recent})
	}
}
