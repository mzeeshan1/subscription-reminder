package worker

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/robfig/cron/v3"
	"subscription-manager/cache"
	"subscription-manager/metrics"
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
	r.sendReminders(ctx)
	r.advanceRenewals(ctx)
}

func (r *Reminder) sendReminders(ctx context.Context) {
	target := time.Now().UTC().AddDate(0, 0, 5).Format("2006-01-02")

	rows, err := r.db.QueryContext(ctx, `
		SELECT s.id, s.user_id, s.name, s.cost, s.currency, s.cycle, s.next_renewal, s.notes,
		       u.telegram_chat_id, u.whatsapp_number, u.slack_webhook_url
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
		var telegramID, whatsappNum, slackWebhook string

		if err := rows.Scan(
			&sub.ID, &sub.UserID, &sub.Name, &sub.Cost, &sub.Currency,
			&sub.Cycle, &sub.NextRenewal, &sub.Notes,
			&telegramID, &whatsappNum, &slackWebhook,
		); err != nil {
			log.Printf("reminder: scan error: %v", err)
			continue
		}

		renewalStr := sub.NextRenewal.Format("2006-01-02")

		if sent, _ := r.cache.WasNotificationSent(ctx, sub.ID, renewalStr, "telegram"); !sent {
			if err := r.notifier.SendTelegram(telegramID, sub); err != nil {
				log.Printf("reminder: telegram error (sub %s): %v", sub.ID, err)
				metrics.NotificationsFailedTotal.WithLabelValues("telegram").Inc()
			} else {
				r.cache.MarkNotificationSent(ctx, sub.ID, renewalStr, "telegram")
				metrics.NotificationsSentTotal.WithLabelValues("telegram").Inc()
			}
		}

		if sent, _ := r.cache.WasNotificationSent(ctx, sub.ID, renewalStr, "whatsapp"); !sent {
			if err := r.notifier.SendWhatsApp(whatsappNum, sub); err != nil {
				log.Printf("reminder: whatsapp error (sub %s): %v", sub.ID, err)
				metrics.NotificationsFailedTotal.WithLabelValues("whatsapp").Inc()
			} else {
				r.cache.MarkNotificationSent(ctx, sub.ID, renewalStr, "whatsapp")
				metrics.NotificationsSentTotal.WithLabelValues("whatsapp").Inc()
			}
		}

		if sent, _ := r.cache.WasNotificationSent(ctx, sub.ID, renewalStr, "slack"); !sent {
			if err := r.notifier.SendSlack(slackWebhook, sub); err != nil {
				log.Printf("reminder: slack error (sub %s): %v", sub.ID, err)
				metrics.NotificationsFailedTotal.WithLabelValues("slack").Inc()
			} else {
				r.cache.MarkNotificationSent(ctx, sub.ID, renewalStr, "slack")
				metrics.NotificationsSentTotal.WithLabelValues("slack").Inc()
			}
		}
	}
}

func (r *Reminder) advanceRenewals(ctx context.Context) {
	today := time.Now().UTC().Format("2006-01-02")

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, cycle, next_renewal
		FROM subscriptions
		WHERE next_renewal <= $1
	`, today)
	if err != nil {
		log.Printf("advance renewals: query error: %v", err)
		return
	}
	defer rows.Close()

	type record struct {
		id      string
		userID  string
		cycle   models.Cycle
		current time.Time
	}

	var due []record
	for rows.Next() {
		var rec record
		if err := rows.Scan(&rec.id, &rec.userID, &rec.cycle, &rec.current); err != nil {
			log.Printf("advance renewals: scan error: %v", err)
			continue
		}
		due = append(due, rec)
	}
	rows.Close()

	for _, rec := range due {
		sub := models.Subscription{Cycle: rec.cycle, NextRenewal: rec.current}
		next := sub.NextRenewalDate()

		_, err := r.db.ExecContext(ctx, `
			UPDATE subscriptions SET next_renewal=$1, updated_at=NOW() WHERE id=$2
		`, next.Format("2006-01-02"), rec.id)
		if err != nil {
			log.Printf("advance renewals: update error (sub %s): %v", rec.id, err)
			continue
		}

		r.cache.InvalidateSubsCache(ctx, rec.userID)
		log.Printf("advance renewals: advanced sub %s from %s to %s", rec.id, rec.current.Format("2006-01-02"), next.Format("2006-01-02"))
	}
}
