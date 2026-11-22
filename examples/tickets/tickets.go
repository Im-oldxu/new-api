// Package tickets is an optional support-ticket example.
package tickets

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Im-oldxu/new-api/common"
	"github.com/Im-oldxu/new-api/constant"
	"github.com/Im-oldxu/new-api/middleware"
	"github.com/Im-oldxu/new-api/model"
	"github.com/Im-oldxu/new-api/service/authz"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const maxAttachment = 1 << 20
const pending = "pending"
const processing = "in_progress"
const completed = "completed"

type Ticket struct {
	ID             int    `json:"id" gorm:"primaryKey"`
	AssigneeID     *int   `json:"assignee_id" gorm:"index"`
	OwnerName      string `json:"owner_name" gorm:"-"`
	AssigneeName   string `json:"assignee_name" gorm:"-"`
	OwnerID        int    `json:"owner_id" gorm:"not null;index"`
	Title          string `json:"title" gorm:"size:160;not null"`
	Description    string `json:"description" gorm:"type:text;not null"`
	Status         string `json:"status" gorm:"size:16;not null;index"`
	Version        int    `json:"version" gorm:"not null"`
	AttachmentName string `json:"attachment_name" gorm:"size:200"`
	Attachment     []byte `json:"-" gorm:"size:1048576"`
	CreatedAt      int64  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      int64  `json:"updated_at" gorm:"autoUpdateTime"`
}

func (Ticket) TableName() string { return "example_tickets" }
func Migrate(db *gorm.DB) error  { return db.AutoMigrate(&Ticket{}, &Entry{}, &Settings{}) }

// Entry is the participant-visible timeline. Operational audit metadata stays in the audit log.
type Entry struct {
	ID        int    `json:"id" gorm:"primaryKey"`
	TicketID  int    `json:"-" gorm:"not null;index"`
	ActorID   int    `json:"actor_id" gorm:"not null"`
	ActorName string `json:"actor_name" gorm:"size:64"`
	Action    string `json:"action" gorm:"size:16;not null"`
	Body      string `json:"body" gorm:"type:text"`
	CreatedAt int64  `json:"created_at" gorm:"autoCreateTime"`
}

func (Entry) TableName() string { return "example_ticket_entries" }

func loadPeople(db *gorm.DB, rows []Ticket) error {
	ids := map[int]bool{}
	for _, row := range rows {
		ids[row.OwnerID] = true
		if row.AssigneeID != nil {
			ids[*row.AssigneeID] = true
		}
	}
	if len(ids) == 0 {
		return nil
	}
	keys := make([]int, 0, len(ids))
	for id := range ids {
		keys = append(keys, id)
	}
	var users []model.User
	if err := db.Select("id", "username", "display_name").Where("id IN ?", keys).Find(&users).Error; err != nil {
		return err
	}
	names := map[int]string{}
	for _, user := range users {
		name := strings.TrimSpace(user.DisplayName)
		if name == "" {
			name = user.Username
		}
		names[user.Id] = name
	}
	for i := range rows {
		rows[i].OwnerName = names[rows[i].OwnerID]
		if rows[i].AssigneeID != nil {
			rows[i].AssigneeName = names[*rows[i].AssigneeID]
		}
	}
	return nil
}

func init() {
	authz.RegisterResource(authz.ResourceDefinition{Resource: "ticket", LabelKey: "Tickets", Actions: []authz.ActionDefinition{
		{Action: authz.ActionRead, LabelKey: "View", DescriptionKey: "View all tickets and attachments."},
		{Action: authz.ActionWrite, LabelKey: "Edit", DescriptionKey: "Process tickets."},
	}})
}

