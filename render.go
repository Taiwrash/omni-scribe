package main

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"
)

func renderHTML(doc *StoredDoc) string {
	changelog := parseMarkdown(html.EscapeString(doc.Docs.Changelog))
	explainer := renderExplainer(doc.Docs.Explainer)
	category := prCategory(doc.PRTitle)
	readTime := estimateReadingTime(doc.Docs.Changelog, doc.Docs.Explainer)
	heroSVG := renderHeroSVG(doc.PRNumber, doc.PRTitle)
	currentYear := time.Now().Year()

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Pull Request #%d: %s — Omni-Scribe</title>
  <style>
    *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

    :root {
      --bg:           #ffffff;
      --border:       #e5e5e5;
      --border-dark:  #111111;
      --text:         #111111;
      --muted:        #666666;
      --max:          760px;
    }

    body {
      background: var(--bg);
      color: var(--text);
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif;
      font-size: 15px;
      line-height: 1.7;
      padding: 4rem 2rem;
      -webkit-font-smoothing: antialiased;
    }

    .wrap {
      max-width: var(--max);
      margin: 0 auto;
    }

    /* top bar */
    .topbar {
      display: flex;
      align-items: center;
      justify-content: space-between;
      margin-bottom: 3rem;
      padding-bottom: 1.25rem;
      border-bottom: 1px solid var(--border-dark);
    }
    .brand {
      font-size: 0.85rem;
      font-weight: 700;
      letter-spacing: 0.15em;
      text-transform: uppercase;
      color: var(--text);
    }
    .pr-badge {
      font-size: 0.75rem;
      font-weight: 600;
      letter-spacing: 0.05em;
      text-transform: uppercase;
      color: var(--muted);
      border: 1px solid var(--border);
      padding: 0.25rem 0.6rem;
    }

    /* back link */
    .back-link {
      display: inline-flex;
      align-items: center;
      gap: 0.5rem;
      font-size: 0.75rem;
      font-weight: 700;
      color: var(--muted);
      text-decoration: none;
      text-transform: uppercase;
      letter-spacing: 0.08em;
      margin-bottom: 2.5rem;
      transition: color 0.1s;
    }
    .back-link:hover {
      color: var(--text);
    }

    /* header */
    .pr-header {
      margin-bottom: 3.5rem;
      padding-bottom: 1rem;
    }
    .tag {
      display: inline-block;
      font-size: 0.7rem;
      font-weight: 700;
      letter-spacing: 0.1em;
      text-transform: uppercase;
      padding: 0.25rem 0.6rem;
      border: 1.5px solid var(--border-dark);
      color: var(--text);
      margin-bottom: 1.25rem;
    }
    .meta-time {
      font-size: 0.8rem;
      color: var(--muted);
      font-family: SFMono-Regular, Consolas, monospace;
      font-weight: 600;
      text-transform: uppercase;
      letter-spacing: 0.05em;
      margin-bottom: 0.75rem;
    }
    h1 {
      font-size: 2.2rem;
      font-weight: 800;
      line-height: 1.2;
      color: var(--text);
      letter-spacing: -0.03em;
      margin-bottom: 1rem;
    }
    .meta-author {
      font-size: 0.85rem;
      color: var(--muted);
      display: flex;
      align-items: center;
      flex-wrap: wrap;
      gap: 0.75rem;
      margin-top: 1rem;
      margin-bottom: 2rem;
    }
    .meta-author a {
      color: var(--text);
      text-decoration: underline;
      text-underline-offset: 4px;
      font-weight: 600;
      transition: color 0.1s;
    }
    .meta-author a:hover {
      color: var(--muted);
    }
    .dot-separator {
      color: var(--border);
    }
    .hero-image {
      margin-top: 2rem;
      border: 1px solid var(--border-dark);
      background: #111111;
      width: 100%%;
      overflow: hidden;
    }

    /* section layout */
    .section {
      margin-bottom: 3.5rem;
    }
    .section-label {
      font-size: 0.75rem;
      font-weight: 700;
      letter-spacing: 0.15em;
      text-transform: uppercase;
      color: var(--muted);
      margin-bottom: 1.25rem;
      border-bottom: 1px solid var(--border-dark);
      padding-bottom: 0.4rem;
    }
    .changelog-text {
      font-size: 1.05rem;
      line-height: 1.65;
      color: var(--text);
      border-left: 2px solid var(--border-dark);
      padding-left: 1.5rem;
      font-weight: 500;
    }

    /* explainer */
    .explainer-body h2 {
      font-size: 0.95rem;
      font-weight: 700;
      letter-spacing: 0.05em;
      text-transform: uppercase;
      color: var(--text);
      margin: 2rem 0 0.5rem;
    }
    .explainer-body p {
      color: var(--text);
      margin-bottom: 1.25rem;
    }
    .explainer-body p:last-child {
      margin-bottom: 0;
    }

    /* footer */
    footer {
      margin-top: 6rem;
      padding-top: 2rem;
      border-top: 1px solid var(--border);
      font-size: 0.75rem;
      color: var(--muted);
      display: flex;
      justify-content: space-between;
      align-items: center;
      flex-wrap: wrap;
      gap: 1.5rem;
    }
    .footer-left, .footer-right {
      display: flex;
      align-items: center;
      flex-wrap: wrap;
      gap: 1.25rem;
    }
    footer a {
      color: var(--muted);
      text-decoration: none;
      transition: color 0.1s;
    }
    footer a:hover {
      color: var(--text);
      text-decoration: underline;
      text-underline-offset: 3px;
    }
    .dot {
      display: inline-block;
      width: 8px; height: 8px;
      background: var(--text);
      margin-right: 0.5rem;
      vertical-align: middle;
    }
  </style>
</head>
<body>
<div class="wrap">

  <div class="topbar">
    <span class="brand"><span class="dot"></span>Omni-Scribe</span>
    <span class="pr-badge">omnigraph / Pull Request #%d</span>
  </div>

  <a href="/" class="back-link">&larr; Back to pull requests</a>

  <div class="pr-header">
    <div class="tag">%s</div>
    <div class="meta-time">%s &bull; %d min read</div>
    <h1>%s</h1>
    <div class="meta-author">
      <span>By <strong>%s</strong></span>
      <span class="dot-separator">&bull;</span>
      <a href="%s" target="_blank" rel="noopener">View on GitHub &rarr;</a>
    </div>
    %s
  </div>

  <div class="section">
    <div class="section-label">What changed</div>
    <div class="changelog-text">%s</div>
  </div>

  <div class="section">
    <div class="section-label">Conceptual explainer</div>
    <div class="explainer-body">%s</div>
  </div>

  <footer>
    <div class="footer-left">
      <span>&copy; %d Omni-Scribe</span>
      <a href="https://github.com/Taiwrash/omni-scribe" target="_blank" rel="noopener">Source Code</a>
    </div>
    <div class="footer-right">
      <span>Built for <a href="https://github.com/ModernRelay/omnigraph" target="_blank" rel="noopener">Omnigraph</a></span>
    </div>
  </footer>

</div>
</body>
</html>`,
		doc.PRNumber,
		html.EscapeString(doc.PRTitle),
		doc.PRNumber,
		category,
		doc.GeneratedAt.Format("January 02, 2006"),
		readTime,
		html.EscapeString(doc.PRTitle),
		html.EscapeString(doc.PRAuthor),
		html.EscapeString(doc.PRURL),
		heroSVG,
		changelog,
		explainer,
		currentYear,
	)
}

// renderExplainer converts the plain text explainer into HTML paragraphs.
// Lines starting with a capital word followed by a colon become headings.
func renderExplainer(text string) string {
	var sb strings.Builder
	paragraphs := strings.Split(strings.TrimSpace(text), "\n\n")
	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		// Check if this paragraph starts with a short heading line
		lines := strings.SplitN(para, "\n", 2)
		first := strings.TrimSpace(lines[0])
		if len(lines) > 1 && len(first) < 80 && strings.HasSuffix(first, ":") {
			sb.WriteString(fmt.Sprintf("<h2>%s</h2>\n", parseMarkdown(html.EscapeString(strings.TrimSuffix(first, ":")))))
			rest := strings.TrimSpace(lines[1])
			if rest != "" {
				sb.WriteString(fmt.Sprintf("<p>%s</p>\n", parseMarkdown(html.EscapeString(rest))))
			}
		} else {
			sb.WriteString(fmt.Sprintf("<p>%s</p>\n", parseMarkdown(html.EscapeString(para))))
		}
	}
	return sb.String()
}

// renderIndex produces the homepage listing all generated PR docs.
func renderIndex(docs []*StoredDoc) string {
	currentYear := time.Now().Year()
	var rows strings.Builder
	if len(docs) == 0 {
		rows.WriteString(`<tr><td colspan="4" style="text-align:center;color:var(--muted);padding:3rem;font-style:italic;">No docs generated yet. Run <code>omni-scribe generate --pr &lt;number&gt;</code> to get started.</td></tr>`)
	}
	for _, doc := range docs {
		rows.WriteString(fmt.Sprintf(`<tr>
      <td style="font-weight:700;"><a href="/pr/%d">#%d</a></td>
      <td>%s</td>
      <td>%s</td>
      <td style="color:var(--muted); font-size:0.85rem;">%s</td>
    </tr>`,
			doc.PRNumber, doc.PRNumber,
			html.EscapeString(doc.PRTitle),
			html.EscapeString(doc.PRAuthor),
			doc.GeneratedAt.Format("2006.01.02 — 15:04 MST"),
		))
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Omni-Scribe — Omnigraph PR Documentation</title>
  <meta name="description" content="AI-generated documentation for Omnigraph pull requests, powered by Gemini.">
  <style>
    *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

    :root {
      --bg:           #ffffff;
      --border:       #e5e5e5;
      --border-dark:  #111111;
      --text:         #111111;
      --muted:        #666666;
      --radius:       0px;
      --max:          1140px;
    }

    body {
      background: var(--bg);
      color: var(--text);
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif;
      font-size: 14px;
      line-height: 1.6;
      padding: 4rem 2rem;
      -webkit-font-smoothing: antialiased;
    }

    .wrap {
      max-width: var(--max);
      margin: 0 auto;
    }

    .topbar {
      display: flex;
      align-items: center;
      justify-content: space-between;
      margin-bottom: 4rem;
      padding-bottom: 1.25rem;
      border-bottom: 1px solid var(--border-dark);
    }
    .brand {
      font-size: 0.85rem;
      font-weight: 700;
      letter-spacing: 0.15em;
      text-transform: uppercase;
      color: var(--text);
    }
    .dot {
      display: inline-block;
      width: 8px; height: 8px;
      background: var(--text);
      margin-right: 0.5rem;
      vertical-align: middle;
    }
    .count-badge {
      font-size: 0.75rem;
      font-weight: 600;
      letter-spacing: 0.05em;
      text-transform: uppercase;
      color: var(--muted);
      border: 1px solid var(--border);
      padding: 0.25rem 0.6rem;
    }

    .page-header {
      margin-bottom: 4rem;
      padding-bottom: 1.5rem;
      border-bottom: 1px solid var(--border);
    }
    h1 {
      font-size: 2.2rem;
      font-weight: 700;
      line-height: 1.15;
      color: var(--text);
      letter-spacing: -0.03em;
      margin-bottom: 0.75rem;
    }
    .subtitle {
      font-size: 1rem;
      color: var(--muted);
      max-width: 600px;
    }
    details {
      border-bottom: 1px solid var(--border);
      margin-bottom: 1rem;
    }
    details[open] {
      padding-bottom: 1.5rem;
    }
    summary {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 1.25rem 0;
      font-size: 0.85rem;
      font-weight: 700;
      letter-spacing: 0.15em;
      text-transform: uppercase;
      color: var(--text);
      cursor: pointer;
      list-style: none;
    }
    summary::-webkit-details-marker {
      display: none;
    }
    summary::after {
      content: "+";
      font-size: 1.1rem;
      font-weight: 400;
      color: var(--muted);
    }
    details[open] summary::after {
      content: "—";
    }
    .details-content {
      padding-top: 1.25rem;
    }
    @keyframes slideDown {
      from {
        opacity: 0;
        transform: translateY(-8px);
      }
      to {
        opacity: 1;
        transform: translateY(0);
      }
    }
    details[open] .details-content {
      animation: slideDown 0.2s cubic-bezier(0.16, 1, 0.3, 1) forwards;
    }

    .section {
      margin-bottom: 2rem;
    }
    .section-heading {
      font-size: 0.85rem;
      font-weight: 700;
      letter-spacing: 0.15em;
      text-transform: uppercase;
      color: var(--text);
      margin-bottom: 1.5rem;
      border-bottom: 1px solid var(--border-dark);
      padding-bottom: 0.5rem;
    }
    .description {
      font-size: 0.9rem;
      color: var(--muted);
      margin-bottom: 1.25rem;
      line-height: 1.5;
    }

    table {
      width: 100%%;
      border-collapse: collapse;
      font-size: 0.9rem;
    }
    thead th {
      text-align: left;
      padding: 1rem 0;
      font-size: 0.75rem;
      font-weight: 700;
      letter-spacing: 0.15em;
      text-transform: uppercase;
      color: var(--muted);
      border-bottom: 1px solid var(--border-dark);
    }
    tbody td {
      padding: 1.25rem 0;
      border-bottom: 1px solid var(--border);
      color: var(--text);
      vertical-align: top;
    }
    tbody tr:last-child td {
      border-bottom: none;
    }
    tbody tr:hover td {
      color: var(--muted);
    }
    a {
      color: var(--text);
      text-decoration: underline;
      text-underline-offset: 4px;
      text-decoration-thickness: 1px;
      font-weight: 600;
      transition: color 0.1s;
    }
    a:hover {
      color: var(--muted);
    }
    code {
      font-family: SFMono-Regular, Consolas, "Liberation Mono", Menlo, monospace;
      font-size: 0.85em;
      background: #fafafa;
      border: 1px solid var(--border);
      padding: 0.15em 0.4em;
    }

    .pre-container {
      position: relative;
    }
    pre {
      background: #fafafa;
      border: 1px solid var(--border);
      padding: 1.5rem;
      overflow-x: auto;
      font-family: SFMono-Regular, Consolas, "Liberation Mono", Menlo, monospace;
      font-size: 0.85rem;
      line-height: 1.5;
      color: var(--text);
      margin-bottom: 1.5rem;
    }
    .copy-btn {
      position: absolute;
      top: 12px;
      right: 12px;
      background: #ffffff;
      border: 1px solid var(--border-dark);
      color: var(--text);
      font-size: 0.7rem;
      font-weight: 700;
      letter-spacing: 0.05em;
      text-transform: uppercase;
      padding: 0.4rem 0.8rem;
      cursor: pointer;
      border-radius: 0px;
      transition: background 0.1s, color 0.1s;
    }
    .copy-btn:hover {
      background: var(--text);
      color: #ffffff;
    }

    footer {
      margin-top: 6rem;
      padding-top: 2rem;
      border-top: 1px solid var(--border);
      font-size: 0.75rem;
      color: var(--muted);
      display: flex;
      justify-content: space-between;
      align-items: center;
      flex-wrap: wrap;
      gap: 1.5rem;
    }
    .footer-left, .footer-right {
      display: flex;
      align-items: center;
      flex-wrap: wrap;
      gap: 1.25rem;
    }
    footer a {
      color: var(--muted);
      text-decoration: none;
      transition: color 0.1s;
    }
    footer a:hover {
      color: var(--text);
      text-decoration: underline;
      text-underline-offset: 3px;
    }
  </style>
</head>
<body>
<div class="wrap">

  <div class="topbar">
    <span class="brand"><span class="dot"></span>Omni-Scribe</span>
    <span class="count-badge">%d docs generated</span>
  </div>

  <div class="page-header">
    <h1>Omnigraph Pull Request Documentation</h1>
    <p class="subtitle">Auto-generated conceptual explainers & changelogs for pull requests in your project's voice.</p>
  </div>

  <!-- Primary Section: Full-Width Recent Pull Requests -->
  <div class="section">
    <div class="section-heading">Recent Pull Requests</div>
    <table>
      <thead>
        <tr>
          <th style="width: 25%%;">Pull Request</th>
          <th>Title</th>
          <th style="width: 20%%;">Author</th>
          <th style="width: 25%%;">Generated</th>
        </tr>
      </thead>
      <tbody>
        %s
      </tbody>
    </table>
  </div>

  <!-- Collapsible Technical Reference / Setup Sections -->
  <div style="margin-top: 4.5rem; margin-bottom: 2rem;">
    <details>
      <summary>Pipeline Integration</summary>
      <div class="details-content">
        <p class="description">To automatically generate documentation when a pull request is merged, save this workflow to <code>.github/workflows/omni-scribe.yml</code> in your main repository:</p>
        <div class="pre-container">
          <button onclick="copyCode(this)" class="copy-btn">Copy</button>
          <pre><code id="integration-code">name: Generate PR Docs
on:
  pull_request:
    types: [closed]

jobs:
  docs:
    if: github.event.pull_request.merged == true
    runs-on: ubuntu-latest
    steps:
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.22"

      - name: Install omni-scribe
        run: go install github.com/Taiwrash/omni-scribe@latest

      - name: Authenticate GCP
        if: ${{ secrets.GCP_SA_KEY != '' }}
        uses: google-github-actions/auth@v2
        with:
          credentials_json: ${{ secrets.GCP_SA_KEY }}

      - name: Generate PR Docs
        env:
          GEMINI_API_KEY: ${{ secrets.GEMINI_API_KEY }}
          GCS_BUCKET: ${{ vars.GCS_BUCKET || 'omni-scribe-docs' }}
        run: omni-scribe generate --pr ${{ github.event.pull_request.number }}</code></pre>
        </div>
      </div>
    </details>

    <details>
      <summary>Configuration Reference</summary>
      <div class="details-content">
        <p class="description">Configure local execution or your Cloud Run container using these environment variables:</p>
        <table>
          <thead>
            <tr>
              <th style="width: 35%%;">Variable</th>
              <th>Description</th>
              <th style="width: 30%%;">Default</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td><code>GEMINI_API_KEY</code></td>
              <td>Required for generation. Your Gemini API Key.</td>
              <td>—</td>
            </tr>
            <tr>
              <td><code>GCS_BUCKET</code></td>
              <td>Google Cloud Storage bucket name for multi-mode storage.</td>
              <td>— (local filesystem)</td>
            </tr>
            <tr>
              <td><code>DATA_DIR</code></td>
              <td>Directory path for local filesystem storage.</td>
              <td><code>./data</code></td>
            </tr>
            <tr>
              <td><code>PORT</code></td>
              <td>Port number for the HTTP server.</td>
              <td><code>8080</code></td>
            </tr>
          </tbody>
        </table>
      </div>
    </details>
  </div>

  <script>
    function copyCode(btn) {
      const code = document.getElementById('integration-code').innerText;
      navigator.clipboard.writeText(code).then(() => {
        btn.innerText = 'Copied';
        setTimeout(() => {
          btn.innerText = 'Copy';
        }, 2000);
      }).catch(err => {
        console.error('Failed to copy: ', err);
      });
    }
  </script>

  <footer>
    <div class="footer-left">
      <span>&copy; %d Omni-Scribe</span>
      <a href="https://github.com/Taiwrash/omni-scribe" target="_blank" rel="noopener">Source Code</a>
    </div>
    <div class="footer-right">
      <span>Built for <a href="https://github.com/ModernRelay/omnigraph" target="_blank" rel="noopener">Omnigraph</a></span>
    </div>
  </footer>

</div>
</body>
</html>`, len(docs), rows.String(), currentYear)
}


