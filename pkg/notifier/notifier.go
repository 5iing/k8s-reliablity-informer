package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Notifier interface {
	Notifiy(message string) error
}

type DiscordNotifier struct {
	webhookURL string
	client     *http.Client
}

func NewDiscord(webhookURL string) *DiscordNotifier {
	return &DiscordNotifier{
		webhookURL: webhookURL,
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

func (d *DiscordNotifier) Notifiy(message string) error {
	payload := map[string]interface{}{
		"content": message,
	}
	
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal discord payload: %w", err)
	}
	
	resp, err := d.client.Post(d.webhookURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to send discord webhook: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("discord webhook failed with status %d", resp.StatusCode)
	}
	
	return nil
}

type ConsoleNotifier struct{}

func NewConsole() *ConsoleNotifier {
	return &ConsoleNotifier{}
}

func (c *ConsoleNotifier) Notifiy(message string) error {
	fmt.Println("[CONSOLE]", message)
	return nil
}