package enhanced

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"strings"
	"time"
)

// Channel is the interface that all delivery channels must implement
type Channel interface {
	Send(ctx context.Context, notification *Notification) error
	Name() string
	SupportsRichContent() bool
}

// =============================================================================
// Email Channel
// =============================================================================

// EmailChannel delivers notifications via email
type EmailChannel struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	FromAddress  string
	FromName     string
}

func NewEmailChannel(host string, port int, username, password, fromAddress, fromName string) *EmailChannel {
	return &EmailChannel{
		SMTPHost:     host,
		SMTPPort:     port,
		SMTPUsername: username,
		SMTPPassword: password,
		FromAddress:  fromAddress,
		FromName:     fromName,
	}
}

func (c *EmailChannel) Name() string {
	return string(ChannelEmail)
}

func (c *EmailChannel) SupportsRichContent() bool {
	return true
}

func (c *EmailChannel) Send(ctx context.Context, notification *Notification) error {
	// Get recipient email from metadata
	to, ok := notification.Metadata["email"].(string)
	if !ok {
		return fmt.Errorf("email address not found in metadata")
	}

	// Build email
	subject := notification.Title
	body := c.buildHTMLEmail(notification)

	// Compose message
	msg := c.composeMIMEMessage(to, subject, body)

	// Send via SMTP
	auth := smtp.PlainAuth("", c.SMTPUsername, c.SMTPPassword, c.SMTPHost)
	addr := fmt.Sprintf("%s:%d", c.SMTPHost, c.SMTPPort)

	return smtp.SendMail(addr, auth, c.FromAddress, []string{to}, []byte(msg))
}

