package notifier

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestConsoleNotifier_Notifiy(t *testing.T) {
	tests := []struct {
		name    string
		message string
		wantErr bool
	}{
		{
			name:    "simple message",
			message: "test message",
			wantErr: false,
		},
		{
			name:    "empty message",
			message: "",
			wantErr: false,
		},
		{
			name:    "message with special characters",
			message: "테스트 메시지 with special chars: !@#$%",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewConsole()
			err := c.Notifiy(tt.message)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConsoleNotifier.Notifiy() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDiscordNotifier_Notifiy(t *testing.T) {
	tests := []struct {
		name           string
		message        string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		errContains    string
	}{
		{
			name:    "successful notification",
			message: "test message",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				// Verify request method
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST request, got %s", r.Method)
				}

				// Verify content type
				contentType := r.Header.Get("Content-Type")
				if contentType != "application/json" {
					t.Errorf("Expected Content-Type application/json, got %s", contentType)
				}

				// Verify payload
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Errorf("Failed to read request body: %v", err)
				}

				var payload map[string]interface{}
				if err := json.Unmarshal(body, &payload); err != nil {
					t.Errorf("Failed to unmarshal payload: %v", err)
				}

				if payload["content"] != "test message" {
					t.Errorf("Expected content 'test message', got %v", payload["content"])
				}

				w.WriteHeader(http.StatusOK)
			},
			wantErr: false,
		},
		{
			name:    "discord returns 400 error",
			message: "test message",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			},
			wantErr:     true,
			errContains: "discord webhook failed with status 400",
		},
		{
			name:    "discord returns 404 error",
			message: "test message",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantErr:     true,
			errContains: "discord webhook failed with status 404",
		},
		{
			name:    "discord returns 500 error",
			message: "test message",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr:     true,
			errContains: "discord webhook failed with status 500",
		},
		{
			name:    "empty message",
			message: "",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				var payload map[string]interface{}
				json.Unmarshal(body, &payload)

				if payload["content"] != "" {
					t.Errorf("Expected empty content, got %v", payload["content"])
				}

				w.WriteHeader(http.StatusOK)
			},
			wantErr: false,
		},
		{
			name:    "message with special characters",
			message: "테스트 메시지 with special chars: !@#$%",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			d := NewDiscord(server.URL)

			err := d.Notifiy(tt.message)

			if (err != nil) != tt.wantErr {
				t.Errorf("DiscordNotifier.Notifiy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errContains != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("DiscordNotifier.Notifiy() error = %v, should contain %s", err, tt.errContains)
				}
			}
		})
	}
}

func TestDiscordNotifier_Notifiy_NetworkError(t *testing.T) {
	d := NewDiscord("http://invalid-url-that-does-not-exist.local:99999")

	err := d.Notifiy("test message")
	if err == nil {
		t.Error("Expected network error, got nil")
	}

	if !strings.Contains(err.Error(), "failed to send discord webhook") {
		t.Errorf("Expected error message to contain 'failed to send discord webhook', got: %v", err)
	}
}

func TestNewDiscord(t *testing.T) {
	webhookURL := "https://discord.com/api/webhooks/test"
	d := NewDiscord(webhookURL)

	if d == nil {
		t.Fatal("Expected non-nil DiscordNotifier")
	}

	if d.webhookURL != webhookURL {
		t.Errorf("Expected webhookURL %s, got %s", webhookURL, d.webhookURL)
	}

	if d.client == nil {
		t.Error("Expected non-nil HTTP client")
	}

	if d.client.Timeout != 10*1000000000 { 
		t.Errorf("Expected timeout 10s, got %v", d.client.Timeout)
	}
}

func TestNewConsole(t *testing.T) {
	c := NewConsole()

	if c == nil {
		t.Fatal("Expected non-nil ConsoleNotifier")
	}
}
