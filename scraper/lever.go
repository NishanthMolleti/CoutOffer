package scraper

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type LeverScraper struct{}

type leverPosting struct {
	ID         string `json:"id"`
	Text       string `json:"text"`
	Categories struct {
		Location string `json:"location"`
	} `json:"categories"`
	HostedURL string `json:"hostedUrl"`
}

func (l *LeverScraper) FetchJobs(slug string) ([]Job, error) {
	url := fmt.Sprintf("https://api.lever.co/v0/postings/%s", slug)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lever returned %d for slug %q", resp.StatusCode, slug)
	}

	var data []leverPosting
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	jobs := make([]Job, 0, len(data))
	for _, j := range data {
		jobs = append(jobs, Job{
			ID:       j.ID,
			Title:    j.Text,
			Location: j.Categories.Location,
			URL:      j.HostedURL,
		})
	}
	return jobs, nil
}
