package constant

type ContextKey string

const (
	ContextKeyRequestStartTime ContextKey = "request_start_time"
	ContextKeyUserId           ContextKey = "id"
	ContextKeyUserSetting      ContextKey = "user_setting"
	ContextKeyUserStatus       ContextKey = "user_status"
	ContextKeyUserEmail        ContextKey = "user_email"
	ContextKeyUserGroup        ContextKey = "user_group"
	ContextKeyUserName         ContextKey = "username"
	ContextKeyLanguage         ContextKey = "language"
	ContextKeyAuditLogged      ContextKey = "audit_logged"
)
