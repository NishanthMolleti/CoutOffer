package scraper

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type GoogleScraper struct{}

var (
	// Pairs job title with its nearest location span in one pass.
	reGEntry = regexp.MustCompile(`(?s)<h3[^>]*class="[^"]*QJPWVe[^"]*"[^>]*>([^<]+)</h3>.*?<span[^>]*class="[^"]*r0wTof[^"]*"[^>]*>([^<]+)</span>`)
	// Job IDs appear as jobId\u003d{id}\u0026 in embedded JS data (unicode-escaped = and &).
	reGJobID = regexp.MustCompile(`jobId(?:\\u003d|=)([\w%+/_-]+?)(?:\\u0026|&)`)
)

// slug = comma-separated Google entities e.g. "Google" or "Google,YouTube,DeepMind"
func (g *GoogleScraper) FetchJobs(slug string) ([]Job, error) {
	companies := strings.Split(slug, ",")
	seen := map[string]bool{}
	var all []Job

	for _, company := range companies {
		company = strings.TrimSpace(company)
		if company == "" {
			continue
		}
		jobs, err := fetchGoogleCompanyJobs(company)
		if err != nil {
			continue
		}
		for _, j := range jobs {
			if !seen[j.ID] {
				seen[j.ID] = true
				all = append(all, j)
			}
		}
	}
	return all, nil
}

func fetchGoogleCompanyJobs(company string) ([]Job, error) {
	var all []Job
	for start := 0; start < 500; start += 20 {
		jobs, hasMore, err := fetchGooglePage(company, start)
		if err != nil {
			return all, err
		}
		all = append(all, jobs...)
		if !hasMore || len(jobs) == 0 {
			break
		}
	}
	return all, nil
}

func fetchGooglePage(company string, start int) (jobs []Job, hasMore bool, err error) {
	u := "https://careers.google.com/jobs/results/?" + url.Values{
		"company": {company},
		"num":     {"20"},
		"start":   {fmt.Sprintf("%d", start)},
	}.Encode()

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("google careers returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, false, err
	}
	html := string(body)

	// Extract job IDs from embedded JS (appear as unicode-escaped sequences in AF_initDataCallback).
	var uniqueIDs []string
	seenID := map[string]bool{}
	for _, m := range reGJobID.FindAllStringSubmatch(html, -1) {
		if !seenID[m[1]] {
			seenID[m[1]] = true
			uniqueIDs = append(uniqueIDs, m[1])
		}
	}

	entries := reGEntry.FindAllStringSubmatch(html, -1)

	n := len(entries)
	if len(uniqueIDs) < n {
		n = len(uniqueIDs)
	}

	for i := 0; i < n; i++ {
		jobID := uniqueIDs[i]
		jobs = append(jobs, Job{
			ID:       jobID,
			Title:    strings.TrimSpace(entries[i][1]),
			Location: strings.TrimSpace(entries[i][2]),
			URL:      "https://www.google.com/about/careers/applications/signin?jobId=" + jobID,
		})
	}

	hasMore = len(entries) >= 20
	return jobs, hasMore, nil
}
