package scheduler

import (
	"log"
	"strings"

	"gorm.io/gorm"

	"coutoffer/mailer"
	"coutoffer/models"
	"coutoffer/scraper"
)

func getScraper(atsType models.ATSType) scraper.Scraper {
	switch atsType {
	case models.ATSGreenhouse:
		return &scraper.GreenhouseScraper{}
	case models.ATSLever:
		return &scraper.LeverScraper{}
	case models.ATSWorkday:
		return &scraper.WorkdayScraper{}
	case models.ATSGoogle:
		return &scraper.GoogleScraper{}
	case models.ATSAshby:
		return &scraper.AshbyScraper{}
	default:
		return nil
	}
}

func CheckJobs(db *gorm.DB) {
	var subs []models.Subscription
	db.Where("active = ?", true).Find(&subs)

	m := mailer.New()

	for _, sub := range subs {
		s := getScraper(sub.ATSType)
		if s == nil {
			log.Printf("unknown ATS type %q for subscription %d", sub.ATSType, sub.ID)
			continue
		}

		jobs, err := s.FetchJobs(sub.ATSSlug)
		if err != nil {
			log.Printf("fetch error for %s: %v", sub.Company, err)
			continue
		}

		var newJobs []scraper.Job
		for _, job := range jobs {
			if !matchesRoles(job.Title, sub.RoleList()) {
				continue
			}
			var seen models.SeenJob
			if db.Where("subscription_id = ? AND job_id = ?", sub.ID, job.ID).First(&seen).Error == nil {
				continue
			}
			db.Create(&models.SeenJob{
				SubscriptionID: sub.ID,
				JobID:          job.ID,
				JobTitle:       job.Title,
				JobURL:         job.URL,
			})
			newJobs = append(newJobs, job)
		}

		if len(newJobs) == 0 {
			continue
		}
		if err := m.SendJobAlert(sub.Email, sub.Company, newJobs); err != nil {
			log.Printf("email error for %s -> %s: %v", sub.Company, sub.Email, err)
		} else {
			log.Printf("alerted %d new job(s) at %s -> %s", len(newJobs), sub.Company, sub.Email)
		}
	}
}

func matchesRoles(title string, roles []string) bool {
	if len(roles) == 0 {
		return true
	}
	titleLower := strings.ToLower(title)
	for _, role := range roles {
		if strings.Contains(titleLower, strings.ToLower(role)) {
			return true
		}
	}
	return false
}