func (c *EmailChannel) buildHTMLEmail(notification *Notification) string {
	var html strings.Builder

	html.WriteString(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #007bff; color: white; padding: 20px; border-radius: 5px 5px 0 0; }
        .content { background: #f8f9fa; padding: 20px; }
        .priority-urgent { border-left: 5px solid #dc3545; }
        .priority-high { border-left: 5px solid #fd7e14; }
        .priority-normal { border-left: 5px solid #007bff; }
        .priority-low { border-left: 5px solid #6c757d; }
        .actions { margin-top: 20px; }
        .btn { display: inline-block; padding: 10px 20px; background: #007bff; color: white; text-decoration: none; border-radius: 5px; }
        .footer { margin-top: 20px; padding-top: 20px; border-top: 1px solid #dee2e6; font-size: 12px; color: #6c757d; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h2>` + notification.Title + `</h2>
        </div>
        <div class="content priority-` + notification.Priority.String() + `">
            <p>` + notification.Message + `</p>
`)

	// Add image if present
	if notification.ImageURL != "" {
		html.WriteString(fmt.Sprintf(`<img src="%s" style="max-width: 100%%; height: auto;">`, notification.ImageURL))
	}

	// Add actions
	if len(notification.Actions) > 0 {
		html.WriteString(`<div class="actions">`)
		for _, action := range notification.Actions {
			html.WriteString(fmt.Sprintf(`<a href="%s" class="btn">%s</a> `, action.URL, action.Label))
		}
		html.WriteString(`</div>`)
	}

	html.WriteString(`
            <div class="footer">
                <p>DB Backup Manager - Automated Notification</p>
                <p>Priority: ` + notification.Priority.String() + ` | Type: ` + string(notification.Type) + `</p>
            </div>
        </div>
    </div>
</body>
</html>`)

	return html.String()
}

func (c *EmailChannel) composeMIMEMessage(to, subject, htmlBody string) string {
	from := fmt.Sprintf("%s <%s>", c.FromName, c.FromAddress)

	msg := "From: " + from + "\r\n"
	msg += "To: " + to + "\r\n"
	msg += "Subject: " + subject + "\r\n"
	msg += "MIME-Version: 1.0\r\n"
	msg += "Content-Type: text/html; charset=UTF-8\r\n"
	msg += "\r\n"
	msg += htmlBody

	return msg
}

// =============================================================================
// Slack Channel
// =============================================================================

// SlackChannel delivers notifications to Slack
type SlackChannel struct {
	WebhookURL string
}

func NewSlackChannel(webhookURL string) *SlackChannel {
	return &SlackChannel{WebhookURL: webhookURL}
}

func (c *SlackChannel) Name() string {
	return string(ChannelSlack)
}

func (c *SlackChannel) SupportsRichContent() bool {
	return true
}

func (c *SlackChannel) Send(ctx context.Context, notification *Notification) error {
	payload := map[string]interface{}{
		"text": notification.Title,
		"blocks": []map[string]interface{}{
			{
				"type": "header",
				"text": map[string]string{
					"type": "plain_text",
					"text": notification.Title,
				},
			},
			{
				"type": "section",
				"text": map[string]string{
					"type": "mrkdwn",
					"text": notification.Message,
				},
			},
		},
	}

	// Add priority color
	color := c.getPriorityColor(notification.Priority)
	payload["attachments"] = []map[string]interface{}{
		{
			"color": color,
			"fields": []map[string]interface{}{
				{
					"title": "Priority",
					"value": notification.Priority.String(),
					"short": true,
				},
				{
					"title": "Type",
					"value": string(notification.Type),
					"short": true,
				},
			},
		},
	}

	// Add actions as buttons
	if len(notification.Actions) > 0 {
		actions := make([]map[string]interface{}, 0)
		for _, action := range notification.Actions {
			actions = append(actions, map[string]interface{}{
				"type": "button",
				"text": map[string]string{
					"type": "plain_text",
					"text": action.Label,
				},
				"url": action.URL,
			})
		}

		blocks := payload["blocks"].([]map[string]interface{})
		blocks = append(blocks, map[string]interface{}{
			"type":     "actions",
			"elements": actions,
		})
		payload["blocks"] = blocks
	}

	// Send to Slack
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.WebhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack API returned status %d", resp.StatusCode)
	}

	return nil
}

func (c *SlackChannel) getPriorityColor(priority Priority) string {
	colors := map[Priority]string{
		PriorityUrgent: "#dc3545",
		PriorityHigh:   "#fd7e14",
		PriorityNormal: "#007bff",
		PriorityLow:    "#6c757d",
	}
	return colors[priority]
}

// =============================================================================
// Microsoft Teams Channel
// =============================================================================

// TeamsChannel delivers notifications to Microsoft Teams
type TeamsChannel struct {
	WebhookURL string
}

func NewTeamsChannel(webhookURL string) *TeamsChannel {
	return &TeamsChannel{WebhookURL: webhookURL}
}

func (c *TeamsChannel) Name() string {
	return string(ChannelTeams)
}

func (c *TeamsChannel) SupportsRichContent() bool {
	return true
}

func (c *TeamsChannel) Send(ctx context.Context, notification *Notification) error {
	payload := map[string]interface{}{
		"@type":      "MessageCard",
		"@context":   "https://schema.org/extensions",
		"summary":    notification.Title,
		"themeColor": c.getPriorityColor(notification.Priority),
		"title":      notification.Title,
		"text":       notification.Message,
		"sections": []map[string]interface{}{
			{
				"facts": []map[string]string{
					{"name": "Priority", "value": notification.Priority.String()},
					{"name": "Type", "value": string(notification.Type)},
					{"name": "Category", "value": notification.Category},
				},
			},
		},
	}

	// Add actions
	if len(notification.Actions) > 0 {
		actions := make([]map[string]interface{}, 0)
		for _, action := range notification.Actions {
			actions = append(actions, map[string]interface{}{
				"@type": "OpenUri",
				"name":  action.Label,
				"targets": []map[string]string{
					{"os": "default", "uri": action.URL},
				},
			})
		}
		payload["potentialAction"] = actions
	}

	// Send to Teams
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.WebhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("teams API returned status %d", resp.StatusCode)
	}

	return nil
}

func (c *TeamsChannel) getPriorityColor(priority Priority) string {
	colors := map[Priority]string{
		PriorityUrgent: "FF0000",
		PriorityHigh:   "FFA500",
		PriorityNormal: "0078D7",
		PriorityLow:    "808080",
	}
	return colors[priority]
}

// =============================================================================
// Discord Channel
// =============================================================================

// DiscordChannel delivers notifications to Discord
type DiscordChannel struct {
	WebhookURL string
}

func NewDiscordChannel(webhookURL string) *DiscordChannel {
	return &DiscordChannel{WebhookURL: webhookURL}
}

func (c *DiscordChannel) Name() string {
	return string(ChannelDiscord)
}

func (c *DiscordChannel) SupportsRichContent() bool {
	return true
}

func (c *DiscordChannel) Send(ctx context.Context, notification *Notification) error {
	embed := map[string]interface{}{
		"title":       notification.Title,
		"description": notification.Message,
		"color":       c.getPriorityColorInt(notification.Priority),
		"timestamp":   notification.CreatedAt.Format(time.RFC3339),
		"fields": []map[string]interface{}{
			{"name": "Priority", "value": notification.Priority.String(), "inline": true},
			{"name": "Type", "value": string(notification.Type), "inline": true},
		},
	}

	// Add image
	if notification.ImageURL != "" {
		embed["image"] = map[string]string{"url": notification.ImageURL}
	}

	payload := map[string]interface{}{
		"embeds": []map[string]interface{}{embed},
	}

	// Send to Discord
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.WebhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("discord API returned status %d", resp.StatusCode)
	}

	return nil
}

func (c *DiscordChannel) getPriorityColorInt(priority Priority) int {
	colors := map[Priority]int{
		PriorityUrgent: 0xFF0000,
		PriorityHigh:   0xFFA500,
		PriorityNormal: 0x0078D7,
		PriorityLow:    0x808080,
	}
	return colors[priority]
}

// =============================================================================
// Webhook Channel
// =============================================================================

// WebhookChannel delivers notifications to custom webhooks
type WebhookChannel struct {
	URL string
}

func NewWebhookChannel(url string) *WebhookChannel {
	return &WebhookChannel{URL: url}
}

func (c *WebhookChannel) Name() string {
	return string(ChannelWebhook)
}

func (c *WebhookChannel) SupportsRichContent() bool {
	return true
}

func (c *WebhookChannel) Send(ctx context.Context, notification *Notification) error {
	// Send full notification as JSON
	jsonData, err := json.Marshal(notification)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "DB-Backup-Notification-System/1.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// =============================================================================
// Push Notification Channel (FCM/APNS)
// =============================================================================

// PushChannel delivers push notifications (simplified - would need FCM/APNS integration)
type PushChannel struct {
	FCMServerKey string
	APNSKeyID    string
}

func NewPushChannel(fcmKey, apnsKey string) *PushChannel {
	return &PushChannel{
		FCMServerKey: fcmKey,
		APNSKeyID:    apnsKey,
	}
}

func (c *PushChannel) Name() string {
	return string(ChannelPush)
}

func (c *PushChannel) SupportsRichContent() bool {
	return true
}

func (c *PushChannel) Send(ctx context.Context, notification *Notification) error {
	// Simplified push notification (would need proper FCM/APNS implementation)
	deviceToken, ok := notification.Metadata["device_token"].(string)
	if !ok {
		return fmt.Errorf("device token not found in metadata")
	}

	// Send via FCM (Firebase Cloud Messaging)
	return c.sendFCM(ctx, deviceToken, notification)
}

func (c *PushChannel) sendFCM(ctx context.Context, deviceToken string, notification *Notification) error {
	payload := map[string]interface{}{
		"to": deviceToken,
		"notification": map[string]interface{}{
			"title": notification.Title,
			"body":  notification.Message,
			"icon":  "notification_icon",
			"sound": "default",
		},
		"data": notification.Metadata,
		"priority": "high",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://fcm.googleapis.com/fcm/send", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "key="+c.FCMServerKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("FCM API returned status %d", resp.StatusCode)
	}

	return nil
}

// =============================================================================
// In-App Channel (WebSocket)
// =============================================================================

// InAppChannel delivers in-app notifications via WebSocket
type InAppChannel struct {
	broadcaster *WebSocketBroadcaster
}

func NewInAppChannel(broadcaster *WebSocketBroadcaster) *InAppChannel {
	return &InAppChannel{broadcaster: broadcaster}
}

func (c *InAppChannel) Name() string {
	return string(ChannelInApp)
}

func (c *InAppChannel) SupportsRichContent() bool {
	return true
}

func (c *InAppChannel) Send(ctx context.Context, notification *Notification) error {
	return c.broadcaster.Broadcast(notification.UserID, notification)
}

// WebSocketBroadcaster handles WebSocket broadcasting (simplified)
type WebSocketBroadcaster struct {
	// Would contain WebSocket connection pool
}

func (b *WebSocketBroadcaster) Broadcast(userID string, notification *Notification) error {
	// Simplified - would send to active WebSocket connections
	return nil
}
