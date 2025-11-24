package main

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/go-github/v60/github"
	"golang.org/x/oauth2"
)

// fetchGoFilesList lädt das gesamte Repository als ZIP herunter,
// entpackt alle .go-Dateien und gibt deren Inhalte als []string zurück.
func fetchGoFilesList(owner, repo, token string) ([]string, error) {
	ctx := context.Background()
	var client *github.Client
	if token != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		tc := oauth2.NewClient(ctx, ts)
		client = github.NewClient(tc)
	} else {
		client = github.NewClient(nil)
	}

	// Standardbranch abfragen (kostet 1 API-Call)
	r, _, err := client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("konnte Repo nicht abrufen: %w", err)
	}
	defaultBranch := r.GetDefaultBranch()

	// ZIP-URL zusammensetzen
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/zipball/%s", owner, repo, defaultBranch)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	if token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	// ZIP herunterladen
	resp, err := client.Client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("konnte ZIP nicht laden: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Fehler beim Laden des ZIPs: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("konnte ZIP nicht lesen: %w", err)
	}

	// ZIP entpacken
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("konnte ZIP nicht entpacken: %w", err)
	}

	var files []string
	for _, f := range zr.File {
		if !f.FileInfo().IsDir() && strings.HasSuffix(f.Name, ".go") {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			content, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				continue
			}
			files = append(files, string(content))
		}
	}

	return files, nil
}
