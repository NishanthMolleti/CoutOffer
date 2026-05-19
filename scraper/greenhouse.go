package scraper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

type GreenhouseScraper struct{}

type greenhouseResponse struct {
	Jobs []struct {
		ID       int    `json:"id"`
		Title    string `json:"title"`
		Location struct {
			Name string `json:"name"`
		} `json:"location"`
		AbsoluteURL string `json:"absolute_url"`
	} `json:"jobs"`
}

func (g *GreenhouseScraper) FetchJobs(slug string) ([]Job, error) {
	url := fmt.Sprintf("https://boards-api.greenhouse.io/v1/boards/%s/jobs?content=true", slug)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("greenhouse returned %d for slug %q", resp.StatusCode, slug)
	}

	var data greenhouseResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	jobs := make([]Job, 0, len(data.Jobs))
	for _, j := range data.Jobs {
		jobs = append(jobs, Job{
			ID:       strconv.Itoa(j.ID),
			Title:    j.Title,
			Location: j.Location.Name,
			URL:      j.AbsoluteURL,
		})
	}
	return jobs, nil
}