func Register(api *gin.RouterGroup) {
	middleware.DeclareSessionRoute(http.MethodPut, api.BasePath()+"/tickets/settings")
	api.PUT("/tickets/settings", middleware.DisableCache(), middleware.RootAuth(), saveSettings)
	personal := api.Group("/tickets", middleware.DisableCache(), middleware.UserAuth())
	for _, r := range []struct {
		method, path string
		handler      gin.HandlerFunc
	}{
		{http.MethodGet, "/overview", overview(false)}, {http.MethodGet, "/settings", getSettings},
		{http.MethodGet, "", list(false)}, {http.MethodPost, "", create},
		{http.MethodGet, "/:id", detail(false, false)}, {http.MethodPut, "/:id", edit},
		{http.MethodGet, "/:id/attachment", detail(false, true)},
		{http.MethodPost, "/:id/replies", reply(false)},
	} {
		middleware.DeclareSessionRoute(r.method, personal.BasePath()+r.path)
		personal.Handle(r.method, r.path, r.handler)
	}
	management := api.Group("/manage/tickets", middleware.DisableCache(), middleware.AdminAuth())
	management.Use(middleware.RequirePermission(authz.Permission{Resource: "ticket", Action: authz.ActionRead}))
	for _, r := range []struct {
		method, path, action string
		handler              gin.HandlerFunc
	}{
		{http.MethodGet, "/overview", authz.ActionRead, overview(true)},
		{http.MethodGet, "", authz.ActionRead, list(true)},
		{http.MethodGet, "/:id", authz.ActionRead, detail(true, false)},
		{http.MethodGet, "/:id/attachment", authz.ActionRead, detail(true, true)},
		{http.MethodPatch, "/:id/status", authz.ActionWrite, advance},
		{http.MethodPost, "/:id/replies", authz.ActionWrite, reply(true)},
	} {
		permission := authz.Permission{Resource: "ticket", Action: r.action}
		middleware.DeclareAccessTokenPermissionRoute(r.method, management.BasePath()+r.path, permission)
		if r.action == authz.ActionWrite {
			management.Handle(r.method, r.path, middleware.RequirePermission(permission), r.handler)
		} else {
			management.Handle(r.method, r.path, r.handler)
		}
	}
}

func fail(c *gin.Context, status int, message string) {
	common.ApiErrorStatus(c, status, common.NewMessage(message))
}

func scope(c *gin.Context, all bool) *gorm.DB {
	q := model.DB.WithContext(c.Request.Context()).Model(&Ticket{})
	if !all {
		q = q.Where("owner_id = ?", c.GetInt("id"))
	}
	return q
}

func list(all bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		p := common.GetPageQuery(c)
		if p.Page < 1 || p.Page > 1000000 || p.PageSize < 1 {
			fail(c, 400, "Invalid parameters")
			return
		}
		q := scope(c, all)
		if state := c.Query("status"); state != "" {
			if state != pending && state != processing && state != completed {
				fail(c, 400, "Invalid parameters")
				return
			}
			q = q.Where("status = ?", state)
		}
		switch c.Query("assignment") {
		case "":
		case "unassigned":
			q = q.Where("assignee_id IS NULL AND status <> ?", completed)
		case "mine":
			q = q.Where("assignee_id = ? AND status = ?", c.GetInt("id"), processing)
		default:
			fail(c, 400, "Invalid parameters")
			return
		}
		if search := strings.TrimSpace(c.Query("search")); search != "" {
			if utf8.RuneCountInString(search) > 160 {
				fail(c, 400, "Invalid parameters")
				return
			}
			q = q.Where("title LIKE ?", "%"+search+"%")
		}
		var total int64
		var rows []Ticket
		if err := q.Count(&total).Error; err != nil {
			fail(c, 500, "Failed to load")
			return
		}
		if err := q.Omit("attachment", "description").Order("id DESC").Offset(p.GetStartIdx()).Limit(p.PageSize).Find(&rows).Error; err != nil {
			fail(c, 500, "Failed to load")
			return
		}
		if err := loadPeople(model.DB.WithContext(c.Request.Context()), rows); err != nil {
			fail(c, 500, "Failed to load")
			return
		}
		p.SetTotal(int(total))
		p.SetItems(rows)
		common.ApiSuccess(c, p)
	}
}

