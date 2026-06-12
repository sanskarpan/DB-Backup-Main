// Package email provides email notification integration
package email

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/smtp"
	"strings"

	"github.com/sanskarpan/db-backup/internal/notification"
	pkgErrors "github.com/sanskarpan/db-backup/pkg/errors"
)

// EmailNotifier implements email notifications via SMTP
type EmailNotifier struct {
	smtpHost     string
	smtpPort     int
	username     string
	password     string
	from         string
	to           []string
	useTLS       bool
	useHTML      bool
	auth         smtp.Auth
}

// Config holds email notifier configuration
type Config struct {
	SMTPHost string
	SMTPPort int
	Username string
	Password string
	From     string
	To       []string
	UseTLS   bool
	UseHTML  bool
}

// NewEmailNotifier creates a new email notifier
func NewEmailNotifier(cfg *Config) (*EmailNotifier, error) {
	if cfg.SMTPHost == "" {
		return nil, pkgErrors.New(pkgErrors.ErrorTypeConfiguration, "SMTP host is required")
	}
	if cfg.SMTPPort == 0 {
		cfg.SMTPPort = 587 // Default SMTP submission port
	}
	if cfg.From == "" {
		return nil, pkgErrors.New(pkgErrors.ErrorTypeConfiguration, "from email address is required")
	}
	if len(cfg.To) == 0 {
		return nil, pkgErrors.New(pkgErrors.ErrorTypeConfiguration, "at least one recipient is required")
	}

	var auth smtp.Auth
	if cfg.Username != "" && cfg.Password != "" {
		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.SMTPHost)
	}

	return &EmailNotifier{
		smtpHost: cfg.SMTPHost,
		smtpPort: cfg.SMTPPort,
		username: cfg.Username,
		password: cfg.Password,
		from:     cfg.From,
		to:       cfg.To,
		useTLS:   cfg.UseTLS,
		useHTML:  cfg.UseHTML,
		auth:     auth,
	}, nil
}

// Send sends an email notification
func (e *EmailNotifier) Send(ctx context.Context, notif *notification.Notification) error {
	var body string
	var err error

	if e.useHTML {
		body, err = e.buildHTMLBody(notif)
	} else {
		body = e.buildTextBody(notif)
	}

	if err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeOperation, "failed to build email body")
	}

	subject := fmt.Sprintf("[%s] %s", strings.ToUpper(string(notif.Level)), notif.Title)

	message := e.buildMessage(subject, body)

	addr := fmt.Sprintf("%s:%d", e.smtpHost, e.smtpPort)

	err = smtp.SendMail(addr, e.auth, e.from, e.to, []byte(message))
	if err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeNetwork, "failed to send email")
	}

	return nil
}

// GetType returns the provider type
func (e *EmailNotifier) GetType() notification.ProviderType {
	return notification.ProviderTypeEmail
}

// ValidateConfig validates the configuration
func (e *EmailNotifier) ValidateConfig() error {
	if e.smtpHost == "" {
		return pkgErrors.New(pkgErrors.ErrorTypeConfiguration, "SMTP host is required")
	}
	if e.from == "" {
		return pkgErrors.New(pkgErrors.ErrorTypeConfiguration, "from address is required")
	}
	if len(e.to) == 0 {
		return pkgErrors.New(pkgErrors.ErrorTypeConfiguration, "at least one recipient is required")
	}
	return nil
}

// buildMessage builds the complete email message with headers
func (e *EmailNotifier) buildMessage(subject, body string) string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("From: %s\r\n", e.from))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(e.to, ", ")))
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))

	if e.useHTML {
		buf.WriteString("MIME-Version: 1.0\r\n")
		buf.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	} else {
		buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	}

	buf.WriteString("\r\n")
	buf.WriteString(body)

	return buf.String()
}

// buildTextBody builds a plain text email body
func (e *EmailNotifier) buildTextBody(notif *notification.Notification) string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("%s\n\n", notif.Title))
	buf.WriteString(fmt.Sprintf("%s\n\n", notif.Message))

	if len(notif.Metadata) > 0 {
		buf.WriteString("Details:\n")
		buf.WriteString("--------\n")
		for key, value := range notif.Metadata {
			buf.WriteString(fmt.Sprintf("%s: %v\n", key, value))
		}
		buf.WriteString("\n")
	}

	if len(notif.Tags) > 0 {
		buf.WriteString(fmt.Sprintf("Tags: %s\n\n", strings.Join(notif.Tags, ", ")))
	}

	buf.WriteString(fmt.Sprintf("Timestamp: %s\n", notif.Timestamp.Format("2006-01-02 15:04:05")))
	buf.WriteString("\n--\n")
	buf.WriteString("DB Backup Utility\n")

	return buf.String()
}

// buildHTMLBody builds an HTML email body
func (e *EmailNotifier) buildHTMLBody(notif *notification.Notification) (string, error) {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: {{.HeaderColor}}; color: white; padding: 15px; border-radius: 5px 5px 0 0; }
        .content { background-color: #f9f9f9; padding: 20px; border: 1px solid #ddd; }
        .metadata { background-color: white; padding: 15px; margin: 15px 0; border-left: 4px solid {{.HeaderColor}}; }
        .metadata-item { margin: 5px 0; }
        .metadata-key { font-weight: bold; color: #666; }
        .footer { text-align: center; margin-top: 20px; padding: 10px; color: #666; font-size: 12px; }
        .tags { margin: 10px 0; }
        .tag { display: inline-block; background-color: #e0e0e0; padding: 3px 8px; margin: 2px; border-radius: 3px; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h2>{{.Title}}</h2>
        </div>
        <div class="content">
            <p>{{.Message}}</p>

            {{if .Metadata}}
            <div class="metadata">
                <h3>Details</h3>
                {{range $key, $value := .Metadata}}
                <div class="metadata-item">
                    <span class="metadata-key">{{$key}}:</span> {{$value}}
                </div>
                {{end}}
            </div>
            {{end}}

            {{if .Tags}}
            <div class="tags">
                <strong>Tags:</strong>
                {{range .Tags}}
                <span class="tag">{{.}}</span>
                {{end}}
            </div>
            {{end}}

            <p style="margin-top: 20px; color: #666; font-size: 14px;">
                <strong>Timestamp:</strong> {{.Timestamp}}
            </p>
        </div>
        <div class="footer">
            DB Backup Utility - Automated Backup Notification
        </div>
    </div>
</body>
</html>
`

	t, err := template.New("email").Parse(tmpl)
	if err != nil {
		return "", err
	}

	data := map[string]interface{}{
		"Title":       notif.Title,
		"Message":     notif.Message,
		"Metadata":    notif.Metadata,
		"Tags":        notif.Tags,
		"Timestamp":   notif.Timestamp.Format("2006-01-02 15:04:05"),
		"HeaderColor": e.getLevelColor(notif.Level),
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// getLevelColor returns the color for a notification level
func (e *EmailNotifier) getLevelColor(level notification.NotificationLevel) string {
	switch level {
	case notification.LevelSuccess:
		return "#4CAF50" // green
	case notification.LevelWarning:
		return "#FF9800" // orange
	case notification.LevelError:
		return "#F44336" // red
	case notification.LevelInfo:
		return "#2196F3" // blue
	default:
		return "#9E9E9E" // gray
	}
}
