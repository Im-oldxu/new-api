package dto

// UserSetting is the account preference projection used by the host.
// Preference writes patch the original JSON so unknown fields survive.
type UserSetting struct {
	Language          string `json:"language,omitempty"`
	SidebarModules    string `json:"sidebar_modules,omitempty"`
	NotifyType        string `json:"notify_type,omitempty"`
	NotificationEmail string `json:"notification_email,omitempty"`
	WebhookUrl        string `json:"webhook_url,omitempty"`
	WebhookSecret     string `json:"webhook_secret,omitempty"`
	BarkUrl           string `json:"bark_url,omitempty"`
	GotifyUrl         string `json:"gotify_url,omitempty"`
	GotifyToken       string `json:"gotify_token,omitempty"`
	GotifyPriority    int    `json:"gotify_priority"`
}
