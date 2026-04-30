package notifications

import (
	"fmt"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type telegramSender struct {
	bot *tgbotapi.BotAPI
}

func newTelegram(token string) (*telegramSender, error) {
	if token == "" {
		return nil, nil
	}
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}
	return &telegramSender{bot: bot}, nil
}

func (t *telegramSender) send(chatIDStr, message string) error {
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid telegram chat id %q", chatIDStr)
	}
	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = "Markdown"
	_, err = t.bot.Send(msg)
	return err
}
