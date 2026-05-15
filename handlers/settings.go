package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"subscription-manager/cache"
)

type SettingsHandler struct {
	db    *sql.DB
	cache *cache.Cache
}

func NewSettingsHandler(db *sql.DB, c *cache.Cache) *SettingsHandler {
	return &SettingsHandler{db: db, cache: c}
}

func (h *SettingsHandler) SettingsPage(c *gin.Context) {
	userID := c.GetString("userID")

	var telegramChatID, whatsappNumber, slackWebhookURL string
	if err := h.db.QueryRowContext(c.Request.Context(),
		`SELECT telegram_chat_id, whatsapp_number, slack_webhook_url FROM users WHERE id=$1`, userID,
	).Scan(&telegramChatID, &whatsappNumber, &slackWebhookURL); err != nil {
		c.HTML(http.StatusInternalServerError, "settings.html", gin.H{
			"Email": c.GetString("email"),
			"Error": "Failed to load settings.",
		})
		return
	}

	c.HTML(http.StatusOK, "settings.html", gin.H{
		"Email":           c.GetString("email"),
		"TelegramChatID":  telegramChatID,
		"WhatsAppNumber":  whatsappNumber,
		"SlackWebhookURL": slackWebhookURL,
	})
}

func (h *SettingsHandler) UpdateSettings(c *gin.Context) {
	userID := c.GetString("userID")
	telegramChatID := c.PostForm("telegram_chat_id")
	whatsappNumber := c.PostForm("whatsapp_number")
	slackWebhookURL := c.PostForm("slack_webhook_url")

	_, err := h.db.ExecContext(c.Request.Context(),
		`UPDATE users SET telegram_chat_id=$1, whatsapp_number=$2, slack_webhook_url=$3, updated_at=NOW() WHERE id=$4`,
		telegramChatID, whatsappNumber, slackWebhookURL, userID,
	)

	data := gin.H{
		"Email":           c.GetString("email"),
		"TelegramChatID":  telegramChatID,
		"WhatsAppNumber":  whatsappNumber,
		"SlackWebhookURL": slackWebhookURL,
	}
	if err != nil {
		data["Error"] = "Failed to save settings."
		c.HTML(http.StatusInternalServerError, "settings.html", data)
		return
	}
	data["Success"] = "Settings saved."
	c.HTML(http.StatusOK, "settings.html", data)
}
