package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeSecretStore struct {
	byRef map[string]string
}

func newFakeSecretStore() *fakeSecretStore {
	return &fakeSecretStore{byRef: map[string]string{}}
}

func (f *fakeSecretStore) Put(ref CredentialRef, value string) error {
	f.byRef[credentialRefKey(ref)] = value
	return nil
}

func (f *fakeSecretStore) Get(ref CredentialRef) (string, error) {
	value, ok := f.byRef[credentialRefKey(ref)]
	if !ok {
		return "", ErrCredentialNotFound
	}
	return value, nil
}

func credentialRefKey(ref CredentialRef) string {
	return ref.Service + "|" + ref.Account
}

func TestStore_SaveLoadRoundTripV2(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "settings.json")
	store, err := NewStoreWithSecrets(filePath, newFakeSecretStore())
	if err != nil {
		t.Fatalf("NewStore returned error: %v", err)
	}

	input := Data{
		Token:           "ABC",
		MasterPassword:  "123",
		ActiveProfileID: "profile-1",
		Profiles: []Profile{
			{
				ID:          "profile-1",
				Name:        "Work",
				RestoreMode: restoreOriginal,
				Enabled:     true,
				Items: []ProfileItem{
					{
						ID:                 "item-1",
						SourcePathTemplate: "{{HOME}}/.gitconfig",
						RelativePath:       ".gitconfig",
						Enabled:            true,
					},
				},
			},
		},
	}
	if err = store.Save(input); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	assertNoPlaintextCredentials(t, filePath)

	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if got.Token != input.Token || got.MasterPassword != input.MasterPassword {
		t.Fatalf("credentials mismatch: got %#v want %#v", got, input)
	}
	if len(got.Profiles) != 1 || len(got.Profiles[0].Items) != 1 {
		t.Fatalf("profile data mismatch: got %#v", got)
	}
}

func TestStore_LoadMissingFile(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "missing.json")
	store, err := NewStoreWithSecrets(filePath, newFakeSecretStore())
	if err != nil {
		t.Fatalf("NewStore returned error: %v", err)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if got.Token != "" || got.MasterPassword != "" || got.ActiveProfileID != "" || len(got.Profiles) != 0 {
		t.Fatalf("expected zero value data for missing file, got %#v", got)
	}
}

func TestStore_MigrateLegacySyncPath(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "settings.json")
	legacyJSON := `{"token":"ABC","masterPassword":"123","syncPath":"{{HOME}}/.ssh/config"}`
	if err := os.WriteFile(filePath, []byte(legacyJSON), 0o600); err != nil {
		t.Fatalf("write legacy file: %v", err)
	}

	store, err := NewStoreWithSecrets(filePath, newFakeSecretStore())
	if err != nil {
		t.Fatalf("NewStore returned error: %v", err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(got.Profiles) != 1 || len(got.Profiles[0].Items) != 1 {
		t.Fatalf("legacy migration failed, got %#v", got)
	}
	if strings.TrimSpace(got.ActiveProfileID) == "" {
		t.Fatalf("expected active profile after migration")
	}
	if !got.CloudBootstrapDone {
		t.Fatalf("expected cloud bootstrap done after migration")
	}
}

func TestStore_LoadMissingFileBootstrapFlag(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "missing.json")
	store, err := NewStoreWithSecrets(filePath, newFakeSecretStore())
	if err != nil {
		t.Fatalf("NewStore returned error: %v", err)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if got.CloudBootstrapDone {
		t.Fatalf("expected cloud bootstrap flag to be false for missing file")
	}
}

func TestStore_Save_WritesCredentialRefsOnly(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "settings.json")
	store, err := NewStoreWithSecrets(filePath, newFakeSecretStore())
	if err != nil {
		t.Fatalf("NewStoreWithSecrets returned error: %v", err)
	}
	input := Data{Token: "ABC", MasterPassword: "123"}

	if err = store.Save(input); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	assertNoPlaintextCredentials(t, filePath)
}

func TestStore_Load_MigratesLegacyCredentialsToSecretStore(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "settings.json")
	legacyJSON := `{"token":"ABC","masterPassword":"123","profiles":[]}`
	if err := os.WriteFile(filePath, []byte(legacyJSON), 0o600); err != nil {
		t.Fatalf("write legacy file: %v", err)
	}
	store, err := NewStoreWithSecrets(filePath, newFakeSecretStore())
	if err != nil {
		t.Fatalf("NewStoreWithSecrets returned error: %v", err)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if got.Token != "ABC" || got.MasterPassword != "123" {
		t.Fatalf("credentials mismatch after migration, got %#v", got)
	}
	assertNoPlaintextCredentials(t, filePath)
}

func assertNoPlaintextCredentials(t *testing.T, filePath string) {
	t.Helper()
	raw, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("read settings file failed: %v", err)
	}
	var payload map[string]any
	if err = json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode settings json failed: %v", err)
	}
	if _, exists := payload["token"]; exists {
		t.Fatalf("token should not exist in settings.json")
	}
	if _, exists := payload["masterPassword"]; exists {
		t.Fatalf("masterPassword should not exist in settings.json")
	}
}
