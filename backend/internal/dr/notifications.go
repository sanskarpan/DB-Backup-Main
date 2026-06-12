// Package dr provides disaster recovery testing and validation
package dr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"strings"
	"time"
)

// NotificationService handles sending notifications for DR test results
type NotificationService struct {
	slackWebhook string
	emailConfig  *EmailConfig
	httpClient   *http.Client
}

// EmailConfig contains SMTP configuration for email notifications
type EmailConfig struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	FromAddress  string
	FromName     string
}

// NewNotificationService creates a new notification service
func NewNotificationService(slackWebhook string, emailConfig *EmailConfig) *NotificationService {
	return &NotificationService{
		slackWebhook: slackWebhook,
		emailConfig:  emailConfig,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NotifyTestResult sends notifications for a test result
func (ns *NotificationService) NotifyTestResult(ctx context.Context, schedule *TestSchedule, result *TestResult) error {
	// Determine if we should notify based on result and schedule settings
	shouldNotify := (result.Success && schedule.NotifyOnSuccess) ||
		(!result.Success && schedule.NotifyOnFailure)

	if !shouldNotify {
		return nil
	}

	var errs []error

	// Send Slack notification if configured
	if ns.slackWebhook != "" && schedule.NotificationSlackWebhook != "" {
		if err := ns.SendSlackNotification(ctx, schedule, result); err != nil {
			errs = append(errs, fmt.Errorf("slack notification failed: %w", err))
		}
	}

	// Send email notifications if configured
	if ns.emailConfig != nil && len(schedule.NotificationEmails) > 0 {
		if err := ns.SendEmailNotification(ctx, schedule, result); err != nil {
			errs = append(errs, fmt.Errorf("email notification failed: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("notification errors: %v", errs)
	}

	return nil
}

// SendSlackNotification sends a Slack notification
func (ns *NotificationService) SendSlackNotification(ctx context.Context, schedule *TestSchedule, result *TestResult) error {
	webhook := schedule.NotificationSlackWebhook
	if webhook == "" {
		webhook = ns.slackWebhook
	}

	if webhook == "" {
		return fmt.Errorf("no Slack webhook configured")
	}

	message := ns.buildSlackMessage(schedule, result)

	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", webhook, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := ns.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Slack notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Slack API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// buildSlackMessage builds a Slack message payload
func (ns *NotificationService) buildSlackMessage(schedule *TestSchedule, result *TestResult) map[string]interface{} {
	// Determine color based on result
	color := "#36a64f" // Green for success
	status := "✅ PASSED"
	if !result.Success {
		color = "#ff0000" // Red for failure
		status = "❌ FAILED"
	} else if !result.RTOMet || !result.RPOMet {
		color = "#ffaa00" // Orange for warning
		status = "⚠️ PASSED WITH WARNINGS"
	}

	// Build fields
	fields := []map[string]interface{}{
		{
			"title": "Database",
			"value": result.DatabaseName,
			"short": true,
		},
		{
			"title": "Duration",
			"value": result.Duration.String(),
			"short": true,
		},
		{
			"title": "RTO",
			"value": fmt.Sprintf("%s / %s %s", result.RTO, result.RTOThreshold, boolToEmoji(result.RTOMet)),
			"short": true,
		},
		{
			"title": "RPO",
			"value": fmt.Sprintf("%s / %s %s", result.RPO, result.RPOThreshold, boolToEmoji(result.RPOMet)),
			"short": true,
		},
		{
			"title": "Schema Validation",
			"value": boolToPassFailEmoji(result.SchemaValid),
			"short": true,
		},
		{
			"title": "Data Validation",
			"value": boolToPassFailEmoji(result.RowCountValid && result.SampleDataValid),
			"short": true,
		},
	}

	// Add smoke test results
	if result.SmokeTestsPassed > 0 || result.SmokeTestsFailed > 0 {
		fields = append(fields, map[string]interface{}{
			"title": "Smoke Tests",
			"value": fmt.Sprintf("%d passed, %d failed", result.SmokeTestsPassed, result.SmokeTestsFailed),
			"short": true,
		})
	}

	// Build attachment
	attachment := map[string]interface{}{
		"color":      color,
		"title":      fmt.Sprintf("DR Test: %s", schedule.Name),
		"text":       fmt.Sprintf("Test Status: %s", status),
		"fields":     fields,
		"footer":     "DB-Backup DR Testing",
		"footer_icon": "https://platform.slack-edge.com/img/default_application_icon.png",
		"ts":         result.EndTime.Unix(),
	}

	// Add error details if test failed
	if !result.Success {
		var errorDetails []string

		if len(result.SchemaErrors) > 0 {
			errorDetails = append(errorDetails, fmt.Sprintf("*Schema Errors:* %d", len(result.SchemaErrors)))
		}
		if len(result.RowCountErrors) > 0 {
			errorDetails = append(errorDetails, fmt.Sprintf("*Row Count Errors:* %d", len(result.RowCountErrors)))
		}
		if len(result.SampleDataErrors) > 0 {
			errorDetails = append(errorDetails, fmt.Sprintf("*Data Errors:* %d", len(result.SampleDataErrors)))
		}
		if len(result.SmokeTestErrors) > 0 {
			errorDetails = append(errorDetails, fmt.Sprintf("*Smoke Test Errors:* %d", len(result.SmokeTestErrors)))
		}

		if len(errorDetails) > 0 {
			attachment["text"] = fmt.Sprintf("Test Status: %s\n\n%s", status, strings.Join(errorDetails, "\n"))
		}
	}

	return map[string]interface{}{
		"username":    "DR Test Bot",
		"icon_emoji":  ":shield:",
		"attachments": []interface{}{attachment},
	}
}

// SendEmailNotification sends an email notification
func (ns *NotificationService) SendEmailNotification(ctx context.Context, schedule *TestSchedule, result *TestResult) error {
	if ns.emailConfig == nil {
		return fmt.Errorf("email configuration not set")
	}

	if len(schedule.NotificationEmails) == 0 {
		return fmt.Errorf("no email recipients configured")
	}

	subject := ns.buildEmailSubject(schedule, result)
	body := ns.buildEmailBody(schedule, result)

	return ns.sendEmail(schedule.NotificationEmails, subject, body)
}

// buildEmailSubject builds the email subject line
func (ns *NotificationService) buildEmailSubject(schedule *TestSchedule, result *TestResult) string {
	status := "PASSED"
	if !result.Success {
		status = "FAILED"
	} else if !result.RTOMet || !result.RPOMet {
		status = "PASSED WITH WARNINGS"
	}

	return fmt.Sprintf("[DR Test] %s - %s - %s",
		status,
		schedule.Name,
		result.DatabaseName)
}

// buildEmailBody builds the email body (HTML)
func (ns *NotificationService) buildEmailBody(schedule *TestSchedule, result *TestResult) string {
	statusColor := "#28a745" // Green
	statusText := "PASSED"
	if !result.Success {
		statusColor = "#dc3545" // Red
		statusText = "FAILED"
	} else if !result.RTOMet || !result.RPOMet {
		statusColor = "#ffc107" // Yellow
		statusText = "PASSED WITH WARNINGS"
	}

	var html strings.Builder

	html.WriteString(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: ` + statusColor + `; color: white; padding: 20px; text-align: center; border-radius: 5px; }
        .content { background-color: #f9f9f9; padding: 20px; margin-top: 20px; border-radius: 5px; }
        .metric { margin: 10px 0; padding: 10px; background-color: white; border-left: 4px solid #007bff; }
        .metric-label { font-weight: bold; color: #555; }
        .metric-value { color: #333; }
        .pass { color: #28a745; }
        .fail { color: #dc3545; }
        .warning { color: #ffc107; }
        .footer { margin-top: 20px; text-align: center; color: #777; font-size: 12px; }
        table { width: 100%; border-collapse: collapse; margin-top: 10px; }
        table th, table td { padding: 8px; text-align: left; border-bottom: 1px solid #ddd; }
        table th { background-color: #f2f2f2; }
        .error-section { background-color: #fff3cd; border-left: 4px solid #ffc107; padding: 10px; margin-top: 10px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>DR Test Report</h1>
            <h2>` + statusText + `</h2>
        </div>

        <div class="content">
            <h3>Test Details</h3>
            <div class="metric">
                <span class="metric-label">Test Name:</span>
                <span class="metric-value">` + schedule.Name + `</span>
            </div>
            <div class="metric">
                <span class="metric-label">Database:</span>
                <span class="metric-value">` + result.DatabaseName + `</span>
            </div>
            <div class="metric">
                <span class="metric-label">Test ID:</span>
                <span class="metric-value">` + result.TestID + `</span>
            </div>
            <div class="metric">
                <span class="metric-label">Start Time:</span>
                <span class="metric-value">` + result.StartTime.Format("2006-01-02 15:04:05 MST") + `</span>
            </div>
            <div class="metric">
                <span class="metric-label">Duration:</span>
                <span class="metric-value">` + result.Duration.String() + `</span>
            </div>

            <h3>Performance Metrics</h3>
            <table>
                <tr>
                    <th>Metric</th>
                    <th>Actual</th>
                    <th>Threshold</th>
                    <th>Status</th>
                </tr>
                <tr>
                    <td>RTO (Recovery Time)</td>
                    <td>` + result.RTO.String() + `</td>
                    <td>` + result.RTOThreshold.String() + `</td>
                    <td class="` + boolToClass(result.RTOMet) + `">` + boolToPassFailText(result.RTOMet) + `</td>
                </tr>
                <tr>
                    <td>RPO (Data Loss)</td>
                    <td>` + result.RPO.String() + `</td>
                    <td>` + result.RPOThreshold.String() + `</td>
                    <td class="` + boolToClass(result.RPOMet) + `">` + boolToPassFailText(result.RPOMet) + `</td>
                </tr>
            </table>

            <h3>Validation Results</h3>
            <table>
                <tr>
                    <th>Validation</th>
                    <th>Status</th>
                </tr>
                <tr>
                    <td>Schema Validation</td>
                    <td class="` + boolToClass(result.SchemaValid) + `">` + boolToPassFailText(result.SchemaValid) + `</td>
                </tr>
                <tr>
                    <td>Row Count Validation</td>
                    <td class="` + boolToClass(result.RowCountValid) + `">` + boolToPassFailText(result.RowCountValid) + `</td>
                </tr>
                <tr>
                    <td>Sample Data Validation</td>
                    <td class="` + boolToClass(result.SampleDataValid) + `">` + boolToPassFailText(result.SampleDataValid) + `</td>
                </tr>
                <tr>
                    <td>Smoke Tests</td>
                    <td>` + fmt.Sprintf("%d passed, %d failed", result.SmokeTestsPassed, result.SmokeTestsFailed) + `</td>
                </tr>
            </table>
`)

	// Add error details if any
	if !result.Success {
		html.WriteString(`
            <h3>Error Details</h3>
`)

		if len(result.SchemaErrors) > 0 {
			html.WriteString(`
            <div class="error-section">
                <strong>Schema Errors:</strong>
                <ul>
`)
			for _, err := range result.SchemaErrors {
				html.WriteString(fmt.Sprintf("                    <li>%s</li>\n", err))
			}
			html.WriteString(`
                </ul>
            </div>
`)
		}

		if len(result.RowCountErrors) > 0 {
			html.WriteString(`
            <div class="error-section">
                <strong>Row Count Errors:</strong>
                <ul>
`)
			for _, err := range result.RowCountErrors {
				html.WriteString(fmt.Sprintf("                    <li>%s</li>\n", err))
			}
			html.WriteString(`
                </ul>
            </div>
`)
		}

		if len(result.SampleDataErrors) > 0 {
			html.WriteString(`
            <div class="error-section">
                <strong>Sample Data Errors:</strong>
                <ul>
`)
			for _, err := range result.SampleDataErrors {
				html.WriteString(fmt.Sprintf("                    <li>%s</li>\n", err))
			}
			html.WriteString(`
                </ul>
            </div>
`)
		}

		if len(result.SmokeTestErrors) > 0 {
			html.WriteString(`
            <div class="error-section">
                <strong>Smoke Test Errors:</strong>
                <ul>
`)
			for _, err := range result.SmokeTestErrors {
				html.WriteString(fmt.Sprintf("                    <li>%s</li>\n", err))
			}
			html.WriteString(`
                </ul>
            </div>
`)
		}
	}

	html.WriteString(`
        </div>

        <div class="footer">
            <p>This is an automated DR test notification from DB-Backup DR Testing Framework</p>
            <p>Test Environment ID: ` + result.TestEnvironmentID + `</p>
        </div>
    </div>
</body>
</html>
`)

	return html.String()
}

// sendEmail sends an email using SMTP
func (ns *NotificationService) sendEmail(to []string, subject, body string) error {
	if ns.emailConfig == nil {
		return fmt.Errorf("email configuration not set")
	}

	// Build email message
	from := ns.emailConfig.FromAddress
	if ns.emailConfig.FromName != "" {
		from = fmt.Sprintf("%s <%s>", ns.emailConfig.FromName, ns.emailConfig.FromAddress)
	}

	headers := make(map[string]string)
	headers["From"] = from
	headers["To"] = strings.Join(to, ", ")
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=\"UTF-8\""

	var message strings.Builder
	for k, v := range headers {
		message.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	message.WriteString("\r\n")
	message.WriteString(body)

	// Connect to SMTP server
	addr := fmt.Sprintf("%s:%d", ns.emailConfig.SMTPHost, ns.emailConfig.SMTPPort)

	// Create authentication
	auth := smtp.PlainAuth("",
		ns.emailConfig.SMTPUsername,
		ns.emailConfig.SMTPPassword,
		ns.emailConfig.SMTPHost)

	// Send email
	err := smtp.SendMail(addr, auth, ns.emailConfig.FromAddress, to, []byte(message.String()))
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// Helper functions

func boolToEmoji(b bool) string {
	if b {
		return "✅"
	}
	return "❌"
}

func boolToPassFailEmoji(b bool) string {
	if b {
		return "✅ PASS"
	}
	return "❌ FAIL"
}

func boolToClass(b bool) string {
	if b {
		return "pass"
	}
	return "fail"
}

func boolToPassFailText(b bool) string {
	if b {
		return "PASS"
	}
	return "FAIL"
}
