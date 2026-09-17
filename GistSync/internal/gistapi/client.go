package gistapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

const (
	defaultBaseURL  = "https://api.github.com"
	manifestName    = "sync_manifest.json"
	defaultManifest = "{}"
)

var (
	ErrMissingToken     = errors.New("github token is required")
	ErrMissingGistID    = errors.New("gist id is required")
	ErrMissingFileName  = errors.New("file name is required")
	ErrFileNotFound     = errors.New("file not found in gist")
	ErrManifestNotFound = errors.New("manifest gist not found")
)

type ClientOptions struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

type UpsertFileRequest struct {
	GistID   string
	FileName string
	Content  string
}

type FileRequest struct {
	GistID   string
	FileName string
}

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
	cacheMu    sync.RWMutex
	manifestID string
}

type gistFile struct {
	Content *string `json:"content,omitempty"`
}

type gistEntity struct {
	ID    string               `json:"id"`
	Files map[string]*gistFile `json:"files"`
}

func NewClient(options ClientOptions) (*Client, error) {
	if strings.TrimSpace(options.Token) == "" {
		return nil, ErrMissingToken
	}

	baseURL := strings.TrimSpace(options.BaseURL)
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	httpClient := options.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		token:      options.Token,
		httpClient: httpClient,
	}, nil
}

func (c *Client) EnsureManifestGist(ctx context.Context) (string, error) {
	if id, ok := c.cachedManifestID(); ok {
		return id, nil
	}
	id, err := c.FindManifestGist(ctx)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, ErrManifestNotFound) {
		return "", err
	}
	id, err = c.createManifestGist(ctx)
	if err != nil {
		return "", err
	}
	c.setManifestID(id)
	return id, nil
}

// FindManifestGist locates the existing sync manifest without creating cloud data.
func (c *Client) FindManifestGist(ctx context.Context) (string, error) {
	if id, ok := c.cachedManifestID(); ok {
		return id, nil
	}
	gists, err := c.listGists(ctx)
	if err != nil {
		return "", err
	}
	for _, gist := range gists {
		if _, exists := gist.Files[manifestName]; exists {
			c.setManifestID(gist.ID)
			return gist.ID, nil
		}
	}
	return "", ErrManifestNotFound
}

func (c *Client) cachedManifestID() (string, bool) {
	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()
	if c.manifestID == "" {
		return "", false
	}
	return c.manifestID, true
}

func (c *Client) setManifestID(id string) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	c.manifestID = id
}

func (c *Client) UpsertFile(ctx context.Context, req UpsertFileRequest) error {
	if err := validateFileRequest(req.GistID, req.FileName); err != nil {
		return err
	}
	payload := map[string]map[string]*gistFile{
		"files": {
			req.FileName: {Content: &req.Content},
		},
	}
	return c.patchGist(ctx, req.GistID, payload)
}

func (c *Client) GetFileContent(ctx context.Context, req FileRequest) (string, error) {
	if err := validateFileRequest(req.GistID, req.FileName); err != nil {
		return "", err
	}

	gist, err := c.getGist(ctx, req.GistID)
	if err != nil {
		return "", err
	}
	file, exists := gist.Files[req.FileName]
	if !exists || file == nil || file.Content == nil {
		return "", ErrFileNotFound
	}
	return *file.Content, nil
}

func (c *Client) DeleteFile(ctx context.Context, req FileRequest) error {
	if err := validateFileRequest(req.GistID, req.FileName); err != nil {
		return err
	}
	payload := map[string]map[string]*gistFile{
		"files": {
			req.FileName: nil,
		},
	}
	return c.patchGist(ctx, req.GistID, payload)
}

func validateFileRequest(gistID string, fileName string) error {
	if strings.TrimSpace(gistID) == "" {
		return ErrMissingGistID
	}
	if strings.TrimSpace(fileName) == "" {
		return ErrMissingFileName
	}
	return nil
}

func (c *Client) createManifestGist(ctx context.Context) (string, error) {
	content := defaultManifest
	payload := map[string]any{
		"description": "GistSync manifest",
		"public":      false,
		"files": map[string]*gistFile{
			manifestName: {Content: &content},
		},
	}

	req, err := c.newRequest(ctx, http.MethodPost, "/gists", payload)
	if err != nil {
		return "", err
	}
	var gist gistEntity
	if err = c.doJSON(req, &gist); err != nil {
		return "", err
	}
	if gist.ID == "" {
		return "", errors.New("manifest gist creation returned empty id")
	}
	return gist.ID, nil
}

func (c *Client) listGists(ctx context.Context) ([]gistEntity, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/gists", nil)
	if err != nil {
		return nil, err
	}
	var gists []gistEntity
	if err = c.doJSON(req, &gists); err != nil {
		return nil, err
	}
	return gists, nil
}

func (c *Client) getGist(ctx context.Context, gistID string) (gistEntity, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/gists/"+gistID, nil)
	if err != nil {
		return gistEntity{}, err
	}
	var gist gistEntity
	if err = c.doJSON(req, &gist); err != nil {
		return gistEntity{}, err
	}
	return gist, nil
}

func (c *Client) patchGist(ctx context.Context, gistID string, payload any) error {
	req, err := c.newRequest(ctx, http.MethodPatch, "/gists/"+gistID, payload)
	if err != nil {
		return err
	}
	return c.doJSON(req, nil)
}

func (c *Client) newRequest(ctx context.Context, method string, path string, payload any) (*http.Request, error) {
	body, err := marshalBody(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func marshalBody(payload any) (io.Reader, error) {
	if payload == nil {
		return nil, nil
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	return bytes.NewBuffer(raw), nil
}

func (c *Client) doJSON(req *http.Request, target any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("github api %d: %s", resp.StatusCode, string(respBody))
	}
	if target == nil || len(respBody) == 0 {
		return nil
	}
	if err = json.Unmarshal(respBody, target); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
