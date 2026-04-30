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

	var telegramChatID, whatsappNumber string
	h.db.QueryRowContext(c.Request.Context(),
		`SELECT telegram_chat_id, whatsapp_number FROM users WHERE id=$1`, userID,
	).Scan(&telegramChatID, &whatsappNumber)

	c.HTML(http.StatusOK, "settings.html", gin.H{
		"Email":          c.GetString("email"),
		"TelegramChatID": telegramChatID,
		"WhatsAppNumber": whatsappNumber,
	})
}

func (h *SettingsHandler) UpdateSettings(c *gin.Context) {
	userID := c.GetString("userID")
	telegramChatID := c.PostForm("telegram_chat_id")
	whatsappNumber := c.PostForm("whatsapp_number")

	_, err := h.db.ExecContext(c.Request.Context(),
		`UPDATE users SET telegram_chat_id=$1, whatsapp_number=$2, updated_at=NOW() WHERE id=$3`,
		telegramChatID, whatsappNumber, userID,
	)

	data := gin.H{
		"Email":          c.GetString("email"),
		"TelegramChatID": telegramChatID,
		"WhatsAppNumber": whatsappNumber,
	}
	if err != nil {
		data["Error"] = "Failed to save settings."
		c.HTML(http.StatusInternalServerError, "settings.html", data)
		return
	}
	data["Success"] = "Settings saved."
	c.HTML(http.StatusOK, "settings.html", data)
}
