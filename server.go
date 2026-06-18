package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func startServer(port int) {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", handleHealthz)
	mux.HandleFunc("GET /pr/{number}", handlePR)
	mux.HandleFunc("GET /", handleIndex)

	addr := fmt.Sprintf(":%d", port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		fmt.Printf("omni-scribe server listening on http://localhost%s\n", addr)
		fmt.Printf("  → Index:   http://localhost%s/\n", addr)
		fmt.Printf("  → Health:  http://localhost%s/healthz\n", addr)
		fmt.Printf("  → PR docs: http://localhost%s/pr/<number>\n", addr)
		fmt.Println()
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
	}()

	<-done
	fmt.Println("\nShutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	fmt.Println("Server stopped.")
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "ok")
}

func handlePR(w http.ResponseWriter, r *http.Request) {
	numStr := r.PathValue("number")
	prNumber, err := strconv.Atoi(numStr)
	if err != nil || prNumber <= 0 {
		http.Error(w, "Invalid PR number", http.StatusBadRequest)
		return
	}

	doc, err := loadDoc(prNumber)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading PR #%d: %v\n", prNumber, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if doc == nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, renderNotFound(prNumber))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, renderHTML(doc))
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	// Only serve the index at the exact root path
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	docs, err := listDocs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing docs: %v\n", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, renderIndex(docs))
}

// renderNotFound produces a styled 404 page for missing PR docs.
func renderNotFound(prNumber int) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>PR #%d Not Found — Omni-Scribe</title>
  <style>
    body {
      background: #f5f5f5;
      color: #111111;
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif;
      display: flex;
      align-items: center;
      justify-content: center;
      min-height: 100vh;
      margin: 0;
      padding: 1.5rem;
    }
    .msg {
      max-width: 460px;
      width: 100%%;
      background: #ffffff;
      border: 1px solid #111111;
      padding: 2.5rem 3rem;
      text-align: left;
    }
    h1 {
      font-size: 2.5rem;
      font-weight: 700;
      color: #111111;
      margin-bottom: 1rem;
      letter-spacing: -0.03em;
      border-bottom: 1px solid #111111;
      padding-bottom: 0.5rem;
    }
    p {
      font-size: 0.95rem;
      line-height: 1.6;
      margin-bottom: 2rem;
      color: #666666;
    }
    code {
      font-family: SFMono-Regular, Consolas, "Liberation Mono", Menlo, monospace;
      font-size: 0.85em;
      background: #fafafa;
      border: 1px solid #e0e0e0;
      padding: 0.15em 0.4em;
      color: #111111;
    }
    a {
      display: inline-block;
      color: #ffffff;
      background: #111111;
      text-decoration: none;
      padding: 0.6rem 1.2rem;
      font-size: 0.85rem;
      font-weight: 600;
      letter-spacing: 0.05em;
      text-transform: uppercase;
      border: 1px solid #111111;
    }
    a:hover {
      background: #ffffff;
      color: #111111;
    }
  </style>
</head>
<body>
  <div class="msg">
    <h1>404</h1>
    <p>Documentation for PR #%d hasn't been generated yet.<br><br>
    Run <code>omni-scribe generate --pr %d</code> to create it.</p>
    <a href="/">&larr; Back to index</a>
  </div>
</body>
</html>`, prNumber, prNumber, prNumber)
}


// parsePRNumberFromPath extracts the PR number from a path like /pr/248.
// Returns 0 if the path doesn't match.
func parsePRNumberFromPath(path string) int {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 || parts[0] != "pr" {
		return 0
	}
	n, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0
	}
	return n
}
