package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const geminiEndpoint = "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent"

// omnigraphContext describes Omnigraph for prompts targeting ModernRelay/omnigraph.
const omnigraphContext = `
Omnigraph is a typed property graph database built on Lance (a columnar storage format).
Key concepts:
- Schema-first: every node and edge has a typed schema defined in .pg files.
- Git-style workflows: the database supports branches, commits, merges, and transactional runs,
  exactly like version-controlling code but for graph data.
- Storage: runs equally on a local directory or an s3:// URI (S3-native via RustFS or AWS S3).
- Query language: typed queries (.gq files) with traversal, text, fuzzy, BM25, vector, and RRF search.
- Mutations: typed change queries for inserting, updating, and deleting nodes and edges.
- Server: Axum-based HTTP server exposing /read, /change, /schema/apply, /ingest, /branches,
  /commits, /runs, /snapshot, /export, and /healthz routes.
- Authorization: Cedar-based policy-as-code for server-side access control.
- Written in Rust, workspace crates: omnigraph-compiler, omnigraph, omnigraph-cli, omnigraph-server.
- Target users: developers building knowledge graphs, team context graphs, research tracking systems,
  and private self-hosted graph backends for AI agents.
`

// omniscribeContext describes omni-scribe itself for prompts targeting Taiwrash/omni-scribe.
const omniscribeContext = `
omni-scribe is an open-source CLI tool and web service that automatically generates AI-powered
documentation for GitHub pull requests using the Gemini API.
Key concepts:
- CLI commands: 'generate' fetches a PR from GitHub and calls Gemini to produce a changelog and
  conceptual explainer, saving results to JSON (locally or in Google Cloud Storage).
- 'serve' starts an HTTP server that renders the stored JSON docs as styled HTML pages.
- Storage backends: supports local filesystem (./data/prs/) or Google Cloud Storage (GCS_BUCKET env).
- GitHub Actions integration: can be dropped into any repo as a workflow that runs on PR merge,
  calls the Gemini API, stores docs in GCS, and posts a comment linking to the rendered page.
- Deployment: ships as a Docker container suitable for Google Cloud Run.
- Written in Go; intentionally dependency-light outside the GCS client library.
- Target users: open-source maintainers and engineering teams who want automated PR documentation.
`

// repoContext returns the project description and technical writer persona to inject into the
// Gemini prompt, based on whichever repository is currently active.
func repoContext() (projectContext, persona, audience string) {
	switch githubRepo {
	case "ModernRelay/omnigraph":
		return omnigraphContext,
			"You are a technical writer for Omnigraph, an open source typed property graph database.",
			"a developer who uses Omnigraph"
	case "Taiwrash/omni-scribe":
		return omniscribeContext,
			"You are a technical writer for omni-scribe, an open source AI-powered PR documentation tool.",
			"a developer who uses or contributes to omni-scribe"
	default:
		genericCtx := fmt.Sprintf("`%s` is a software project hosted on GitHub.", githubRepo)
		return genericCtx,
			fmt.Sprintf("You are a technical writer for the `%s` project.", githubRepo),
			"a developer who works with this project"
	}
}

// GeneratedDocs holds both outputs Gemini produces for a PR.
type GeneratedDocs struct {
	Changelog string `json:"changelog"`
	Explainer string `json:"explainer"`
}

