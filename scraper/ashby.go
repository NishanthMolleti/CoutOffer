package scraper

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type AshbyScraper struct{}

type ashbyResponse struct {
	Jobs []struct {
		ID       string `json:"id"`
		Title    string `json:"title"`
		Location string `json:"location"`
		JobURL   string `json:"jobUrl"`
	} `json:"jobs"`
}

func (a *AshbyScraper) FetchJobs(slug string) ([]Job, error) {
	resp, err := http.Get(fmt.Sprintf("https://api.ashbyhq.com/posting-api/job-board/%s", slug))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ashby returned %d for %s", resp.StatusCode, slug)
	}

	var data ashbyResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	var jobs []Job
	for _, j := range data.Jobs {
		jobs = append(jobs, Job{
			ID:       j.ID,
			Title:    j.Title,
			Location: j.Location,
			URL:      j.JobURL,
		})
	}
	return jobs, nil
}
