package remote

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	DefaultRepo = "Abraxas-365/manifesto"
	GitHubAPI   = "https://api.github.com"
	RawGitHub   = "https://raw.githubusercontent.com"
	DefaultRef  = "main"
)

type Release struct {
	TagName string `json:"tag_name"`
}

type Client struct {
	repo       string
	httpClient *http.Client
}

func NewClient(repo string) *Client {
	if repo == "" {
		repo = DefaultRepo
	}
	return &Client{
		repo:       repo,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *Client) GetLatestVersion() (string, error) {
	url := fmt.Sprintf("%s/repos/%s/releases/latest", GitHubAPI, c.repo)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return DefaultRef, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return DefaultRef, nil
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil || release.TagName == "" {
		return DefaultRef, nil
	}
	return release.TagName, nil
}

// FetchModulePaths downloads the repo at ref and extracts only the given paths.
// It rewrites Go imports from goModuleOld to goModuleNew.
func (c *Client) FetchModulePaths(ref string, paths []string, destRoot, goModuleOld, goModuleNew string) error {
	archiveData, err := c.downloadArchive(ref)
	if err != nil {
		return err
	}

	gz, err := gzip.NewReader(bytes.NewReader(archiveData))
	if err != nil {
		return fmt.Errorf("decompress: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar read: %w", err)
		}

		// Strip top-level GitHub dir (e.g. "manifesto-main/").
		parts := strings.SplitN(header.Name, "/", 2)
		if len(parts) < 2 || parts[1] == "" {
			continue
		}
		relPath := parts[1]

		if !matchesAnyPrefix(relPath, paths) {
			continue
		}

		destPath := filepath.Join(destRoot, relPath)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(destPath, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return err
			}

			content, err := io.ReadAll(tr)
			if err != nil {
				return fmt.Errorf("read %s: %w", relPath, err)
			}

			// Rewrite Go imports.
			if strings.HasSuffix(relPath, ".go") && goModuleOld != "" && goModuleNew != "" {
				content = []byte(strings.ReplaceAll(string(content), goModuleOld, goModuleNew))
			}

			if err := os.WriteFile(destPath, content, os.FileMode(header.Mode)); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Client) FetchGoMod(ref string) (string, error) {
	url := fmt.Sprintf("%s/%s/%s/go.mod", RawGitHub, c.repo, ref)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("go.mod not found: HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	return string(data), err
}

func (c *Client) downloadArchive(ref string) ([]byte, error) {
	urls := []string{
		fmt.Sprintf("https://github.com/%s/archive/refs/tags/%s.tar.gz", c.repo, ref),
		fmt.Sprintf("https://github.com/%s/archive/refs/heads/%s.tar.gz", c.repo, ref),
	}
	if ref == DefaultRef || ref == "" {
		urls = []string{urls[1]}
	}

	for _, u := range urls {
		resp, err := c.httpClient.Get(u)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			return io.ReadAll(resp.Body)
		}
	}

	return nil, fmt.Errorf("failed to download archive for ref '%s'", ref)
}

func matchesAnyPrefix(path string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}
