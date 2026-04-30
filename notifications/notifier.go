package notifications

import (
	"fmt"
	"log"

	"subscription-manager/config"
	"subscription-manager/models"
)

type Notifier struct {
	telegram *telegramSender
	whatsapp *whatsappSender
}

func New(cfg *config.Config) *Notifier {
	tg, err := newTelegram(cfg.TelegramBotToken)
	if err != nil {
		log.Printf("warn: telegram init failed: %v", err)
	}
	return &Notifier{
		telegram: tg,
		whatsapp: newWhatsApp(cfg.TwilioAccountSID, cfg.TwilioAuthToken, cfg.TwilioFromNumber),
	}
}

func (n *Notifier) SendTelegram(chatID string, sub models.Subscription) error {
	if n.telegram == nil || chatID == "" {
		return nil
	}
	return n.telegram.send(chatID, buildMessage(sub))
}

func (n *Notifier) SendWhatsApp(number string, sub models.Subscription) error {
	if n.whatsapp == nil || number == "" {
		return nil
	}
	return n.whatsapp.send(number, buildMessage(sub))
}

func buildMessage(sub models.Subscription) string {
	msg := fmt.Sprintf(
		"⏰ *Renewal Reminder*\n\nYour *%s* subscription renews in 5 days on *%s*.\nCost: *%.2f %s* / %s",
		sub.Name,
		sub.NextRenewal.Format("Jan 2, 2006"),
		sub.Cost,
		sub.Currency,
		sub.Cycle,
	)
	if sub.Notes != "" {
		msg += "\n\n_" + sub.Notes + "_"
	}
	return msg
}
