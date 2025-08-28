package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"gopkg.in/telebot.v4"
)

// AlertmanagerPayload corresponds to the JSON structure sent by Alertmanager.
type AlertmanagerPayload struct {
	Receiver string  `json:"receiver"`
	Status   string  `json:"status"`
	Alerts   []Alert `json:"alerts"`
}

// Alert contains detail information about the one notification.
type Alert struct {
	Status      string            `json:"status"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartsAt    time.Time         `json:"startsAt"`
	EndsAt      time.Time         `json:"endsAt"`
}

// AlertmanagerWebhookHandler - its a main handler foor webhooks.
func (b *Bot) AlertmanagerWebhookHandler(writer http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(writer, "Only POST requests are accepted", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		b.log.Error("Failed to read webhook body", "error", err)
		http.Error(writer, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	defer req.Body.Close()

	var payload AlertmanagerPayload
	if err = json.Unmarshal(body, &payload); err != nil {
		b.log.Error("Failed to unmarshal webhook payload", "error", err, "body", string(body))
		http.Error(writer, "Failed to decode payload", http.StatusBadRequest)
		return
	}

	admins, err := b.usrepo.GetAdmins(context.Background())
	if err != nil {
		b.log.Error("Failed to get admins for alert", "error", err)
	}

	if len(admins) == 0 {
		b.log.Warn("No admins found to send alerts to.")
		writer.WriteHeader(http.StatusOK)
		return
	}

	go func() {
		for _, alert := range payload.Alerts {
			message := formatAlertMessage(alert)
			for _, admin := range admins {
				_, err = b.bot.Send(telebot.ChatID(admin.TelegramID), message, telebot.ModeMarkdown)
				if err != nil {
					b.log.Warn("Failed to send alert to admin", "admin_id", admin.TelegramID, "error", err)
				}
				const telegramRateTimeout = 100 * time.Millisecond
				time.Sleep(telegramRateTimeout)
			}
		}
	}()

	writer.WriteHeader(http.StatusOK)
	if _, err = writer.Write([]byte("Alerts received successfully.")); err != nil {
		b.log.Error("Failed to send success message to requester", "error", err)
	}
}

// formatAlertMessage formats the one alert in readable messsage for Telegram.
func formatAlertMessage(alert Alert) string {
	var icon string
	status := strings.ToUpper(alert.Status)
	if status == "FIRING" {
		icon = "🔥"
	} else {
		icon = "✅"
	}

	summary := alert.Annotations["summary"]
	description := alert.Annotations["description"]
	job := alert.Labels["job"]
	severity := alert.Labels["severity"]

	var messageBuilder strings.Builder
	messageBuilder.WriteString(fmt.Sprintf("%s **%s** (%s)\n\n", icon, status, severity))
	messageBuilder.WriteString(fmt.Sprintf("**Summary**: %s\n", summary))
	if description != "" {
		messageBuilder.WriteString(fmt.Sprintf("**Description**: %s\n", description))
	}
	if job != "" {
		messageBuilder.WriteString(fmt.Sprintf("**Service**: `%s`\n", job))
	}

	return messageBuilder.String()
}