type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func generateDocs(pr *PRData) (*GeneratedDocs, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable is not set")
	}

	prompt := buildPrompt(pr)

	reqBody := geminiRequest{
		Contents: []geminiContent{
			{Parts: []geminiPart{{Text: prompt}}},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshalling request: %w", err)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	url := fmt.Sprintf("%s?key=%s", geminiEndpoint, apiKey)

	var resp *http.Response
	maxAttempts := 5
	backoff := 2 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err = client.Post(url, "application/json", bytes.NewReader(body))
		if err == nil {
			if resp.StatusCode == http.StatusOK {
				break
			}
			if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable {
				respBody, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				fmt.Fprintf(os.Stderr, "  ⚠ Gemini API returned %d (attempt %d/%d): %s. Retrying in %v...\n", resp.StatusCode, attempt, maxAttempts, strings.TrimSpace(string(respBody)), backoff)
				time.Sleep(backoff)
				backoff *= 2
				continue
			}
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("Gemini API returned %d: %s", resp.StatusCode, string(respBody))
		}

		fmt.Fprintf(os.Stderr, "  ⚠ Network error calling Gemini API (attempt %d/%d): %v. Retrying in %v...\n", attempt, maxAttempts, err, backoff)
		time.Sleep(backoff)
		backoff *= 2
	}

	if err != nil {
		return nil, fmt.Errorf("calling Gemini API after %d attempts: %w", maxAttempts, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Gemini API returned %d after %d attempts", resp.StatusCode, maxAttempts)
	}
	defer resp.Body.Close()

	var gemResp geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&gemResp); err != nil {
		return nil, fmt.Errorf("decoding Gemini response: %w", err)
	}

	if len(gemResp.Candidates) == 0 || len(gemResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("Gemini returned empty response")
	}

	raw := gemResp.Candidates[0].Content.Parts[0].Text
	return parseGeminiOutput(raw)
}

func buildPrompt(pr *PRData) string {
	fileSummary := summariseFiles(pr.Files)

	// Truncate diff if very long to stay within token budget
	diff := pr.Diff
	if len(diff) > 20000 {
		diff = diff[:20000] + "\n\n[diff truncated for length]"
	}

	projectContext, persona, audience := repoContext()

	return fmt.Sprintf(`%s

Here is context about the project so you understand what it does:
%s

You have been given a merged pull request. Your job is to produce two documents.

---
PULL REQUEST DETAILS

Number: #%d
Title: %s
Author: %s
URL: %s

Description:
%s

Files changed:
%s

Diff:
%s
---

Produce exactly two sections, separated by the exact markers shown below. Do not add any other text outside these sections.

<<<CHANGELOG>>>
Write a user-facing changelog entry for this PR. One to three sentences. Plain prose.
No bullet points. No markdown headers. Write as if addressing %s who did not read the PR.
Explain what changed and why it matters to them. Do not mention the PR number or the author name.
<<<END_CHANGELOG>>>

<<<EXPLAINER>>>
Write a conceptual explainer for this change. Three to five paragraphs. Target
%s who encountered this change for the first time.
Explain what the changed component does in the project, why this change was made, and
what it means for someone using or building on it. Use plain prose. No bullet
points. No markdown headers inside paragraphs. You may use a single short heading per
paragraph if it helps navigation.
<<<END_EXPLAINER>>>
`,
		persona,
		projectContext,
		pr.Meta.Number,
		pr.Meta.Title,
		pr.Meta.User.Login,
		pr.Meta.HTMLURL,
		pr.Meta.Body,
		fileSummary,
		diff,
		audience,
		audience,
	)
}

func parseGeminiOutput(raw string) (*GeneratedDocs, error) {
	changelog := extractBetween(raw, "<<<CHANGELOG>>>", "<<<END_CHANGELOG>>>")
	explainer := extractBetween(raw, "<<<EXPLAINER>>>", "<<<END_EXPLAINER>>>")

	if changelog == "" {
		return nil, fmt.Errorf("could not parse CHANGELOG section from Gemini output")
	}
	if explainer == "" {
		return nil, fmt.Errorf("could not parse EXPLAINER section from Gemini output")
	}

	return &GeneratedDocs{
		Changelog: strings.TrimSpace(changelog),
		Explainer: strings.TrimSpace(explainer),
	}, nil
}

func extractBetween(s, start, end string) string {
	startIdx := strings.Index(s, start)
	if startIdx == -1 {
		return ""
	}
	startIdx += len(start)
	endIdx := strings.Index(s[startIdx:], end)
	if endIdx == -1 {
		return ""
	}
	return s[startIdx : startIdx+endIdx]
}
