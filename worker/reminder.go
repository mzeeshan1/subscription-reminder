package worker

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/robfig/cron/v3"
	"subscription-manager/cache"
	"subscription-manager/models"
	"subscription-manager/notifications"
)

type Reminder struct {
	db       *sql.DB
	cache    *cache.Cache
	notifier *notifications.Notifier
	cron     *cron.Cron
}

func New(db *sql.DB, c *cache.Cache, n *notifications.Notifier) *Reminder {
	return &Reminder{db: db, cache: c, notifier: n, cron: cron.New()}
}

func (r *Reminder) Start() {
	r.cron.AddFunc("0 9 * * *", r.run) // every day at 09:00
	r.cron.Start()
	log.Println("reminder worker started — fires daily at 09:00")
}

func (r *Reminder) Stop() { r.cron.Stop() }

func (r *Reminder) run() {
	ctx := context.Background()
	target := time.Now().UTC().AddDate(0, 0, 5).Format("2006-01-02")

	rows, err := r.db.QueryContext(ctx, `
		SELECT s.id, s.user_id, s.name, s.cost, s.currency, s.cycle, s.next_renewal, s.notes,
		       u.telegram_chat_id, u.whatsapp_number
		FROM subscriptions s
		JOIN users u ON s.user_id = u.id
		WHERE s.next_renewal = $1
	`, target)
	if err != nil {
		log.Printf("reminder: query error: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var sub models.Subscription
		var telegramID, whatsappNum string

		if err := rows.Scan(
			&sub.ID, &sub.UserID, &sub.Name, &sub.Cost, &sub.Currency,
			&sub.Cycle, &sub.NextRenewal, &sub.Notes,
			&telegramID, &whatsappNum,
		); err != nil {
			log.Printf("reminder: scan error: %v", err)
			continue
		}

		renewalStr := sub.NextRenewal.Format("2006-01-02")

		if sent, _ := r.cache.WasNotificationSent(ctx, sub.ID, renewalStr, "telegram"); !sent {
			if err := r.notifier.SendTelegram(telegramID, sub); err != nil {
				log.Printf("reminder: telegram error (sub %s): %v", sub.ID, err)
			} else {
				r.cache.MarkNotificationSent(ctx, sub.ID, renewalStr, "telegram")
			}
		}

		if sent, _ := r.cache.WasNotificationSent(ctx, sub.ID, renewalStr, "whatsapp"); !sent {
			if err := r.notifier.SendWhatsApp(whatsappNum, sub); err != nil {
				log.Printf("reminder: whatsapp error (sub %s): %v", sub.ID, err)
			} else {
				r.cache.MarkNotificationSent(ctx, sub.ID, renewalStr, "whatsapp")
			}
		}
	}
}
