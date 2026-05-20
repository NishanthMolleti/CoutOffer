package scraper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type WorkdayScraper struct{}

type workdayRequest struct {
	AppliedFacets map[string]interface{} `json:"appliedFacets"`
	Limit         int                    `json:"limit"`
	Offset        int                    `json:"offset"`
	SearchText    string                 `json:"searchText"`
}

type workdayResponse struct {
	JobPostings []struct {
		Title         string `json:"title"`
		ExternalPath  string `json:"externalPath"`
		LocationsText string `json:"locationsText"`
		ReqId         string `json:"reqId"`
	} `json:"jobPostings"`
	Total int `json:"total"`
}

// slug format: "{tenant}.wd{N}" or "{tenant}.wd{N}/BoardName"
// e.g. "redhat.wd5" or "nvidia.wd5/NVIDIAExternalCareerSite"
func (w *WorkdayScraper) FetchJobs(slug string) ([]Job, error) {
	board := "jobs"
	if idx := strings.Index(slug, "/"); idx != -1 {
		board = slug[idx+1:]
		slug = slug[:idx]
	}
	parts := strings.SplitN(slug, ".", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("workday slug must be '{tenant}.wd{N}[/Board]', got %q", slug)
	}
	tenant := parts[0]
	instance := parts[1]

	var all []Job
	limit := 20
	offset := 0

	for {
		body, err := json.Marshal(workdayRequest{
			AppliedFacets: map[string]interface{}{},
			Limit:         limit,
			Offset:        offset,
			SearchText:    "",
		})
		if err != nil {
			return nil, err
		}

		url := fmt.Sprintf("https://%s.%s.myworkdayjobs.com/wday/cxs/%s/%s/jobs", tenant, instance, tenant, board)
		req, err := http.NewRequest("POST", url, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("workday returned %d for %s", resp.StatusCode, slug)
		}

		var data workdayResponse
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return nil, err
		}

		baseURL := fmt.Sprintf("https://%s.%s.myworkdayjobs.com/en-US/%s", tenant, instance, board)
		for _, j := range data.JobPostings {
			all = append(all, Job{
				ID:       j.ReqId,
				Title:    j.Title,
				Location: j.LocationsText,
				URL:      baseURL + j.ExternalPath,
			})
		}

		offset += limit
		if offset >= data.Total {
			break
		}
	}

	return all, nil
}
