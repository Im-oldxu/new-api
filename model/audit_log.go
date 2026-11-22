package model

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/Im-oldxu/new-api/common"
	"github.com/Im-oldxu/new-api/logger"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	AuditCategoryLogin       = "login"
	AuditCategorySecurity    = "security"
	AuditCategoryOperation   = "operation"
	AuditCategoryAccessToken = "access_token"
)

// AuditLog is retained independently of usage logs and their cleanup/TTL policy.
// TokenRef identifies a PAT generation without storing its bearer credential.
type AuditLog struct {
	SummaryOnly bool       `gorm:"-" json:"summary_only,omitempty"`
	Id          int        `json:"id"`
	EventId     string     `json:"event_id" gorm:"type:varchar(64);uniqueIndex;index:idx_audit_time_event,priority:2"`
	UserId      int        `json:"user_id" gorm:"index:idx_audit_user_time,priority:1"`
	Username    string     `json:"username" gorm:"type:varchar(64);index"`
	ActorRole   int        `json:"actor_role"` // Immutable role of the actor when the event began, not the log owner.
	CreatedAt   int64      `json:"created_at" gorm:"type:bigint;index:idx_audit_user_time,priority:2;index:idx_audit_token_time,priority:2;index;index:idx_audit_time_event,priority:1"`
	Category    string     `json:"category" gorm:"type:varchar(24);index"`
	Action      string     `json:"action" gorm:"type:varchar(128)"`
	TokenRef    string     `json:"token_ref" gorm:"type:varchar(64);index:idx_audit_token_time,priority:1"`
	AuthMethod  string     `json:"auth_method" gorm:"type:varchar(24)"`
	Ip          string     `json:"ip" gorm:"type:varchar(64)"`
	UserAgent   string     `json:"user_agent" gorm:"type:varchar(512)"`
	Method      string     `json:"method" gorm:"type:varchar(16)"`
	Route       string     `json:"route" gorm:"type:varchar(255)"`
	Status      int        `json:"status"`
	Success     bool       `json:"success"`
	RequestId   string     `json:"request_id" gorm:"type:varchar(64);index"`
	Content     string     `json:"content" gorm:"type:text"`
	Other       AuditOther `json:"other" gorm:"type:json"`
}

type AuditLogFilter struct {
	SelfView        bool // Server-selected metadata projection, independent of the viewer's actual role.
	UserId          int
	Username        string
	Category        string
	TokenRef        string
	ExcludeTokenRef string
	RequestId       string
	StartTimestamp  int64
	EndTimestamp    int64
	Success         *bool
}

func AccessTokenFingerprint(token string) string {
	// PostgreSQL returns CHAR(32) tokens padded with spaces. Normalize that
	// storage padding so persisted tokens and incoming credentials share a ref.
	token = strings.TrimRight(token, " ")
	if token == "" {
		return ""
	}
	digest := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", digest)
}

// RecordAuditLog captures safe request metadata only; raw URLs, query strings,
// credentials and response/request bodies must never enter this table.
func RecordAuditLog(c *gin.Context, entry AuditLog) {
	ctx := context.Background()
	if c != nil && c.Request != nil {
		ctx = c.Request.Context()
		entry.RequestId = c.GetString(common.RequestIdKey)
		entry.Ip = c.ClientIP()
		entry.UserAgent = c.Request.UserAgent()
		entry.Method = c.Request.Method
		entry.Route = c.FullPath()
		if entry.Status == 0 {
			entry.Status = c.Writer.Status()
		}
		if entry.AuthMethod == "" {
			entry.AuthMethod = "session"
			if c.GetBool("use_access_token") {
				entry.AuthMethod = "access_token"
			}
		}
	}
	if entry.CreatedAt == 0 {
		entry.CreatedAt = common.GetTimestamp()
	}
	if entry.RequestId == "" {
		entry.RequestId = common.NewRequestId()
	}
	if entry.EventId == "" {
		entry.EventId = common.NewRequestId()
	}
	switch entry.ActorRole {
	case common.RoleCommonUser, common.RoleAdminUser, common.RoleRootUser:
	default:
		logger.LogError(ctx, common.LogText("audit actor role unavailable (request_id=%s, actor_role=%d)", entry.RequestId, entry.ActorRole))
		entry.ActorRole = 0 // Unknown actors remain visible to root only.
	}
	if entry.Username == "" {
		entry.Username, _ = GetUsernameById(entry.UserId, false)
	}
	ua := []rune(entry.UserAgent)
	if len(ua) > 512 {
		entry.UserAgent = string(ua[:512])
	}
	if LOG_DB == nil {
		logger.LogError(ctx, common.LogText("audit log write failed (request_id=%s): log database unavailable", entry.RequestId))
		return
	}
	var row any = &entry
	if common.UsingLogDatabase(common.DatabaseTypeClickHouse) {
		encoded, err := common.Marshal(entry.Other)
		if err != nil {
			logger.LogError(ctx, common.LogText("audit log write failed (request_id=%s): %v", entry.RequestId, err))
			return
		}
		// The ClickHouse GORM insert callback passes structs to the native
		// driver without resolving their Valuer. Bind this column's JSON
		// encoding while retaining AuditOther in the domain and API models.
		row = &struct {
			AuditLog     `gorm:"embedded"`
			EncodedOther string `gorm:"column:other;type:json"`
		}{AuditLog: entry, EncodedOther: string(encoded)}
	}
	if err := LOG_DB.Table("audit_logs").Create(row).Error; err != nil {
		logger.LogError(ctx, common.LogText("audit log write failed (request_id=%s): %v", entry.RequestId, err))
	}
}

