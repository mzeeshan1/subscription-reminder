package models

import "time"

type Cycle string

const (
	CycleWeekly    Cycle = "weekly"
	CycleMonthly   Cycle = "monthly"
	CycleQuarterly Cycle = "quarterly"
	CycleYearly    Cycle = "yearly"
)

type Subscription struct {
	ID          string
	UserID      string
	Name        string
	Cost        float64
	Currency    string
	Cycle       Cycle
	NextRenewal time.Time
	Notes       string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (s Subscription) MonthlyCost() float64 {
	switch s.Cycle {
	case CycleWeekly:
		return s.Cost * 52 / 12
	case CycleMonthly:
		return s.Cost
	case CycleQuarterly:
		return s.Cost / 3
	case CycleYearly:
		return s.Cost / 12
	}
	return s.Cost
}

func (s Subscription) YearlyCost() float64 { return s.MonthlyCost() * 12 }

func (s Subscription) DaysUntilRenewal() int {
	return int(time.Until(s.NextRenewal).Hours() / 24)
}

func (s Subscription) RenewalDateStr() string {
	return s.NextRenewal.Format("2006-01-02")
}
