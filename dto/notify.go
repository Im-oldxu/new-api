package dto

const (
	NotifyTypeEmail   = "email"
	NotifyTypeWebhook = "webhook"
	NotifyTypeBark    = "bark"
	NotifyTypeGotify  = "gotify"
	ContentValueParam = "{{value}}"
)

type Notify struct {
	Type    string `json:"type"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Values  []any  `json:"values"`
}
