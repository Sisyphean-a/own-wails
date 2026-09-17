package gistapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

const testToken = "token-123"

type gistRecord struct {
	ID    string             `json:"id"`
	Files map[string]*string `json:"files"`
}

type mockGitHubAPI struct {
	gists      map[string]gistRecord
	nextGistID int
}

func TestEnsureManifestGist_UseExisting(t *testing.T) {
	server, state := newMockGitHubServer()
	defer server.Close()

	manifestContent := "{}"
	state.gists["gist-existing"] = gistRecord{
		ID: "gist-existing",
		Files: map[string]*string{
			manifestName: &manifestContent,
		},
	}

	client := mustNewClient(t, server.URL)
	gistID, err := client.EnsureManifestGist(context.Background())
	if err != nil {
		t.Fatalf("EnsureManifestGist returned error: %v", err)
	}
	if gistID != "gist-existing" {
		t.Fatalf("expected existing gist id, got %q", gistID)
	}
}

func TestFindManifestGist_DoesNotCreateWhenMissing(t *testing.T) {
	server, state := newMockGitHubServer()
	defer server.Close()

	client := mustNewClient(t, server.URL)
	_, err := client.FindManifestGist(context.Background())
	if !errors.Is(err, ErrManifestNotFound) {
		t.Fatalf("expected ErrManifestNotFound, got %v", err)
	}
	if len(state.gists) != 0 {
		t.Fatalf("FindManifestGist must not create a gist, got %#v", state.gists)
	}
}

func TestEnsureManifestGist_CreateWhenMissing(t *testing.T) {
	server, _ := newMockGitHubServer()
	defer server.Close()

	client := mustNewClient(t, server.URL)
	gistID, err := client.EnsureManifestGist(context.Background())
	if err != nil {
		t.Fatalf("EnsureManifestGist returned error: %v", err)
	}
	if gistID == "" {
		t.Fatalf("expected new gist id")
	}

	content, err := client.GetFileContent(context.Background(), FileRequest{
		GistID:   gistID,
		FileName: manifestName,
	})
	if err != nil {
		t.Fatalf("GetFileContent returned error: %v", err)
	}
	if content != defaultManifest {
		t.Fatalf("manifest content mismatch: got %q want %q", content, defaultManifest)
	}
}

func TestUpsertGetDeleteFile(t *testing.T) {
	server, _ := newMockGitHubServer()
	defer server.Close()

	client := mustNewClient(t, server.URL)
	gistID, err := client.EnsureManifestGist(context.Background())
	if err != nil {
		t.Fatalf("EnsureManifestGist returned error: %v", err)
	}

	writeReq := UpsertFileRequest{
		GistID:   gistID,
		FileName: "test.txt",
		Content:  "Hello Cloud",
	}
	if err = client.UpsertFile(context.Background(), writeReq); err != nil {
		t.Fatalf("UpsertFile returned error: %v", err)
	}

	got, err := client.GetFileContent(context.Background(), FileRequest{
		GistID:   gistID,
		FileName: "test.txt",
	})
	if err != nil {
		t.Fatalf("GetFileContent returned error: %v", err)
	}
	if got != "Hello Cloud" {
		t.Fatalf("content mismatch: got %q want %q", got, "Hello Cloud")
	}

	if err = client.DeleteFile(context.Background(), FileRequest{
		GistID:   gistID,
		FileName: "test.txt",
	}); err != nil {
		t.Fatalf("DeleteFile returned error: %v", err)
	}

	_, err = client.GetFileContent(context.Background(), FileRequest{
		GistID:   gistID,
		FileName: "test.txt",
	})
	if err == nil || !strings.Contains(err.Error(), ErrFileNotFound.Error()) {
		t.Fatalf("expected file-not-found error, got: %v", err)
	}
}

func mustNewClient(t *testing.T, baseURL string) *Client {
	t.Helper()
	client, err := NewClient(ClientOptions{
		BaseURL: baseURL,
		Token:   testToken,
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	return client
}

func newMockGitHubServer() (*httptest.Server, *mockGitHubAPI) {
	state := &mockGitHubAPI{
		gists:      map[string]gistRecord{},
		nextGistID: 1,
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+testToken {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/gists":
			state.handleListGists(w)
		case r.Method == http.MethodPost && r.URL.Path == "/gists":
			state.handleCreateGist(w, r)
		case strings.HasPrefix(r.URL.Path, "/gists/"):
			state.handleGistByID(w, r)
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	return server, state
}

func (s *mockGitHubAPI) handleListGists(w http.ResponseWriter) {
	result := make([]map[string]any, 0, len(s.gists))
	for _, gist := range s.gists {
		result = append(result, map[string]any{
			"id":    gist.ID,
			"files": encodeFiles(gist.Files),
		})
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *mockGitHubAPI) handleCreateGist(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Files map[string]struct {
			Content string `json:"content"`
		} `json:"files"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	id := s.allocateGistID()
	s.gists[id] = gistRecord{
		ID:    id,
		Files: decodePayloadFiles(payload.Files),
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":    id,
		"files": encodeFiles(s.gists[id].Files),
	})
}

func (s *mockGitHubAPI) handleGistByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/gists/")
	gist, exists := s.gists[id]
	if !exists {
		http.Error(w, "gist not found", http.StatusNotFound)
		return
	}
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, map[string]any{"id": id, "files": encodeFiles(gist.Files)})
		return
	}
	if r.Method == http.MethodPatch {
		s.handlePatchGist(w, r, gist)
		return
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func (s *mockGitHubAPI) handlePatchGist(w http.ResponseWriter, r *http.Request, gist gistRecord) {
	var payload struct {
		Files map[string]*struct {
			Content string `json:"content"`
		} `json:"files"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	for name, file := range payload.Files {
		if file == nil {
			delete(gist.Files, name)
			continue
		}
		content := file.Content
		gist.Files[name] = &content
	}
	s.gists[gist.ID] = gist
	writeJSON(w, http.StatusOK, map[string]any{"id": gist.ID, "files": encodeFiles(gist.Files)})
}

func (s *mockGitHubAPI) allocateGistID() string {
	id := "gist-" + strconv.Itoa(s.nextGistID)
	s.nextGistID++
	return id
}

func decodePayloadFiles(files map[string]struct {
	Content string `json:"content"`
}) map[string]*string {
	result := map[string]*string{}
	for name, file := range files {
		content := file.Content
		result[name] = &content
	}
	return result
}

func encodeFiles(files map[string]*string) map[string]map[string]string {
	result := map[string]map[string]string{}
	for name, content := range files {
		if content == nil {
			continue
		}
		result[name] = map[string]string{"content": *content}
	}
	return result
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}
