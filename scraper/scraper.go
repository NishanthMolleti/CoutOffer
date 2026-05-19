package scraper

type Job struct {
	ID       string
	Title    string
	Location string
	URL      string
}

type Scraper interface {
	FetchJobs(slug string) ([]Job, error)
}
