package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/storage"
)

// StoredDoc is what gets persisted per PR.
type StoredDoc struct {
	PRNumber    int           `json:"pr_number"`
	PRTitle     string        `json:"pr_title"`
	PRAuthor    string        `json:"pr_author"`
	PRURL       string        `json:"pr_url"`
	GeneratedAt time.Time     `json:"generated_at"`
	Docs        GeneratedDocs `json:"docs"`
}

// useGCS returns true if the GCS_BUCKET environment variable is set.
func useGCS() bool {
	return os.Getenv("GCS_BUCKET") != ""
}

// bucketName returns the configured GCS bucket name.
func bucketName() string {
	return os.Getenv("GCS_BUCKET")
}

// objectKey returns the GCS object key or file suffix for a PR.
func objectKey(prNumber int) string {
	return fmt.Sprintf("prs/%d.json", prNumber)
}

// dataDir returns the root directory for local stored docs.
func dataDir() string {
	d := os.Getenv("DATA_DIR")
	if d == "" {
		return "./data"
	}
	return d
}

// prsDir returns the directory where individual PR JSON files are stored locally.
func prsDir() string {
	return filepath.Join(dataDir(), "prs")
}

// docPath returns the local file path for a specific PR number.
func docPath(prNumber int) string {
	return filepath.Join(prsDir(), fmt.Sprintf("%d.json", prNumber))
}

// saveDoc writes a StoredDoc to either GCS (if GCS_BUCKET is set) or the local filesystem.
func saveDoc(doc *StoredDoc) error {
	if useGCS() {
		ctx := context.Background()
		client, err := storage.NewClient(ctx)
		if err != nil {
			return fmt.Errorf("creating GCS client: %w", err)
		}
		defer client.Close()

		data, err := json.Marshal(doc)
		if err != nil {
			return fmt.Errorf("marshalling doc: %w", err)
		}

		wc := client.Bucket(bucketName()).Object(objectKey(doc.PRNumber)).NewWriter(ctx)
		wc.ContentType = "application/json"

		if _, err := wc.Write(data); err != nil {
			return fmt.Errorf("writing to GCS: %w", err)
		}
		if err := wc.Close(); err != nil {
			return fmt.Errorf("closing GCS writer: %w", err)
		}
		return nil
	}

	// Local storage fallback
	dir := prsDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating data directory %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling doc: %w", err)
	}

	path := docPath(doc.PRNumber)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

// loadDoc reads a StoredDoc from either GCS or the local filesystem.
func loadDoc(prNumber int) (*StoredDoc, error) {
	if useGCS() {
		ctx := context.Background()
		client, err := storage.NewClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("creating GCS client: %w", err)
		}
		defer client.Close()

		rc, err := client.Bucket(bucketName()).Object(objectKey(prNumber)).NewReader(ctx)
		if err != nil {
			if err == storage.ErrObjectNotExist {
				return nil, nil // not found
			}
			return nil, fmt.Errorf("reading from GCS: %w", err)
		}
		defer rc.Close()

		data, err := io.ReadAll(rc)
		if err != nil {
			return nil, fmt.Errorf("reading GCS object: %w", err)
		}

		var doc StoredDoc
		if err := json.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("unmarshalling doc: %w", err)
		}
		return &doc, nil
	}

	// Local storage fallback
	path := docPath(prNumber)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // not found
		}
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var doc StoredDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("unmarshalling %s: %w", path, err)
	}
	return &doc, nil
}

// listDocs returns all stored docs, sorted by PR number descending (newest first).
func listDocs() ([]*StoredDoc, error) {
	if useGCS() {
		ctx := context.Background()
		client, err := storage.NewClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("creating GCS client: %w", err)
		}
		defer client.Close()

		it := client.Bucket(bucketName()).Objects(ctx, &storage.Query{Prefix: "prs/"})
		var docs []*StoredDoc
		for {
			attrs, err := it.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("listing GCS objects: %w", err)
			}
			if !strings.HasSuffix(attrs.Name, ".json") {
				continue
			}
			parts := strings.Split(attrs.Name, "/")
			if len(parts) != 2 {
				continue
			}
			name := strings.TrimSuffix(parts[1], ".json")
			prNum, err := strconv.Atoi(name)
			if err != nil {
				continue
			}
			doc, err := loadDoc(prNum)
			if err != nil {
				continue
			}
			if doc != nil {
				docs = append(docs, doc)
			}
		}

		sort.Slice(docs, func(i, j int) bool {
			return docs[i].PRNumber > docs[j].PRNumber
		})
		return docs, nil
	}

	// Local storage fallback
	dir := prsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // no docs yet
		}
		return nil, fmt.Errorf("reading directory %s: %w", dir, err)
	}

	var docs []*StoredDoc
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".json")
		prNum, err := strconv.Atoi(name)
		if err != nil {
			continue
		}
		doc, err := loadDoc(prNum)
		if err != nil {
			continue
		}
		if doc != nil {
			docs = append(docs, doc)
		}
	}

	sort.Slice(docs, func(i, j int) bool {
		return docs[i].PRNumber > docs[j].PRNumber
	})
	return docs, nil
}