func get(c *gin.Context, all, withFile bool) (Ticket, bool) {
	var row Ticket
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		fail(c, 400, "Invalid parameters")
		return row, false
	}
	q := scope(c, all)
	if !withFile {
		q = q.Omit("attachment")
	}
	err = q.First(&row, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		fail(c, 404, "Not found")
		return row, false
	}
	if err != nil {
		fail(c, 500, "Failed to load")
		return row, false
	}
	return row, true
}

func detail(all, withFile bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		row, ok := get(c, all, withFile)
		if !ok {
			return
		}
		if !withFile {
			rows := []Ticket{row}
			db := model.DB.WithContext(c.Request.Context())
			var entries []Entry
			if err := loadPeople(db, rows); err != nil {
				fail(c, 500, "Failed to load")
				return
			}
			if err := db.Where("ticket_id = ?", row.ID).Order("id").Find(&entries).Error; err != nil {
				fail(c, 500, "Failed to load")
				return
			}
			common.ApiSuccess(c, struct {
				Ticket
				Entries []Entry `json:"entries"`
			}{rows[0], entries})
			return
		}
		if row.AttachmentName == "" {
			fail(c, 404, "Not found")
			return
		}
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": row.AttachmentName}))
		c.Data(http.StatusOK, "application/octet-stream", row.Attachment)
	}
}

func readForm(c *gin.Context) (Ticket, bool) {
	var row Ticket
	limit, ok := attachmentLimit(c)
	if !ok {
		return row, false
	}
	maxBytes := int64(limit) * 1024
	maxBody := maxBytes + 64<<10
	if c.Request.ContentLength > maxBody {
		fail(c, 413, "Attachment is too large")
		return row, false
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBody)
	form, err := common.ParseMultipartFormReusable(c)
	if err != nil {
		fail(c, 400, "Invalid parameters")
		return row, false
	}
	fields := url.Values(form.Value)
	row.Title = strings.TrimSpace(fields.Get("title"))
	row.Description = strings.TrimSpace(fields.Get("description"))
	row.Version, _ = strconv.Atoi(fields.Get("version"))
	if row.Title == "" || utf8.RuneCountInString(row.Title) > 160 || row.Description == "" || utf8.RuneCountInString(row.Description) > 4000 {
		fail(c, 400, "Invalid parameters")
		return row, false
	}
	files := form.File["attachment"]
	if len(form.File) > 1 || len(files) > 1 || (len(form.File) > 0 && len(files) == 0) {
		fail(c, 400, "Invalid parameters")
		return row, false
	}
	if len(files) == 0 {
		return row, true
	}
	file := files[0]
	name := path.Base(strings.ReplaceAll(file.Filename, "\\", "/"))
	if name == "." || name == ".." || name == "/" || name == "" || utf8.RuneCountInString(name) > 200 || strings.IndexFunc(name, unicode.IsControl) >= 0 {
		fail(c, 400, "Invalid parameters")
		return row, false
	}
	if file.Size > maxBytes {
		fail(c, 413, "Attachment is too large")
		return row, false
	}
	f, err := file.Open()
	if err != nil {
		fail(c, 400, "Invalid parameters")
		return row, false
	}
	defer f.Close()
	row.Attachment, err = io.ReadAll(io.LimitReader(f, maxBytes+1))
	if err != nil {
		fail(c, 400, "Invalid parameters")
		return row, false
	}
	if int64(len(row.Attachment)) > maxBytes {
		fail(c, 413, "Attachment is too large")
		return row, false
	}
	row.AttachmentName = name
	return row, true
}

func audit(c *gin.Context, row Ticket, action string) {
	model.RecordAuditLog(c, model.AuditLog{
		UserId: c.GetInt("id"), ActorRole: c.GetInt("role"), Username: c.GetString("username"),
		Category: model.AuditCategoryOperation, Action: action, Success: true,
		Content: "Ticket #" + strconv.Itoa(row.ID) + " · " + action + " · " + row.Status,
		Other: model.AuditOther{Op: &model.AuditOperation{
			Action: action, Params: map[string]any{"ticket_id": row.ID, "status": row.Status},
		}},
	})
	common.SetContextKey(c, constant.ContextKeyAuditLogged, true)
}

