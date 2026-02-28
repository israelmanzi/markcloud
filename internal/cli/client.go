package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/israelmanzi/markcloud/internal/sync"
)

type ServerClient struct {
	baseURL string
	secret  string
	http    *http.Client
}

func NewServerClient(serverURL, deploySecret string) *ServerClient {
	return &ServerClient{
		baseURL: serverURL,
		secret:  deploySecret,
		http:    &http.Client{},
	}
}

func (c *ServerClient) Manifest(entries []sync.ManifestEntry) ([]string, error) {
	body, err := json.Marshal(map[string]any{"files": entries})
	if err != nil {
		return nil, err
	}

	resp, err := c.post("/api/sync/manifest", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("manifest: %s — %s", resp.Status, string(msg))
	}

	var result sync.DiffResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.NeedContent, nil
}

func (c *ServerClient) Upload(files []sync.FileEntry, allPaths []string) error {
	body, err := json.Marshal(map[string]any{
		"files":     files,
		"all_paths": allPaths,
	})
	if err != nil {
		return err
	}

	resp, err := c.post("/api/sync/upload", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload: %s — %s", resp.Status, string(msg))
	}
	return nil
}

func (c *ServerClient) post(path string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest("POST", c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.secret)
	return c.http.Do(req)
}
