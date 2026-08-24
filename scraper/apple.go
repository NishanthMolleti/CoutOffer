package scraper

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

type AppleScraper struct{}

// The search page is a server-rendered React app; results ship embedded as
// JSON.parse("...") inside a hydration script tag rather than via a public API.
var reAppleHydration = regexp.MustCompile(`JSON\.parse\("((?:[^"\\]|\\.)*)"\)`)

type appleLocation struct {
	Name        string `json:"name"`
	City        string `json:"city"`
	CountryName string `json:"countryName"`
}

type appleSearchResult struct {
	ReqID                   string          `json:"reqId"`
	PostingTitle            string          `json:"postingTitle"`
	TransformedPostingTitle string          `json:"transformedPostingTitle"`
	Locations               []appleLocation `json:"locations"`
}

type appleHydrationData struct {
	LoaderData struct {
		Search struct {
			SearchResults []appleSearchResult `json:"searchResults"`
		} `json:"search"`
	} `json:"loaderData"`
}

// Apple Careers has no per-company scoping (unlike the multi-tenant ATSes above),
// so slug is accepted for interface/schema consistency but unused.
func (a *AppleScraper) FetchJobs(slug string) ([]Job, error) {
	var all []Job
	for page := 1; page <= 25; page++ {
		jobs, err := fetchApplePage(page)
		if err != nil {
			return all, err
		}
		all = append(all, jobs...)
		if len(jobs) < 20 {
			break
		}
	}
	return all, nil
}

func fetchApplePage(page int) ([]Job, error) {
	u := fmt.Sprintf("https://jobs.apple.com/en-us/search?sort=newest&page=%d", page)

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("apple careers returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	m := reAppleHydration.FindSubmatch(body)
	if m == nil {
		return nil, fmt.Errorf("apple careers: hydration data not found")
	}

	var jsonText string
	if err := json.Unmarshal([]byte(`"`+string(m[1])+`"`), &jsonText); err != nil {
		return nil, fmt.Errorf("apple careers: unescape hydration data: %w", err)
	}

	var data appleHydrationData
	if err := json.Unmarshal([]byte(jsonText), &data); err != nil {
		return nil, fmt.Errorf("apple careers: parse hydration data: %w", err)
	}

	results := data.LoaderData.Search.SearchResults
	jobs := make([]Job, 0, len(results))
	for _, r := range results {
		jobs = append(jobs, Job{
			ID:       r.ReqID,
			Title:    r.PostingTitle,
			Location: appleLocationString(r.Locations),
			URL:      fmt.Sprintf("https://jobs.apple.com/en-us/details/%s/%s", r.ReqID, r.TransformedPostingTitle),
		})
	}
	return jobs, nil
}

func appleLocationString(locs []appleLocation) string {
	var parts []string
	for _, l := range locs {
		name := l.Name
		if name == "" {
			name = l.City
		}
		if l.CountryName != "" {
			name = strings.TrimSpace(strings.Trim(name+", "+l.CountryName, ", "))
		}
		if name != "" {
			parts = append(parts, name)
		}
	}
	return strings.Join(parts, "; ")
}
