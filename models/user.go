package models

import "time"

type User struct {
	ID              string
	Email           string
	PasswordHash    string
	TelegramChatID  string
	WhatsAppNumber  string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
