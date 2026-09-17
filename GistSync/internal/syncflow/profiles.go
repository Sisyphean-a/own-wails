package syncflow

import (
	"context"
	"sort"

	"GistSync/internal/pathmap"
	"GistSync/internal/settings"
)

func (s *Service) ListProfilesFromCloud(ctx context.Context) ([]settings.Profile, error) {
	_, data, err := s.loadManifest(ctx)
	if err != nil {
		return nil, err
	}
	profiles := make([]settings.Profile, 0, len(data.Profiles))
	for _, profile := range data.Profiles {
		profiles = append(profiles, toSettingsProfile(profile))
	}
	return profiles, nil
}

func toSettingsProfile(profile manifestProfile) settings.Profile {
	items := make([]settings.ProfileItem, 0, len(profile.Items))
	for _, item := range profile.Items {
		templatePath := pathmap.CompactHomePath(item.SourcePathTemplate)
		items = append(items, settings.ProfileItem{
			ID:                 item.ID,
			SourcePathTemplate: templatePath,
			RelativePath:       resolveRelativePath(templatePath, item.RelativePath),
			Enabled:            item.Enabled,
		})
	}
	return settings.Profile{
		ID:          profile.ID,
		Name:        profile.Name,
		RestoreMode: profile.RestoreMode,
		RestoreRoot: profile.RestoreRoot,
		Enabled:     true,
		Items:       items,
	}
}

func (s *Service) ListSnapshots(ctx context.Context, profileID string) ([]SnapshotMeta, error) {
	_, data, err := s.loadManifest(ctx)
	if err != nil {
		return nil, err
	}
	var out []SnapshotMeta
	for _, snap := range data.Snapshots {
		if snap.ProfileID != profileID {
			continue
		}
		out = append(out, SnapshotMeta{ID: snap.ID, CreatedAt: snap.CreatedAt})
	}
	sort.Slice(out, func(i int, j int) bool { return out[i].CreatedAt > out[j].CreatedAt })
	return out, nil
}
