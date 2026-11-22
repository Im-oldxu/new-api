package constant

var MaxFileDownloadMB int
var MaxRequestBodyMB int

var AnonymousRequestBodyLimitKB int
var NotifyLimitCount int
var NotificationLimitDurationMinute int

// TrustedRedirectDomains is a list of trusted domains for redirect URL validation.
// Domains support subdomain matching (e.g., "example.com" matches "sub.example.com").
var TrustedRedirectDomains []string
