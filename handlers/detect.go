package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

type detectResult struct {
	ATSType string `json:"ats_type"`
	ATSSlug string `json:"ats_slug"`
	Found   bool   `json:"found"`
}

func (h *Handler) Detect(c *gin.Context) {
	company := strings.TrimSpace(c.Query("company"))
	if company == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "company required"})
		return
	}

	slugs := toSlugs(company)

	type candidate struct {
		atsType string
		atsSlug string
	}

	resultCh := make(chan candidate, 1)
	var once sync.Once
	var wg sync.WaitGroup

	for _, slug := range slugs {
		slug := slug

		wg.Add(1)
		go func() {
			defer wg.Done()
			if tryGreenhouse(slug) {
				once.Do(func() { resultCh <- candidate{"greenhouse", slug} })
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			if tryLever(slug) {
				once.Do(func() { resultCh <- candidate{"lever", slug} })
			}
		}()

		for _, instance := range []string{"wd1", "wd3", "wd5"} {
			instance := instance
			wdSlug := fmt.Sprintf("%s.%s", slug, instance)
			wg.Add(1)
			go func() {
				defer wg.Done()
				if tryWorkday(slug, instance) {
					once.Do(func() { resultCh <- candidate{"workday", wdSlug} })
				}
			}()
		}
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	if res, ok := <-resultCh; ok {
		c.JSON(http.StatusOK, detectResult{ATSType: res.atsType, ATSSlug: res.atsSlug, Found: true})
		return
	}
	c.JSON(http.StatusOK, detectResult{Found: false})
}

var nonAlpha = regexp.MustCompile(`[^a-z0-9]+`)

func toSlugs(company string) []string {
	lower := strings.ToLower(company)
	hyphen := strings.Trim(nonAlpha.ReplaceAllString(lower, "-"), "-")
	flat := nonAlpha.ReplaceAllString(lower, "")
	seen := map[string]bool{}
	var out []string
	for _, s := range []string{hyphen, flat} {
		if s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func tryGreenhouse(slug string) bool {
	resp, err := http.Get(fmt.Sprintf("https://boards-api.greenhouse.io/v1/boards/%s/jobs", slug))
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func tryLever(slug string) bool {
	resp, err := http.Get(fmt.Sprintf("https://api.lever.co/v0/postings/%s", slug))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}
	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return false
	}
	if ok, exists := data["ok"]; exists {
		if b, _ := ok.(bool); !b {
			return false
		}
	}
	return true
}

func tryWorkday(tenant, instance string) bool {
	url := fmt.Sprintf("https://%s.%s.myworkdayjobs.com/wday/cxs/%s/jobs/jobs", tenant, instance, tenant)
	body, _ := json.Marshal(map[string]interface{}{"appliedFacets": map[string]interface{}{}, "limit": 1, "offset": 0, "searchText": ""})
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
