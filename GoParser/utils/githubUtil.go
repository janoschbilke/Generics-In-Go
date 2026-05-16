package utils

import (
	"GoParser/model"
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

func FetchGoFilesList(owner, repo, token string) ([]model.FileInfo, error) {
	ctx := context.Background()
	var client *github.Client
	if token != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		tc := oauth2.NewClient(ctx, ts)
		client = github.NewClient(tc)
	} else {
		client = github.NewClient(nil)
	}

	// Fetch default branch (1 API call)
	r, _, err := client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("konnte Repo nicht abrufen: %w", err)
	}
	defaultBranch := r.GetDefaultBranch()

	// Build ZIP URL
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/zipball/%s", owner, repo, defaultBranch)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	if token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	// Download ZIP
	resp, err := client.Client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("konnte ZIP nicht laden: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("konnte ZIP nicht laden: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("konnte ZIP nicht lesen: %w", err)
	}

	// Unzip
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("konnte ZIP nicht entpacken: %w", err)
	}
	
	prefix := zipTopLevelPrefix(zr)

	var files []model.FileInfo
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}

		name := f.Name
		if prefix != "" && strings.HasPrefix(name, prefix) {
			name = name[len(prefix):]
		}

		if !strings.HasSuffix(name, ".go") && name != "go.mod" {
			continue
		}

		if isSkippedPath(name) {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			continue
		}
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}

		files = append(files, model.FileInfo{
			Path:    name,
			Content: string(content),
		})
	}

	return files, nil
}

func zipTopLevelPrefix(zr *zip.Reader) string {
	for _, f := range zr.File {
		idx := strings.Index(f.Name, "/")
		if idx >= 0 {
			return f.Name[:idx+1]
		}
	}
	return ""
}

func isSkippedPath(path string) bool {
	parts := strings.Split(path, "/")
	for _, p := range parts {
		if p == "vendor" || p == "node_modules" || p == ".git" || strings.HasPrefix(p, ".") {
			return true
		}
	}
	return false
}
