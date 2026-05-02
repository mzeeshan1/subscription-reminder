package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	client *redis.Client
}

func New(redisURL string) (*Cache, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	c := redis.NewClient(opts)
	if err := c.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}
	return &Cache{client: c}, nil
}

// BlacklistToken marks a JWT as invalid until expiry (used on logout).
func (c *Cache) BlacklistToken(ctx context.Context, token string, ttl time.Duration) error {
	return c.client.Set(ctx, "bl:"+token, 1, ttl).Err()
}

func (c *Cache) IsTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	n, err := c.client.Exists(ctx, "bl:"+token).Result()
	return n > 0, err
}

// IncrLoginAttempts increments the rate-limit counter for an IP and returns the new count.
func (c *Cache) IncrLoginAttempts(ctx context.Context, ip string) (int64, error) {
	key := "rl:login:" + ip
	pipe := c.client.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, time.Minute)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}
	return incr.Val(), nil
}

// MarkNotificationSent records that a channel notification was sent for a subscription
// renewal date. TTL of 25h ensures it won't fire twice in the same daily window.
func (c *Cache) MarkNotificationSent(ctx context.Context, subID, renewalDate, channel string) error {
	return c.client.Set(ctx, "notif:"+subID+":"+renewalDate+":"+channel, 1, 25*time.Hour).Err()
}

func (c *Cache) WasNotificationSent(ctx context.Context, subID, renewalDate, channel string) (bool, error) {
	n, err := c.client.Exists(ctx, "notif:"+subID+":"+renewalDate+":"+channel).Result()
	return n > 0, err
}

// SetSubsCache stores a user's serialised subscription list for 5 minutes.
func (c *Cache) SetSubsCache(ctx context.Context, userID string, data []byte) error {
	return c.client.Set(ctx, "subs:"+userID, data, 5*time.Minute).Err()
}

func (c *Cache) GetSubsCache(ctx context.Context, userID string) ([]byte, error) {
	return c.client.Get(ctx, "subs:"+userID).Bytes()
}

func (c *Cache) InvalidateSubsCache(ctx context.Context, userID string) error {
	return c.client.Del(ctx, "subs:"+userID).Err()
}