// prCategory tries to categorize the PR based on its title prefix.
func prCategory(title string) string {
	title = strings.ToLower(strings.TrimSpace(title))
	if strings.HasPrefix(title, "feat") {
		return "Feature"
	}
	if strings.HasPrefix(title, "fix") {
		return "Bugfix"
	}
	if strings.HasPrefix(title, "docs") {
		return "Documentation"
	}
	if strings.HasPrefix(title, "chore") {
		return "Chore"
	}
	if strings.HasPrefix(title, "refactor") {
		return "Refactor"
	}
	if strings.HasPrefix(title, "test") {
		return "Test"
	}
	if strings.HasPrefix(title, "ci") {
		return "CI/CD"
	}
	return "Update"
}

// estimateReadingTime calculates a quick reading time estimate based on words.
func estimateReadingTime(changelog, explainer string) int {
	words := len(strings.Fields(changelog + " " + explainer))
	minutes := words / 200
	if minutes < 1 {
		return 1
	}
	return minutes
}

// renderHeroSVG builds a beautiful line-art branch merge visualization as an inline SVG.
func renderHeroSVG(prNumber int, prTitle string) string {
	return fmt.Sprintf(`<div class="hero-image">
  <svg viewBox="0 0 760 200" width="100%%" height="100%%" xmlns="http://www.w3.org/2000/svg">
    <style>
      .commit-dot {
        cursor: pointer;
        transition: r 0.12s ease-in-out, fill 0.12s ease, opacity 0.12s ease;
      }
      .commit-dot:hover {
        r: 7px;
        fill: #ffffff;
      }
      .merge-dot:hover {
        r: 9px;
        fill: #ffffff;
      }
    </style>

    <!-- Grid Pattern -->
    <defs>
      <pattern id="grid" width="20" height="20" patternUnits="userSpaceOnUse">
        <path d="M 20 0 L 0 0 0 20" fill="none" stroke="#222" stroke-width="1"/>
      </pattern>
    </defs>
    <rect width="100%%" height="100%%" fill="url(#grid)" />
    
    <!-- Branch Lines -->
    <path d="M 50 100 L 710 100" fill="none" stroke="#333" stroke-width="2" stroke-dasharray="4 4" />
    <path d="M 50 100 L 400 100 M 550 100 L 710 100" fill="none" stroke="#888" stroke-width="2" />
    <path d="M 200 100 C 250 100, 270 150, 320 150 L 480 150 C 530 150, 540 100, 550 100" fill="none" stroke="#ffffff" stroke-width="2" />
    
    <!-- Commit Nodes -->
    <!-- On Main line -->
    <a href="https://github.com/ModernRelay/omnigraph/commits/main" target="_blank" rel="noopener">
      <title>View Main Branch Commits</title>
      <circle cx="100" cy="100" r="4" fill="#666" class="commit-dot" />
    </a>
    <a href="https://github.com/ModernRelay/omnigraph/commits/main" target="_blank" rel="noopener">
      <title>View Main Branch Commits</title>
      <circle cx="180" cy="100" r="4" fill="#666" class="commit-dot" />
    </a>
    
    <!-- Merge Node -->
    <a href="https://github.com/ModernRelay/omnigraph/pull/%[1]d" target="_blank" rel="noopener">
      <title>View Pull Request #%[1]d</title>
      <circle cx="550" cy="100" r="6" fill="#ffffff" stroke="#111111" stroke-width="2" class="commit-dot merge-dot" />
    </a>
    
    <!-- On Feature branch -->
    <a href="https://github.com/ModernRelay/omnigraph/pull/%[1]d/commits" target="_blank" rel="noopener">
      <title>View PR #%[1]d Commits</title>
      <circle cx="320" cy="150" r="4" fill="#ffffff" class="commit-dot" />
    </a>
    <a href="https://github.com/ModernRelay/omnigraph/pull/%[1]d/commits" target="_blank" rel="noopener">
      <title>View PR #%[1]d Commits</title>
      <circle cx="400" cy="150" r="4" fill="#ffffff" class="commit-dot" />
    </a>
    <a href="https://github.com/ModernRelay/omnigraph/pull/%[1]d/commits" target="_blank" rel="noopener">
      <title>View PR #%[1]d Commits</title>
      <circle cx="480" cy="150" r="4" fill="#ffffff" class="commit-dot" />
    </a>
    
    <!-- Branch Labels -->
    <text x="60" y="85" fill="#666" font-family="monospace" font-size="10" font-weight="700" letter-spacing="1">MAIN</text>
    <text x="320" y="135" fill="#ffffff" font-family="monospace" font-size="10" font-weight="700" letter-spacing="1">PR #%[1]d</text>
  </svg>
</div>`, prNumber)
}

var (
	codeRegex   = regexp.MustCompile("\x60([^\x60]+)\x60")
	boldRegex   = regexp.MustCompile(`\*\*(.*?)\*\*`)
	italicRegex = regexp.MustCompile(`\*(.*?)\*`)
)

func parseMarkdown(escaped string) string {
	escaped = codeRegex.ReplaceAllString(escaped, "<code>$1</code>")
	escaped = boldRegex.ReplaceAllString(escaped, "<strong>$1</strong>")
	escaped = italicRegex.ReplaceAllString(escaped, "<em>$1</em>")
	return escaped
}