func GetAuditLogs(filter AuditLogFilter, start, limit, viewerRole int) ([]*AuditLog, int64, error) {
	if viewerRole < common.RoleAdminUser {
		return nil, 0, errors.New("audit access requires an administrator")
	}
	query := LOG_DB.Model(&AuditLog{})
	if common.UsingLogDatabase(common.DatabaseTypeClickHouse) {
		// Decode native JSON through database/sql as text for AuditOther.Scan.
		// Preserve numeric metadata instead of returning quoted Int64 values.
		query = query.WithContext(clickhouse.Context(query.Statement.Context, clickhouse.WithSettings(clickhouse.Settings{
			"output_format_native_write_json_as_string": 1,
			"output_format_json_quote_64bit_integers":   0,
		})))
	}
	// Private identifiers must not reveal privileged events through filtered totals.
	if viewerRole < common.RoleRootUser && (filter.UserId > 0 || filter.Username != "" || filter.TokenRef != "" || filter.ExcludeTokenRef != "" || filter.RequestId != "") {
		query = query.Where("actor_role IN ?", []int{common.RoleCommonUser, common.RoleAdminUser})
	}
	if filter.UserId > 0 {
		query = query.Where("user_id = ?", filter.UserId)
	}
	if filter.Username != "" {
		query = query.Where("username = ?", filter.Username)
	}
	if filter.Category != "" {
		query = query.Where("category = ?", filter.Category)
	}
	if filter.TokenRef != "" {
		query = query.Where("token_ref = ?", filter.TokenRef)
	}
	if filter.ExcludeTokenRef != "" {
		query = query.Where("token_ref <> ?", filter.ExcludeTokenRef)
	}
	if filter.RequestId != "" {
		query = query.Where("request_id = ?", filter.RequestId)
	}
	if filter.StartTimestamp > 0 {
		query = query.Where("created_at >= ?", filter.StartTimestamp)
	}
	if filter.EndTimestamp > 0 {
		query = query.Where("created_at <= ?", filter.EndTimestamp)
	}
	if filter.Success != nil {
		query = query.Where("success = ?", *filter.Success)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	logs := make([]*AuditLog, 0)
	if err := query.Order("created_at DESC").Order("event_id DESC").Offset(start).Limit(limit).Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	showRootFields := !filter.SelfView && viewerRole >= common.RoleRootUser
	showAdminFields := !filter.SelfView && viewerRole >= common.RoleAdminUser
	for _, entry := range logs {
		if viewerRole < common.RoleRootUser && entry.ActorRole != common.RoleCommonUser && entry.ActorRole != common.RoleAdminUser {
			// Root and unclassified historical actions expose only a summary.
			*entry = AuditLog{SummaryOnly: true, EventId: entry.EventId, CreatedAt: entry.CreatedAt, Category: entry.Category, Success: entry.Success, ActorRole: entry.ActorRole}
			continue
		}
		if !showRootFields {
			entry.Other.RootInfo = nil
		}
		if !showAdminFields {
			entry.Other.AdminInfo = nil
			entry.Other.AuditInfo = nil
		}
	}
	return logs, total, nil
}

// UserAccessTokenStatus describes the legacy users.access_token credential.
//
// Deprecated: 旧版面板访问令牌，仅在升级后的过渡期内使用；删除 users.access_token 列时一并移除。
type UserAccessTokenStatus struct {
	Exists     bool   `json:"exists"`
	TokenHint  string `json:"token_hint"`
	TokenRef   string `json:"token_ref"`
	CreatedAt  *int64 `json:"created_at"`
	LastUsedAt *int64 `json:"last_used_at"`
	LastUsedIp string `json:"last_used_ip"`
}

// GetUserAccessTokenStatus reads last use of the legacy token from the access audit.
//
// Deprecated: 旧版面板访问令牌，仅在升级后的过渡期内使用；删除 users.access_token 列时一并移除。
func GetUserAccessTokenStatus(userId int) (*UserAccessTokenStatus, error) {
	var user User
	if err := DB.Select("id", "role", "access_token", "access_token_created_at").First(&user, userId).Error; err != nil {
		return nil, err
	}
	status := &UserAccessTokenStatus{Exists: user.GetAccessToken() != ""}
	if !status.Exists {
		return status, nil
	}
	status.TokenRef = AccessTokenFingerprint(user.GetAccessToken())
	status.TokenHint = AccessTokenHint(user.GetAccessToken())
	status.CreatedAt = user.AccessTokenCreatedAt
	var latest AuditLog
	query := LOG_DB.Select("created_at", "ip").Where("user_id = ? AND token_ref = ? AND category = ?", userId, status.TokenRef, AuditCategoryAccessToken)
	if user.Role < common.RoleRootUser {
		query = query.Where("actor_role IN ?", []int{common.RoleCommonUser, common.RoleAdminUser})
	}
	err := query.Order("created_at DESC").Order("event_id DESC").Take(&latest).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return status, nil
	}
	if err != nil {
		return nil, err
	}
	status.LastUsedAt = &latest.CreatedAt
	status.LastUsedIp = latest.Ip
	return status, nil
}

