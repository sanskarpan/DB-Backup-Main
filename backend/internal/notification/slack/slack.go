// Package slack provides Slack notification integration
package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sanskarpan/db-backup/internal/notification"
	pkgErrors "github.com/sanskarpan/db-backup/pkg/errors"
)

// SlackNotifier implements Slack webhook notifications.
type SlackNotifier struct {
	webhookURL string
	channel    string
	username   string
	iconEmoji  string
	timeout    time.Duration
	client     *http.Client
}

// Config holds Slack notifier configuration.
type Config struct {
	WebhookURL string
	Channel    string
	Username   string
	IconEmoji  string
	Timeout    time.Duration
}

// NewSlackNotifier creates a new Slack notifier.
func NewSlackNotifier(cfg *Config) (*SlackNotifier, error) {
	if cfg.WebhookURL == "" {
		return nil, pkgErrors.New(pkgErrors.ErrorTypeConfiguration, "Slack webhook URL is required")
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	username := cfg.Username
	if username == "" {
		username = "DB Backup Utility"
	}

	iconEmoji := cfg.IconEmoji
	if iconEmoji == "" {
		iconEmoji = ":floppy_disk:"
	}

	return &SlackNotifier{
		webhookURL: cfg.WebhookURL,
		channel:    cfg.Channel,
		username:   username,
		iconEmoji:  iconEmoji,
		timeout:    timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// Send sends a notification to Slack.
func (s *SlackNotifier) Send(ctx context.Context, notif *notification.Notification) error {
	message := s.buildMessage(notif)

	jsonData, err := json.Marshal(message)
	if err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeOperation, "failed to marshal Slack message")
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeOperation, "failed to create request")
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeNetwork, "failed to send Slack notification")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return pkgErrors.New(pkgErrors.ErrorTypeOperation,
			fmt.Sprintf("Slack API returned status %d", resp.StatusCode))
	}

	return nil
}

// GetType returns the provider type.
func (s *SlackNotifier) GetType() notification.ProviderType {
	return notification.ProviderTypeSlack
}

// ValidateConfig validates the configuration.
func (s *SlackNotifier) ValidateConfig() error {
	if s.webhookURL == "" {
		return pkgErrors.New(pkgErrors.ErrorTypeConfiguration, "webhook URL is required")
	}
	return nil
}

// SlackMessage represents a Slack message payload.
type SlackMessage struct {
	Channel     string            `json:"channel,omitempty"`
	Username    string            `json:"username,omitempty"`
	IconEmoji   string            `json:"icon_emoji,omitempty"`
	Text        string            `json:"text,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
}

// SlackAttachment represents a Slack message attachment.
type SlackAttachment struct {
	Fallback   string       `json:"fallback"`
	Color      string       `json:"color"`
	Title      string       `json:"title,omitempty"`
	Text       string       `json:"text,omitempty"`
	Fields     []SlackField `json:"fields,omitempty"`
	Footer     string       `json:"footer,omitempty"`
	FooterIcon string       `json:"footer_icon,omitempty"`
	Timestamp  int64        `json:"ts,omitempty"`
	ImageURL   string       `json:"image_url,omitempty"`
}

// SlackField represents a field in a Slack attachment.
type SlackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// buildMessage builds a Slack message from a notification.
func (s *SlackNotifier) buildMessage(notif *notification.Notification) *SlackMessage {
	msg := &SlackMessage{
		Channel:   s.channel,
		Username:  s.username,
		IconEmoji: s.iconEmoji,
		Text:      notif.Title,
	}

	// Build attachments
	attachment := SlackAttachment{
		Fallback:   notif.Message,
		Color:      s.getLevelColor(notif.Level),
		Title:      notif.Title,
		Text:       notif.Message,
		Footer:     "DB Backup Utility",
		FooterIcon: "https://i.imgur.com/wWzX9uB.png",
		Timestamp:  notif.Timestamp.Unix(),
	}

	// Add metadata fields
	if len(notif.Metadata) > 0 {
		for key, value := range notif.Metadata {
			attachment.Fields = append(attachment.Fields, SlackField{
				Title: key,
				Value: fmt.Sprintf("%v", value),
				Short: true,
			})
		}
	}

	// Add custom attachments
	for _, attach := range notif.Attachments {
		slackAttach := SlackAttachment{
			Fallback: attach.Text,
			Color:    attach.Color,
			Title:    attach.Title,
			Text:     attach.Text,
			ImageURL: attach.ImageURL,
		}

		for _, field := range attach.Fields {
			slackAttach.Fields = append(slackAttach.Fields, SlackField{
				Title: field.Title,
				Value: field.Value,
				Short: field.Short,
			})
		}

		msg.Attachments = append(msg.Attachments, slackAttach)
	}

	msg.Attachments = append(msg.Attachments, attachment)

	return msg
}

// getLevelColor returns the color for a notification level.
func (s *SlackNotifier) getLevelColor(level notification.NotificationLevel) string {
	switch level {
	case notification.LevelSuccess:
		return "good" // green
	case notification.LevelWarning:
		return "warning" // yellow
	case notification.LevelError:
		return "danger" // red
	case notification.LevelInfo:
		return "#36a64f" // blue
	default:
		return "#808080" // gray
	}
}
