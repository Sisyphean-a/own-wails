package appsvc

import (
	"context"
	"strings"

	"GistSync/internal/pathmap"
	"GistSync/internal/profileutil"
	"GistSync/internal/settings"
	"GistSync/internal/syncflow"
)

type DefaultProfileManager struct {
	store      SettingsStore
	buildSync  SyncFactory
	generateID func(prefix string) string
}

func NewDefaultProfileManager(store SettingsStore, buildSync SyncFactory, generateID func(prefix string) string) *DefaultProfileManager {
	return &DefaultProfileManager{store: store, buildSync: buildSync, generateID: generateID}
}

func (m *DefaultProfileManager) LoadSettings(ctx context.Context) (settings.Data, error) {
	data, err := m.store.Load()
	if err != nil {
		return settings.Data{}, err
	}
	if !shouldBootstrap(data) {
		return data, nil
	}
	merged, pullErr := m.pullProfilesIntoSettings(ctx, data)
	if pullErr == nil {
		return merged, nil
	}
	return data, nil
}

func (m *DefaultProfileManager) SaveSettings(_ context.Context, data settings.Data) error {
	return m.store.Save(data)
}

func (m *DefaultProfileManager) PullProfilesFromCloud(ctx context.Context) (int, error) {
	data, err := m.store.Load()
	if err != nil {
		return 0, err
	}
	merged, err := m.pullProfilesIntoSettings(ctx, data)
	if err != nil {
		return 0, err
	}
	return len(merged.Profiles), nil
}

func (m *DefaultProfileManager) CreateProfile(_ context.Context, name string) (settings.Profile, error) {
	data, err := m.store.Load()
	if err != nil {
		return settings.Profile{}, err
	}
	profile := settings.Profile{
		ID:          m.generateID("profile"),
		Name:        strings.TrimSpace(name),
		RestoreMode: "original",
		Enabled:     true,
		Items:       []settings.ProfileItem{},
	}
	data.Profiles = append(data.Profiles, profile)
	if data.ActiveProfileID == "" {
		data.ActiveProfileID = profile.ID
	}
	if err = m.store.Save(data); err != nil {
		return settings.Profile{}, err
	}
	return profile, nil
}

func (m *DefaultProfileManager) DeleteProfile(_ context.Context, profileID string) error {
	data, err := m.store.Load()
	if err != nil {
		return err
	}
	filtered := make([]settings.Profile, 0, len(data.Profiles))
	for _, profile := range data.Profiles {
		if profile.ID != profileID {
			filtered = append(filtered, profile)
		}
	}
	data.Profiles = filtered
	if data.ActiveProfileID == profileID {
		data.ActiveProfileID = ""
		if len(data.Profiles) > 0 {
			data.ActiveProfileID = data.Profiles[0].ID
		}
	}
	return m.store.Save(data)
}

func (m *DefaultProfileManager) SetActiveProfile(_ context.Context, profileID string) error {
	data, err := m.store.Load()
	if err != nil {
		return err
	}
	if _, ok := findProfile(data, profileID); !ok {
		return syncflow.ErrProfileNotFound
	}
	data.ActiveProfileID = profileID
	return m.store.Save(data)
}

func (m *DefaultProfileManager) AddFilesToProfile(_ context.Context, profileID string, paths []string) error {
	data, err := m.store.Load()
	if err != nil {
		return err
	}
	profile, ok := findProfile(data, profileID)
	if !ok {
		return syncflow.ErrProfileNotFound
	}
	for _, path := range paths {
		if empty(path) {
			continue
		}
		templatePath := pathmap.CompactHomePath(path)
		profile.Items = append(profile.Items, settings.ProfileItem{
			ID:                 m.generateID("item"),
			SourcePathTemplate: templatePath,
			RelativePath:       profileutil.NormalizeRelativePath(templatePath),
			Enabled:            true,
		})
	}
	return m.store.Save(data)
}

func (m *DefaultProfileManager) SetProfileItemsEnabled(_ context.Context, profileID string, itemIDs []string, enabled bool) error {
	data, err := m.store.Load()
	if err != nil {
		return err
	}
	profile, ok := findProfile(data, profileID)
	if !ok {
		return syncflow.ErrProfileNotFound
	}
	target := make(map[string]bool, len(itemIDs))
	for _, id := range itemIDs {
		target[id] = true
	}
	for i := range profile.Items {
		if target[profile.Items[i].ID] {
			profile.Items[i].Enabled = enabled
		}
	}
	return m.store.Save(data)
}

func (m *DefaultProfileManager) RemoveProfileItems(_ context.Context, profileID string, itemIDs []string) error {
	data, err := m.store.Load()
	if err != nil {
		return err
	}
	profile, ok := findProfile(data, profileID)
	if !ok {
		return syncflow.ErrProfileNotFound
	}
	drop := make(map[string]bool, len(itemIDs))
	for _, id := range itemIDs {
		drop[id] = true
	}
	filtered := make([]settings.ProfileItem, 0, len(profile.Items))
	for _, item := range profile.Items {
		if !drop[item.ID] {
			filtered = append(filtered, item)
		}
	}
	profile.Items = filtered
	return m.store.Save(data)
}

func (m *DefaultProfileManager) pullProfilesIntoSettings(ctx context.Context, data settings.Data) (settings.Data, error) {
	syncService, err := m.buildSync(data.Token)
	if err != nil {
		return settings.Data{}, err
	}
	cloudProfiles, err := syncService.ListProfilesFromCloud(ctx)
	if err != nil {
		return settings.Data{}, err
	}
	data.Profiles = mergeProfiles(data.Profiles, cloudProfiles)
	data.CloudBootstrapDone = true
	if len(data.Profiles) > 0 && empty(data.ActiveProfileID) {
		data.ActiveProfileID = data.Profiles[0].ID
	}
	if saveErr := m.store.Save(data); saveErr != nil {
		return settings.Data{}, saveErr
	}
	return data, nil
}

func mergeProfiles(local []settings.Profile, cloud []settings.Profile) []settings.Profile {
	byID := make(map[string]int)
	merged := make([]settings.Profile, 0, len(local)+len(cloud))
	for _, profile := range local {
		byID[profile.ID] = len(merged)
		merged = append(merged, profile)
	}
	for _, profile := range cloud {
		if idx, exists := byID[profile.ID]; exists {
			merged[idx] = profile
			continue
		}
		byID[profile.ID] = len(merged)
		merged = append(merged, profile)
	}
	return merged
}