func create(c *gin.Context) {
	row, ok := readForm(c)
	if !ok {
		return
	}
	row.OwnerID = c.GetInt("id")
	row.Status = pending
	row.Version = 1
	if err := model.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return tx.Create(&Entry{TicketID: row.ID, ActorID: row.OwnerID, ActorName: c.GetString("username"), Action: "created"}).Error
	}); err != nil {
		fail(c, 500, "Create failed")
		return
	}
	audit(c, row, "ticket.create")
	common.ApiSuccess(c, row)
}

func edit(c *gin.Context) {
	current, ok := get(c, false, false)
	if !ok {
		return
	}
	row, ok := readForm(c)
	if !ok {
		return
	}
	if row.Version < 1 {
		fail(c, 400, "Invalid parameters")
		return
	}
	updates := map[string]any{"title": row.Title, "description": row.Description}
	if row.AttachmentName != "" {
		updates["attachment"] = row.Attachment
		updates["attachment_name"] = row.AttachmentName
	}
	if current.Status != pending {
		fail(c, 409, "Invalid ticket transition")
		return
	}
	change(c, current, row.Version, updates, "edited", "", "ticket.edit")
}

// The version predicate serializes every timeline entry with its ticket update.
func change(c *gin.Context, row Ticket, version int, updates map[string]any, action, body, auditAction string) {
	updates["version"] = gorm.Expr("version + 1")
	err := model.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&Ticket{}).Where("id = ? AND version = ? AND status = ?", row.ID, version, row.Status).Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return gorm.ErrRecordNotFound
		}
		return tx.Create(&Entry{TicketID: row.ID, ActorID: c.GetInt("id"), ActorName: c.GetString("username"), Action: action, Body: body}).Error
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		fail(c, 409, "Ticket has changed")
		return
	}
	if err != nil {
		fail(c, 500, "Update failed")
		return
	}
	if status, ok := updates["status"].(string); ok {
		row.Status = status
	}
	audit(c, row, auditAction)
	common.ApiSuccess(c, nil)
}

func canHandle(c *gin.Context, row Ticket) bool {
	return row.AssigneeID != nil && (*row.AssigneeID == c.GetInt("id") || c.GetInt("role") == common.RoleRootUser)
}

func reply(all bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		row, ok := get(c, all, false)
		if !ok {
			return
		}
		if row.Status == completed {
			fail(c, 409, "Invalid ticket transition")
			return
		}
		if all && !canHandle(c, row) {
			fail(c, 403, "Permission denied")
			return
		}
		var req struct {
			Body    string `json:"body"`
			Version int    `json:"version"`
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 32<<10)
		if err := common.DecodeJsonBody(c.Request.Body, &req); err != nil {
			fail(c, 400, "Invalid parameters")
			return
		}
		body := strings.TrimSpace(req.Body)
		if req.Version < 1 || body == "" || utf8.RuneCountInString(body) > 4000 {
			fail(c, 400, "Invalid parameters")
			return
		}
		change(c, row, req.Version, map[string]any{}, "reply", body, "ticket.reply")
	}
}

func advance(c *gin.Context) {
	row, ok := get(c, true, false)
	if !ok {
		return
	}
	var req struct {
		Status  string `json:"status"`
		Version int    `json:"version"`
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
	if err := common.DecodeJsonBody(c.Request.Body, &req); err != nil || req.Version < 1 {
		fail(c, 400, "Invalid parameters")
		return
	}
	switch {
	case req.Status == processing && (row.Status == pending || row.Status == processing) && row.AssigneeID == nil:
		change(c, row, req.Version, map[string]any{"status": processing, "assignee_id": c.GetInt("id")}, "claimed", "", "ticket.claim")
	case row.Status == processing && req.Status == completed:
		if !canHandle(c, row) {
			fail(c, 403, "Permission denied")
			return
		}
		change(c, row, req.Version, map[string]any{"status": completed}, "completed", "", "ticket.complete")
	default:
		fail(c, 409, "Invalid ticket transition")
	}
}
