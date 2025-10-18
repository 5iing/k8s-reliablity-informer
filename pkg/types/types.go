package types

import "fmt"

// alert struct
type Alert struct {
	Level    string `json:"level"`
	Resource string `json:"resource"`
	Name     string `json:"name"`
	Message  string `json:"message"`
}

// alert level
const (
	AlertLevelCritical = "critical"
	AlertLevelError    = "error"
	AlertLevelWarning  = "warning"
	AlertLevelInfo     = "info"
)

// resource type
const (
	ResourceTypePod        = "pod"
	ResourceTypeNode       = "node"
	ResourceTypeDeployment = "deployment"
	ResourceTypeService    = "service"
)

func (a *Alert) GetEmoji() string {
	emojiMap := map[string]string{
		AlertLevelWarning:  "⚠️",
		AlertLevelError:    "❌",
		AlertLevelCritical: "🚨",
		AlertLevelInfo:     "ℹ️",
	}
	
	if emoji, exists := emojiMap[a.Level]; exists {
		return emoji
	}
	return "📢"
}

func (a *Alert) FormatMessage() string {
	return fmt.Sprintf("%s [%s] %s: %s",
		a.GetEmoji(),
		a.Resource,
		a.Name,
		a.Message,
	)
}