// MigrateAuditLogs also supports independently configured ClickHouse log stores.
// No TTL clause or usage-log cleanup integration is intentional.
func MigrateAuditLogs() error {
	if !common.UsingLogDatabase(common.DatabaseTypeClickHouse) {
		return LOG_DB.AutoMigrate(&AuditLog{})
	}
	return LOG_DB.Exec(`CREATE TABLE IF NOT EXISTS audit_logs (
		id Int64 DEFAULT 0, event_id String, user_id Int64, username String, actor_role Int32,
		created_at Int64, category String, action String, token_ref String,
		auth_method String, ip String, user_agent String, method String, route String,
		status Int32, success UInt8, request_id String, content String, other JSON
	) ENGINE = MergeTree()
	PARTITION BY toYYYYMM(toDateTime(created_at))
	ORDER BY (created_at, event_id)`).Error
}

func ValidAuditCategory(category string) bool {
	return category == "" || category == AuditCategoryLogin || category == AuditCategorySecurity || category == AuditCategoryOperation || category == AuditCategoryAccessToken
}

func ValidTokenFingerprint(value string) bool {
	if value == "" {
		return true
	}
	if len(value) != 64 {
		return false
	}
	return strings.Trim(value, "0123456789abcdef") == ""
}

// RecordLoginLog writes new login events to the independent audit table.
// username 由调用方传入（登录流程已持有用户对象），避免额外的数据库查询。
// content 为英文兜底文本（用于导出）；action+params 供前端本地化渲染。
// other 包含 login_method、user_agent 等结构化信息。
func RecordLoginLog(userId, actorRole int, username string, content string, ip string, action string, params map[string]any, other AuditOther, request ...*gin.Context) {
	other.Op = &AuditOperation{Action: action, Params: params}
	var c *gin.Context
	if len(request) > 0 {
		c = request[0]
	}
	RecordAuditLog(c, AuditLog{UserId: userId, Username: username, ActorRole: actorRole, Category: AuditCategoryLogin, Action: action, Content: content, Ip: ip, Other: other, Success: true})
}

// RecordOperationAuditLog writes new operation/security events to the audit table.
// logUserId 为日志归属者，管理审计日志应归属实际操作者；目标资源/用户放入
// action params。username 内部按 logUserId 查询。content 为英文兜底文本（供导出使用）。
// action+params 写入 Other.op，供前端本地化渲染（普通用户可见，不含敏感信息）。
// adminInfo 存放操作者身份（写入 Other.admin_info，普通用户查询时剥离）；
// auditInfo 存放路由/方法/结果等中间件兜底信息（写入 Other.audit_info，普通用户查询时剥离）。
func RecordOperationAuditLog(logUserId, actorRole int, content string, ip string, action string, params map[string]any, adminInfo *AuditAdminInfo, auditInfo *AuditRequestInfo, request ...*gin.Context) {
	username, _ := GetUsernameById(logUserId, false)
	other := AuditOther{
		Op:        &AuditOperation{Action: action, Params: params},
		AdminInfo: adminInfo,
		AuditInfo: auditInfo,
	}
	var c *gin.Context
	if len(request) > 0 {
		c = request[0]
	}
	category := AuditCategoryOperation
	if adminInfo == nil {
		category = AuditCategorySecurity
	}
	status, success := 200, true
	if auditInfo != nil {
		status = auditInfo.Status
		success = auditInfo.Success
	}
	RecordAuditLog(c, AuditLog{UserId: logUserId, Username: username, ActorRole: actorRole, Category: category, Action: action, Content: content, Ip: ip, Status: status, Success: success, Other: other})
}
