package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"subscription-manager/cache"
	"subscription-manager/models"
)

type SubscriptionHandler struct {
	db    *sql.DB
	cache *cache.Cache
}

func NewSubscriptionHandler(db *sql.DB, c *cache.Cache) *SubscriptionHandler {
	return &SubscriptionHandler{db: db, cache: c}
}

func (h *SubscriptionHandler) Dashboard(c *gin.Context) {
	userID := c.GetString("userID")

	subs, err := h.getUserSubs(c, userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "dashboard.html", gin.H{
			"Email": c.GetString("email"),
			"Error": "Could not load subscriptions.",
		})
		return
	}

	var monthlyTotal, yearlyTotal float64
	for _, s := range subs {
		monthlyTotal += s.MonthlyCost()
		yearlyTotal += s.YearlyCost()
	}

	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"Email":        c.GetString("email"),
		"Subs":         subs,
		"MonthlyTotal": fmt.Sprintf("%.2f", monthlyTotal),
		"YearlyTotal":  fmt.Sprintf("%.2f", yearlyTotal),
		"Count":        len(subs),
	})
}

func (h *SubscriptionHandler) AddPage(c *gin.Context) {
	c.HTML(http.StatusOK, "add.html", gin.H{"Email": c.GetString("email")})
}

func (h *SubscriptionHandler) Create(c *gin.Context) {
	userID := c.GetString("userID")

	sub, err := parseSubForm(c)
	if err != nil {
		c.HTML(http.StatusBadRequest, "add.html", gin.H{
			"Email": c.GetString("email"),
			"Error": err.Error(),
		})
		return
	}

	_, err = h.db.ExecContext(c.Request.Context(), `
		INSERT INTO subscriptions (user_id, name, cost, currency, cycle, next_renewal, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, userID, sub.Name, sub.Cost, sub.Currency, sub.Cycle, sub.NextRenewal.Format("2006-01-02"), sub.Notes)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "add.html", gin.H{
			"Email": c.GetString("email"),
			"Error": "Failed to save subscription.",
		})
		return
	}

	h.cache.InvalidateSubsCache(c.Request.Context(), userID)
	c.Redirect(http.StatusFound, "/")
}

func (h *SubscriptionHandler) EditPage(c *gin.Context) {
	userID := c.GetString("userID")
	id := c.Param("id")

	sub, err := h.getOne(c, id, userID)
	if err != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}

	c.HTML(http.StatusOK, "edit.html", gin.H{
		"Email": c.GetString("email"),
		"Sub":   sub,
	})
}

func (h *SubscriptionHandler) Update(c *gin.Context) {
	userID := c.GetString("userID")
	id := c.Param("id")

	sub, err := parseSubForm(c)
	if err != nil {
		existing, _ := h.getOne(c, id, userID)
		c.HTML(http.StatusBadRequest, "edit.html", gin.H{
			"Email": c.GetString("email"),
			"Sub":   existing,
			"Error": err.Error(),
		})
		return
	}

	_, err = h.db.ExecContext(c.Request.Context(), `
		UPDATE subscriptions
		SET name=$1, cost=$2, currency=$3, cycle=$4, next_renewal=$5, notes=$6, updated_at=NOW()
		WHERE id=$7 AND user_id=$8
	`, sub.Name, sub.Cost, sub.Currency, sub.Cycle, sub.NextRenewal.Format("2006-01-02"), sub.Notes, id, userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "edit.html", gin.H{
			"Email": c.GetString("email"),
			"Error": "Failed to update subscription.",
		})
		return
	}

	h.cache.InvalidateSubsCache(c.Request.Context(), userID)
	c.Redirect(http.StatusFound, "/")
}

func (h *SubscriptionHandler) Delete(c *gin.Context) {
	userID := c.GetString("userID")
	id := c.Param("id")
	h.db.ExecContext(c.Request.Context(), `DELETE FROM subscriptions WHERE id=$1 AND user_id=$2`, id, userID)
	h.cache.InvalidateSubsCache(c.Request.Context(), userID)
	c.Redirect(http.StatusFound, "/")
}

func (h *SubscriptionHandler) getUserSubs(c *gin.Context, userID string) ([]models.Subscription, error) {
	ctx := c.Request.Context()

	if cached, err := h.cache.GetSubsCache(ctx, userID); err == nil {
		var subs []models.Subscription
		if json.Unmarshal(cached, &subs) == nil {
			return subs, nil
		}
	}

	rows, err := h.db.QueryContext(ctx, `
		SELECT id, user_id, name, cost, currency, cycle, next_renewal, notes, created_at, updated_at
		FROM subscriptions WHERE user_id=$1 ORDER BY next_renewal ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []models.Subscription
	for rows.Next() {
		var s models.Subscription
		if err := rows.Scan(&s.ID, &s.UserID, &s.Name, &s.Cost, &s.Currency, &s.Cycle,
			&s.NextRenewal, &s.Notes, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if data, err := json.Marshal(subs); err == nil {
		h.cache.SetSubsCache(ctx, userID, data)
	}
	return subs, nil
}

func (h *SubscriptionHandler) getOne(c *gin.Context, id, userID string) (models.Subscription, error) {
	var s models.Subscription
	err := h.db.QueryRowContext(c.Request.Context(), `
		SELECT id, user_id, name, cost, currency, cycle, next_renewal, notes, created_at, updated_at
		FROM subscriptions WHERE id=$1 AND user_id=$2
	`, id, userID).Scan(&s.ID, &s.UserID, &s.Name, &s.Cost, &s.Currency, &s.Cycle,
		&s.NextRenewal, &s.Notes, &s.CreatedAt, &s.UpdatedAt)
	return s, err
}

func parseSubForm(c *gin.Context) (models.Subscription, error) {
	name := c.PostForm("name")
	costStr := c.PostForm("cost")
	currency := c.PostForm("currency")
	cycleStr := c.PostForm("cycle")
	renewalStr := c.PostForm("renewal")
	notes := c.PostForm("notes")

	if name == "" {
		return models.Subscription{}, fmt.Errorf("name is required")
	}
	if costStr == "" {
		return models.Subscription{}, fmt.Errorf("cost is required")
	}
	cost, err := strconv.ParseFloat(costStr, 64)
	if err != nil || cost <= 0 {
		return models.Subscription{}, fmt.Errorf("cost must be a positive number")
	}

	cycle := models.Cycle(cycleStr)
	switch cycle {
	case models.CycleWeekly, models.CycleMonthly, models.CycleQuarterly, models.CycleYearly:
	default:
		return models.Subscription{}, fmt.Errorf("invalid billing cycle")
	}

	renewal, err := time.Parse("2006-01-02", renewalStr)
	if err != nil {
		return models.Subscription{}, fmt.Errorf("invalid renewal date — use YYYY-MM-DD")
	}

	if currency == "" {
		currency = "USD"
	}

	return models.Subscription{
		Name: name, Cost: cost, Currency: currency,
		Cycle: cycle, NextRenewal: renewal, Notes: notes,
	}, nil
}
