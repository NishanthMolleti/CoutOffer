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

type atsEntry struct{ ATSType, ATSSlug string }

var knownCompanies = map[string]atsEntry{
	// Greenhouse
	"anthropic":             {"greenhouse", "anthropic"},
	"databricks":            {"greenhouse", "databricks"},
	"jane street":           {"greenhouse", "janestreet"},
	"cloudflare":            {"greenhouse", "cloudflare"},
	"pure storage":          {"greenhouse", "purestorage"},
	"cockroach labs":        {"greenhouse", "cockroachlabs"},
	"stripe":                {"greenhouse", "stripe"},
	"xai":                   {"greenhouse", "xai"},
	"scale ai":              {"greenhouse", "scaleai"},
	"together ai":           {"greenhouse", "togetherai"},
	"canonical":             {"greenhouse", "canonical"},
	"elastic":               {"greenhouse", "elastic"},
	"mongodb":               {"greenhouse", "mongodb"},
	"grafana labs":          {"greenhouse", "grafanalabs"},
	"figma":                 {"greenhouse", "figma"},
	"airbnb":                {"greenhouse", "airbnb"},
	"dropbox":               {"greenhouse", "dropbox"},
	"vercel":                {"greenhouse", "vercel"},
	"rubrik":                {"greenhouse", "rubrik"},
	// Lever
	"netflix":               {"lever", "netflix"},
	"palantir":              {"lever", "palantir"},
	"mistral":               {"lever", "mistral"},
	"mistral ai":            {"lever", "mistral"},
	"atlassian":             {"lever", "atlassian"},
	// Workday
	"red hat":               {"workday", "redhat.wd5"},
	"nvidia":                {"workday", "nvidia.wd5/NVIDIAExternalCareerSite"},
	// Custom ATS — not on Greenhouse/Lever/Workday
	"google":                {},
	"meta":                  {},
	"amazon":                {},
	"amazon aws":            {},
	"microsoft":             {},
	"microsoft azure":       {},
	"apple":                 {},
	"uber":                  {},
	"shopify":               {},
	"openai":                {},
	"snowflake":             {},
	"amd":                   {},
	"broadcom":              {},
	"cisco":                 {},
	"juniper networks":      {},
	"dell technologies":     {},
	"netapp":                {},
	"oracle cloud infrastructure": {},
	"vmware":                {},
	"vmware by broadcom":    {},
	"hashicorp":             {},
	"docker":                {},
	"redis":                 {},
	"confluent":             {},
	"cohere":                {},
	"perplexity ai":         {},
	"perplexity":            {},
	"hudson river trading":  {},
	"citadel securities":    {},
}

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

	// fast path: hardcoded lookup
	if entry, ok := knownCompanies[strings.ToLower(company)]; ok {
		if entry.ATSType == "" {
			c.JSON(http.StatusOK, detectResult{Found: false})
			return
		}
		c.JSON(http.StatusOK, detectResult{ATSType: entry.ATSType, ATSSlug: entry.ATSSlug, Found: true})
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
