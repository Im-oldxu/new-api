package common

import (
	"crypto/tls"
	//"os"
	//"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
)

var StartTime = time.Now().Unix() // unit: second
var Version = "v0.0.0"            // this hard coding will be replaced automatically when building, no need to manually change
var SystemName = "Admin Workspace"
var Footer = ""
var Logo = ""

var DefaultCollapseSidebar = false // default value of collapse sidebar

// Any options with "Secret", "Token" in its key won't be return by GetOptions

var SessionSecret = uuid.New().String()
var CryptoSecret = uuid.New().String()
var SessionCookieSecure = false

// DefaultLanguage is the DEFAULT_LANGUAGE environment variable: the language of
// server log lines and of text for a reader who states no language: en, zh-CN
// or zh-TW. English when empty.
var DefaultLanguage string
var SessionCookieTrustedURLs []string

const (
	DefaultUserSessionActiveLimit           = 50
	DefaultUserSessionIssuanceLimit         = 100
	DefaultUserSessionIssuanceWindowSeconds = 24 * 60 * 60
	DefaultUserSessionRevokedRetentionDays  = 7
	DefaultUserSessionHourlyAlertThreshold  = 5000
)

var (
	UserSessionActiveLimit           = DefaultUserSessionActiveLimit
	UserSessionIssuanceLimit         = DefaultUserSessionIssuanceLimit
	UserSessionIssuanceWindowSeconds = int64(DefaultUserSessionIssuanceWindowSeconds)
	UserSessionRevokedRetentionDays  = DefaultUserSessionRevokedRetentionDays
	UserSessionHourlyAlertThreshold  = DefaultUserSessionHourlyAlertThreshold
)

var OptionMap map[string]string
var OptionMapRWMutex sync.RWMutex

var ItemsPerPage = 10
var MaxRecentItems = 1000

var PasswordLoginEnabled = true
var PasswordLoginEncryptionEnabled = false
var PasswordRegisterEnabled = true
var EmailVerificationEnabled = false
var GitHubOAuthEnabled = false
var LinuxDOOAuthEnabled = false
var WeChatAuthEnabled = false
var TelegramOAuthEnabled = false
var TurnstileCheckEnabled = false
var RegisterEnabled = true

var EmailDomainRestrictionEnabled = false // 是否启用邮箱域名限制
var EmailAliasRestrictionEnabled = false  // 是否启用邮箱别名限制
var EmailDomainWhitelist = []string{
	"gmail.com",
	"163.com",
	"126.com",
	"qq.com",
	"outlook.com",
	"hotmail.com",
	"icloud.com",
	"yahoo.com",
	"foxmail.com",
}
var EmailLoginAuthServerList = []string{
	"smtp.sendcloud.net",
	"smtp.azurecomm.net",
}

var DebugEnabled bool
var MemoryCacheEnabled bool

var TLSInsecureSkipVerify bool
var InsecureTLSConfig = &tls.Config{InsecureSkipVerify: true}

var SMTPServer = ""
var SMTPPort = 587
var SMTPSSLEnabled = false
var SMTPStartTLSEnabled = false
var SMTPInsecureSkipVerify = false
var SMTPForceAuthLogin = false
var SMTPAccount = ""
var SMTPFrom = ""
var SMTPToken = ""

var GitHubClientId = ""
var GitHubClientSecret = ""
var LinuxDOClientId = ""
var LinuxDOClientSecret = ""
var LinuxDOMinimumTrustLevel = 0

var WeChatServerAddress = ""
var WeChatServerToken = ""
var WeChatAccountQRCodeImageURL = ""

var TurnstileSiteKey = ""
var TurnstileSecretKey = ""

var TelegramBotToken = ""
var TelegramBotName = ""

var RetryTimes = 0

//var RootUserEmail = ""

var IsMasterNode bool

const (
	NodeNameSourceManual   = "manual"
	NodeNameSourceHostname = "hostname"
)

// NodeName 节点名称，优先从 NODE_NAME 环境变量读取，未配置时回退主机名。
// 用于审计日志和后台任务中标识节点身份；多实例部署时建议显式配置稳定 NODE_NAME。
var NodeName = ""

// NodeNameSource records how NodeName was chosen so future instance-management
// reporting can distinguish operator-configured names from automatic fallback.
var NodeNameSource = NodeNameSourceHostname

var NodeNameManuallyConfigured bool


var SyncFrequency int // unit is second

const (
	RequestIdKey         = "X-Oneapi-Request-Id"
	UpstreamRequestIdKey = "X-Upstream-Request-Id"
)

const (
	RoleGuestUser  = 0
	RoleCommonUser = 1
	RoleAdminUser  = 10
	RoleRootUser   = 100
)

func IsValidateRole(role int) bool {
	return role == RoleGuestUser || role == RoleCommonUser || role == RoleAdminUser || role == RoleRootUser
}

var (
	FileUploadPermission    = RoleGuestUser
	FileDownloadPermission  = RoleGuestUser
	ImageUploadPermission   = RoleGuestUser
	ImageDownloadPermission = RoleGuestUser
)

// All duration's unit is seconds
// Shouldn't larger then RateLimitKeyExpirationDuration
var (
	GlobalApiRateLimitEnable   bool
	GlobalApiRateLimitNum      int
	GlobalApiRateLimitDuration int64

	GlobalWebRateLimitEnable   bool
	GlobalWebRateLimitNum      int
	GlobalWebRateLimitDuration int64

	GlobalStaticRateLimitEnable   bool
	GlobalStaticRateLimitNum      int
	GlobalStaticRateLimitDuration int64

	CriticalRateLimitEnable   bool
	CriticalRateLimitNum            = 20
	CriticalRateLimitDuration int64 = 20 * 60

	UploadRateLimitNum            = 10
	UploadRateLimitDuration int64 = 60

	DownloadRateLimitNum            = 10
	DownloadRateLimitDuration int64 = 60

	// Per-user search rate limit (applies after authentication, keyed by user ID)
	SearchRateLimitEnable         = true
	SearchRateLimitNum            = 10
	SearchRateLimitDuration int64 = 60
)

var RateLimitKeyExpirationDuration = 20 * time.Minute

const (
	UserStatusEnabled  = 1 // don't use 0, 0 is the default value!
	UserStatusDisabled = 2 // also don't use 0
)

const (
	RedemptionCodeStatusEnabled  = 1 // don't use 0, 0 is the default value!
	RedemptionCodeStatusDisabled = 2 // also don't use 0
	RedemptionCodeStatusUsed     = 3 // also don't use 0
)

const (
	TopUpStatusPending = "pending"
	TopUpStatusSuccess = "success"
	TopUpStatusFailed  = "failed"
	TopUpStatusExpired = "expired"
)
