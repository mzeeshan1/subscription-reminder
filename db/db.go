package db

import (
	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func Connect(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	return db, db.Ping()
}

func Migrate(db *sql.DB) error {
	_, err := db.Exec(schema)
	return err
}

const schema = `
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS users (
	id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
	email            TEXT        UNIQUE NOT NULL,
	password_hash    TEXT        NOT NULL,
	telegram_chat_id TEXT        NOT NULL DEFAULT '',
	whatsapp_number  TEXT        NOT NULL DEFAULT '',
	created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS subscriptions (
	id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id      UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	name         TEXT        NOT NULL,
	cost         NUMERIC(10,2) NOT NULL,
	currency     TEXT        NOT NULL DEFAULT 'USD',
	cycle        TEXT        NOT NULL,
	next_renewal DATE        NOT NULL,
	notes        TEXT        NOT NULL DEFAULT '',
	created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_subs_user    ON subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_subs_renewal ON subscriptions(next_renewal);
`
