package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	githubAPIBase = "https://api.github.com"
	omnigraphRepo = "ModernRelay/omnigraph"
)

type PRMetadata struct {
	Number   int       `json:"number"`
	Title    string    `json:"title"`
	Body     string    `json:"body"`
	MergedAt time.Time `json:"merged_at"`
	HTMLURL  string    `json:"html_url"`
	User     struct {
		Login string `json:"login"`
	} `json:"user"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
}

type PRFile struct {
	Filename  string `json:"filename"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Patch     string `json:"patch"`
}

type PRData struct {
	Meta  PRMetadata
	Files []PRFile
	Diff  string
}

func fetchPR(prNumber int) (*PRData, error) {
	client := &http.Client{Timeout: 15 * time.Second}

	meta, err := fetchPRMeta(client, prNumber)
	if err != nil {
		return nil, fmt.Errorf("fetching PR metadata: %w", err)
	}

	files, err := fetchPRFiles(client, prNumber)
	if err != nil {
		return nil, fmt.Errorf("fetching PR files: %w", err)
	}

	diff, err := fetchPRDiff(client, prNumber)
	if err != nil {
		return nil, fmt.Errorf("fetching PR diff: %w", err)
	}

	return &PRData{
		Meta:  *meta,
		Files: files,
		Diff:  diff,
	}, nil
}

func fetchPRMeta(client *http.Client, prNumber int) (*PRMetadata, error) {
	url := fmt.Sprintf("%s/repos/%s/pulls/%d", githubAPIBase, omnigraphRepo, prNumber)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "omni-scribe/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(body))
	}

	var meta PRMetadata
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func fetchPRFiles(client *http.Client, prNumber int) ([]PRFile, error) {
	url := fmt.Sprintf("%s/repos/%s/pulls/%d/files", githubAPIBase, omnigraphRepo, prNumber)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "omni-scribe/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(body))
	}

	var files []PRFile
	if err := json.NewDecoder(resp.Body).Decode(&files); err != nil {
		return nil, err
	}
	return files, nil
}

func fetchPRDiff(client *http.Client, prNumber int) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/pulls/%d", githubAPIBase, omnigraphRepo, prNumber)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github.v3.diff")
	req.Header.Set("User-Agent", "omni-scribe/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Cap diff size to avoid hitting Gemini token limits
	limited := io.LimitReader(resp.Body, 32*1024)
	body, err := io.ReadAll(limited)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// summariseFiles builds a short human-readable list of changed files
// for inclusion in the prompt.
func summariseFiles(files []PRFile) string {
	if len(files) == 0 {
		return "No files changed."
	}
	summary := ""
	for _, f := range files {
		summary += fmt.Sprintf("- %s (%s, +%d -%d)\n", f.Filename, f.Status, f.Additions, f.Deletions)
	}
	return summary
}
