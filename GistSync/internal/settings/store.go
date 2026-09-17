package settings

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"GistSync/internal/pathmap"
	"GistSync/internal/profileutil"
)

const (
	configDirName   = "GistSync"
	configFileName  = "settings.json"
	restoreOriginal = "original"
	restoreRooted   = "rooted"
)

var ErrEmptySettingsPath = errors.New("settings file path cannot be empty")
var ErrCredentialNotFound = errors.New("credential not found")

const credentialServiceName = "GistSync"

type CredentialRef struct {
	Service string `json:"service"`
	Account string `json:"account"`
}

type CredentialRefs struct {
	Token          CredentialRef `json:"token"`
	MasterPassword CredentialRef `json:"masterPassword"`
}

type CredentialStore interface {
	Put(ref CredentialRef, value string) error
	Get(ref CredentialRef) (string, error)
}

type ProfileItem struct {
	ID                 string `json:"id"`
	SourcePathTemplate string `json:"sourcePathTemplate"`
	RelativePath       string `json:"relativePath"`
	Enabled            bool   `json:"enabled"`
}

type Profile struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	RestoreMode string        `json:"restoreMode"`
	RestoreRoot string        `json:"restoreRoot"`
	Enabled     bool          `json:"enabled"`
	Items       []ProfileItem `json:"items"`
}

type Data struct {
	Token              string    `json:"token"`
	MasterPassword     string    `json:"masterPassword"`
	ActiveProfileID    string    `json:"activeProfileId"`
	Profiles           []Profile `json:"profiles"`
	CloudBootstrapDone bool      `json:"cloudBootstrapDone,omitempty"`
	SyncPath           string    `json:"syncPath,omitempty"`
}

type Store struct {
	filePath string
	secrets  CredentialStore
}

type persistedData struct {
	Token              string         `json:"token,omitempty"`
	MasterPassword     string         `json:"masterPassword,omitempty"`
	Credentials        CredentialRefs `json:"credentials,omitempty"`
	ActiveProfileID    string         `json:"activeProfileId"`
	Profiles           []Profile      `json:"profiles"`
	CloudBootstrapDone bool           `json:"cloudBootstrapDone,omitempty"`
	SyncPath           string         `json:"syncPath,omitempty"`
}

func NewDefaultStore() (*Store, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("resolve user config dir: %w", err)
	}
	filePath := filepath.Join(configDir, configDirName, configFileName)
	return NewStore(filePath)
}

func NewStore(filePath string) (*Store, error) {
	return NewStoreWithSecrets(filePath, newSystemCredentialStore())
}

func NewStoreWithSecrets(filePath string, secrets CredentialStore) (*Store, error) {
	if strings.TrimSpace(filePath) == "" {
		return nil, ErrEmptySettingsPath
	}
	if secrets == nil {
		return nil, errors.New("credential store cannot be nil")
	}
	return &Store{filePath: filePath, secrets: secrets}, nil
}

func (s *Store) Save(data Data) error {
	data = applyDefaults(data)
	if err := os.MkdirAll(filepath.Dir(s.filePath), 0o755); err != nil {
		return fmt.Errorf("create settings directory: %w", err)
	}
	refs := defaultCredentialRefs(s.filePath)
	if err := s.saveCredentials(refs, data); err != nil {
		return err
	}
	persisted := toPersistedData(data, refs)
	raw, err := json.MarshalIndent(persisted, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}
	if err = os.WriteFile(s.filePath, raw, 0o600); err != nil {
		return fmt.Errorf("write settings: %w", err)
	}
	return nil
}

func (s *Store) Load() (Data, error) {
	raw, err := os.ReadFile(s.filePath)
	if errors.Is(err, os.ErrNotExist) {
		return Data{}, nil
	}
	if err != nil {
		return Data{}, fmt.Errorf("read settings: %w", err)
	}

	var persisted persistedData
	if err = json.Unmarshal(raw, &persisted); err != nil {
		return Data{}, fmt.Errorf("decode settings: %w", err)
	}
	data, refs, changed, err := s.resolvePersistedData(persisted)
	if err != nil {
		return Data{}, err
	}
	if changed {
		if saveErr := s.writePersisted(data, refs); saveErr != nil {
			return Data{}, saveErr
		}
	}
	data = migrateLegacy(data)
	return applyDefaults(data), nil
}

