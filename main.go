package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "generate":
		runGenerate(os.Args[2:])
	case "serve":
		runServe(os.Args[2:])
	case "version":
		fmt.Printf("omni-scribe %s\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `omni-scribe — AI-generated documentation for Omnigraph PRs

Usage:
  omni-scribe generate --pr <number>   Fetch a PR, generate docs via Gemini, save to disk
  omni-scribe serve [--port 8080]      Start the HTTP server to view generated docs
  omni-scribe version                  Print version
  omni-scribe help                     Show this help

Environment variables:
  GEMINI_API_KEY   Required for 'generate'. Your Gemini API key.
  DATA_DIR         Directory for stored docs (default: ./data)
  PORT             Server port, overridden by --port flag (default: 8080)
`)
}

func runGenerate(args []string) {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	prNumber := fs.Int("pr", 0, "PR number to generate docs for")
	fs.Parse(args)

	if *prNumber == 0 {
		fmt.Fprintln(os.Stderr, "Error: --pr flag is required")
		fmt.Fprintln(os.Stderr, "Usage: omni-scribe generate --pr <number>")
		os.Exit(1)
	}

	fmt.Printf("→ Fetching PR #%d from GitHub...\n", *prNumber)
	pr, err := fetchPR(*prNumber)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching PR: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  ✓ PR #%d: %s (by %s)\n", pr.Meta.Number, pr.Meta.Title, pr.Meta.User.Login)
	fmt.Printf("  ✓ %d files changed, diff size: %d bytes\n", len(pr.Files), len(pr.Diff))

	fmt.Println("→ Generating documentation via Gemini...")
	docs, err := generateDocs(pr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating docs: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("  ✓ Changelog and explainer generated")

	doc := &StoredDoc{
		PRNumber:    pr.Meta.Number,
		PRTitle:     pr.Meta.Title,
		PRAuthor:    pr.Meta.User.Login,
		PRURL:       pr.Meta.HTMLURL,
		GeneratedAt: time.Now().UTC(),
		Docs:        *docs,
	}

	fmt.Println("→ Saving to local store...")
	if err := saveDoc(doc); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving doc: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  ✓ Saved to %s\n", docPath(pr.Meta.Number))
	fmt.Printf("\n✅ Done! View at /pr/%d when the server is running.\n", pr.Meta.Number)
}

func runServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	port := fs.Int("port", 0, "Port to listen on")
	fs.Parse(args)

	p := *port
	if p == 0 {
		// Check PORT env var (Cloud Run convention)
		if envPort := os.Getenv("PORT"); envPort != "" {
			fmt.Sscanf(envPort, "%d", &p)
		}
	}
	if p == 0 {
		p = 8080
	}

	startServer(p)
}
