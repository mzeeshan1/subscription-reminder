# SubMan — Subscription Manager

A web app to track recurring subscriptions and receive renewal reminders via Telegram and WhatsApp, 5 days before each renewal date.

## Tech Stack

- **Go + Gin** — HTTP server
- **PostgreSQL** — persistent storage
- **Redis** — JWT blacklist, rate limiting, notification dedup, subscription cache
- **Telegram Bot API** — renewal notifications
- **Twilio WhatsApp** — renewal notifications
- **Tailwind CSS** — UI (loaded via CDN)

---

## Prerequisites

| Tool | Version | Install |
|---|---|---|
| Go | 1.21+ | https://go.dev/dl |
| PostgreSQL | 14+ | https://www.postgresql.org/download |
| Redis | 7+ | https://redis.io/docs/install |

---

## Local Setup

### 1. Clone and enter the project

```bash
cd subscription-reminder
```

### 2. Create the PostgreSQL database

```bash
psql -U postgres -c "CREATE DATABASE subman;"
```

### 3. Configure environment variables

```bash
cp .env.example .env
```

Open `.env` and set the values:

```env
PORT=8080
DATABASE_URL=postgres://postgres:yourpassword@localhost:5432/subman?sslmode=disable
REDIS_URL=redis://localhost:6379
JWT_SECRET=any-long-random-string

# Leave blank to skip that notification channel
TELEGRAM_BOT_TOKEN=
TWILIO_ACCOUNT_SID=
TWILIO_AUTH_TOKEN=
TWILIO_FROM_NUMBER=whatsapp:+14155238886
```

> You can leave `TELEGRAM_BOT_TOKEN` and Twilio values empty while testing — the app runs fine without them and simply skips those channels.

### 4. Start PostgreSQL and Redis

```bash
# macOS (Homebrew)
brew services start postgresql
brew services start redis

# Linux (systemd)
sudo systemctl start postgresql
sudo systemctl start redis
```

### 5. Run the app

```bash
# Load env vars and start
export $(cat .env | xargs) && go run .
```

The server starts on `http://localhost:8080`. The database schema is created automatically on first run.

---

## Testing Each Feature

### Auth

| What to test | Steps |
|---|---|
| Register | Go to `http://localhost:8080/register`, create an account |
| Login | Go to `/login`, sign in with your credentials |
| Wrong password | Enter wrong password — you should see "Invalid email or password" |
| Rate limiting | Submit the login form more than 10 times in a minute — you should see a rate limit error |
| Logout | Click Logout in the nav — you are redirected to `/login` and the old token is blacklisted in Redis |

Verify the token blacklist works:
```bash
# After logging out, check Redis for the blacklisted token
redis-cli KEYS "bl:*"
```

### Subscriptions

| What to test | Steps |
|---|---|
| Add | Click **+ Add Subscription**, fill in the form, submit |
| List | Dashboard shows all your subscriptions sorted by renewal date |
| Edit | Click **Edit** on any row, change values, save |
| Delete | Click **Delete** on any row, confirm the dialog |
| Validation | Try submitting the add form with no name or a negative cost — you should see an error |

Verify Redis caching:
```bash
# After loading the dashboard, a cache key should exist
redis-cli KEYS "subs:*"

# After editing or deleting a subscription, the key should be gone
redis-cli KEYS "subs:*"
```

### Cost Summary

On the dashboard, the three stat cards at the top show:
- Total number of subscriptions
- Total monthly cost (all cycles normalised to monthly)
- Total yearly cost

Add a mix of monthly/yearly/quarterly subscriptions to verify the normalisation is correct:
- A $120/year subscription should show as $10/month
- A $30/quarter subscription should show as $10/month

### Renewal Status Badges

The **Days Left** column on the dashboard uses colour coding:

| Colour | Meaning |
|---|---|
| 🟢 Green | More than 14 days away |
| 🟡 Yellow | 7–14 days away |
| 🟠 Orange | 5 days or fewer |
| 🔴 Red | Today or overdue |

To test this, add subscriptions with different renewal dates (today, tomorrow, in 3 days, in 30 days) and verify the colours.

### Notification Settings

1. Go to `http://localhost:8080/settings`
2. Enter your Telegram Chat ID and/or WhatsApp number
3. Click **Save Settings**
4. Verify the values are persisted by refreshing the page

**Getting your Telegram Chat ID:**
1. Open Telegram and search for `@userinfobot`
2. Send it any message — it replies with your numeric Chat ID
3. Paste that number into the settings page

**Setting up WhatsApp (Twilio sandbox):**
1. Create a free Twilio account at https://twilio.com
2. Go to **Messaging → Try it out → Send a WhatsApp message**
3. Follow the sandbox join instructions (send a code to the Twilio number)
4. Add your Twilio credentials to `.env`
5. Enter your number in settings (include country code, e.g. `+14155552671`)

### Reminder Worker (5-day notifications)

The worker runs automatically at **09:00 daily**. To test it without waiting:

**Option 1 — Add a subscription renewing in exactly 5 days:**
```bash
# The worker queries for next_renewal = today + 5 days
# Add a subscription with that date via the UI, then trigger the worker
```

**Option 2 — Trigger the run manually by temporarily editing the cron schedule:**

In `worker/reminder.go`, change the cron expression to run every minute for testing:
```go
// Change this:
r.cron.AddFunc("0 9 * * *", r.run)

// To this (runs every minute):
r.cron.AddFunc("* * * * *", r.run)
```

Then restart the app and watch the logs. You should see Telegram/WhatsApp messages arrive within a minute.

**Verify deduplication:**
```bash
# After a notification fires, Redis should hold a dedup key
redis-cli KEYS "notif:*"

# The worker will not send a second message for the same subscription + date
```

---

## Project Structure

```
.
├── main.go                  # entry point, wires all components
├── config/config.go         # reads env vars
├── db/db.go                 # postgres connection + schema migration
├── cache/redis.go           # all Redis operations
├── models/                  # User and Subscription types
├── handlers/
│   ├── auth.go              # register, login, logout
│   ├── subscription.go      # dashboard, add, edit, delete
│   └── settings.go          # notification settings
├── middleware/
│   ├── auth.go              # JWT cookie validation
│   └── ratelimit.go         # login rate limiting
├── notifications/
│   ├── telegram.go          # Telegram sender
│   ├── whatsapp.go          # Twilio WhatsApp sender
│   └── notifier.go          # message builder + channel dispatcher
├── worker/reminder.go       # daily cron job
└── templates/               # server-rendered HTML (Tailwind CSS)
```

## Redis Key Reference

| Key pattern | TTL | Purpose |
|---|---|---|
| `bl:<jwt_token>` | 25h | Blacklisted tokens (post-logout) |
| `rl:login:<ip>` | 1 min | Login attempt counter per IP |
| `notif:<subID>:<date>:<channel>` | 25h | Sent-notification dedup |
| `subs:<userID>` | 5 min | Cached subscription list |