func (s *Store) resolvePersistedData(p persistedData) (Data, CredentialRefs, bool, error) {
	data := Data{
		Token:              p.Token,
		MasterPassword:     p.MasterPassword,
		ActiveProfileID:    p.ActiveProfileID,
		Profiles:           p.Profiles,
		CloudBootstrapDone: p.CloudBootstrapDone,
		SyncPath:           p.SyncPath,
	}
	refs := p.Credentials
	if refs.Token.Account == "" || refs.MasterPassword.Account == "" {
		refs = defaultCredentialRefs(s.filePath)
	}
	changed := false
	if data.Token != "" {
		if err := s.secrets.Put(refs.Token, data.Token); err != nil {
			return Data{}, CredentialRefs{}, false, fmt.Errorf("save token to credential store: %w", err)
		}
		changed = true
	}
	if data.MasterPassword != "" {
		if err := s.secrets.Put(refs.MasterPassword, data.MasterPassword); err != nil {
			return Data{}, CredentialRefs{}, false, fmt.Errorf("save master password to credential store: %w", err)
		}
		changed = true
	}
	token, tokenErr := s.readCredential(refs.Token)
	if tokenErr != nil {
		return Data{}, CredentialRefs{}, false, tokenErr
	}
	masterPassword, masterErr := s.readCredential(refs.MasterPassword)
	if masterErr != nil {
		return Data{}, CredentialRefs{}, false, masterErr
	}
	data.Token = token
	data.MasterPassword = masterPassword
	if p.Token != "" || p.MasterPassword != "" || p.Credentials.Token.Account == "" || p.Credentials.MasterPassword.Account == "" {
		changed = true
	}
	return data, refs, changed, nil
}

func (s *Store) readCredential(ref CredentialRef) (string, error) {
	value, err := s.secrets.Get(ref)
	if errors.Is(err, ErrCredentialNotFound) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read credential from store: %w", err)
	}
	return value, nil
}

func (s *Store) saveCredentials(refs CredentialRefs, data Data) error {
	if data.Token != "" {
		if err := s.secrets.Put(refs.Token, data.Token); err != nil {
			return fmt.Errorf("save token to credential store: %w", err)
		}
	}
	if data.MasterPassword != "" {
		if err := s.secrets.Put(refs.MasterPassword, data.MasterPassword); err != nil {
			return fmt.Errorf("save master password to credential store: %w", err)
		}
	}
	return nil
}

func (s *Store) writePersisted(data Data, refs CredentialRefs) error {
	persisted := toPersistedData(data, refs)
	raw, err := json.MarshalIndent(persisted, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}
	if err = os.WriteFile(s.filePath, raw, 0o600); err != nil {
		return fmt.Errorf("write settings: %w", err)
	}
	return nil
}

func toPersistedData(data Data, refs CredentialRefs) persistedData {
	return persistedData{
		Credentials:        refs,
		ActiveProfileID:    data.ActiveProfileID,
		Profiles:           data.Profiles,
		CloudBootstrapDone: data.CloudBootstrapDone,
		SyncPath:           data.SyncPath,
	}
}

func defaultCredentialRefs(filePath string) CredentialRefs {
	sum := sha256.Sum256([]byte(filePath))
	base := fmt.Sprintf("%x", sum[:])
	return CredentialRefs{
		Token: CredentialRef{
			Service: credentialServiceName,
			Account: base + "|token",
		},
		MasterPassword: CredentialRef{
			Service: credentialServiceName,
			Account: base + "|master_password",
		},
	}
}

func applyDefaults(data Data) Data {
	if len(data.Profiles) == 0 {
		data.ActiveProfileID = ""
		return data
	}
	data.CloudBootstrapDone = true
	for i := range data.Profiles {
		profile := &data.Profiles[i]
		if profile.ID == "" {
			profile.ID = generateID("profile")
		}
		if strings.TrimSpace(profile.Name) == "" {
			profile.Name = autoProfileName()
		}
		if profile.RestoreMode == "" {
			profile.RestoreMode = restoreOriginal
		}
		for j := range profile.Items {
			item := &profile.Items[j]
			item.SourcePathTemplate = pathmap.CompactHomePath(item.SourcePathTemplate)
			item.RelativePath = buildRelativePath(item.SourcePathTemplate)
			if item.ID == "" {
				item.ID = generateID("item")
			}
		}
	}

	activeExists := false
	for _, profile := range data.Profiles {
		if profile.ID == data.ActiveProfileID {
			activeExists = true
			break
		}
	}
	if data.ActiveProfileID == "" || !activeExists {
		data.ActiveProfileID = data.Profiles[0].ID
	}
	return data
}

func migrateLegacy(data Data) Data {
	if strings.TrimSpace(data.SyncPath) == "" || len(data.Profiles) > 0 {
		return data
	}
	profileID := generateID("profile")
	itemID := generateID("item")
	data.Profiles = []Profile{
		{
			ID:          profileID,
			Name:        "迁移配置",
			RestoreMode: restoreOriginal,
			Enabled:     true,
			Items: []ProfileItem{
				{
					ID:                 itemID,
					SourcePathTemplate: data.SyncPath,
					RelativePath:       buildRelativePath(data.SyncPath),
					Enabled:            true,
				},
			},
		},
	}
	data.ActiveProfileID = profileID
	data.SyncPath = ""
	return data
}

func buildRelativePath(sourcePath string) string {
	return profileutil.NormalizeRelativePath(sourcePath)
}

func autoProfileName() string {
	return "配置-" + time.Now().Format("20060102-150405")
}

func generateID(prefix string) string {
	return profileutil.GenerateID(prefix)
}

func IsValidRestoreMode(mode string) bool {
	return mode == restoreOriginal || mode == restoreRooted
}
