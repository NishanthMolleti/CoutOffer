package models

import (
	"strings"
	"time"
)

type ATSType string

const (
	ATSGreenhouse ATSType = "greenhouse"
	ATSLever      ATSType = "lever"
	ATSWorkday    ATSType = "workday"
	ATSGoogle     ATSType = "google"
	ATSAshby      ATSType = "ashby"
)

type Subscription struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Email     string    `gorm:"not null" json:"email"`
	Company   string    `gorm:"not null" json:"company"`
	ATSType   ATSType   `gorm:"not null" json:"ats_type"`
	ATSSlug   string    `gorm:"not null" json:"ats_slug"`
	Roles     string    `json:"roles"`
	Active    bool      `gorm:"default:true" json:"active"`
	CreatedAt time.Time `json:"created_at"`
	SeenJobs  []SeenJob `gorm:"foreignKey:SubscriptionID" json:"-"`
}

func (s *Subscription) RoleList() []string {
	var roles []string
	for _, r := range strings.Split(s.Roles, ",") {
		r = strings.TrimSpace(r)
		if r != "" {
			roles = append(roles, r)
		}
	}
	return roles
}

type SeenJob struct {
	ID             uint      `gorm:"primaryKey"`
	SubscriptionID uint      `gorm:"index"`
	JobID          string    `gorm:"index"`
	JobTitle       string
	JobURL         string
	SeenAt         time.Time `gorm:"autoCreateTime"`
}
