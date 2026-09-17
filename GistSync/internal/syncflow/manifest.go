package syncflow

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"GistSync/internal/pathmap"
	"GistSync/internal/settings"
)

// Guarantee: every cloud-facing operation reads the current manifest so metadata changes made on another device are immediately observable.
func (s *Service) loadManifest(ctx context.Context) (string, manifest, error) {
	startedAt := time.Now()
	action := "load_manifest"
	gistID, err := s.cloud.EnsureManifestGist(ctx)
	if err != nil {
		s.observer.Record(action, time.Since(startedAt), false, 0, 0)
		return "", manifest{}, err
	}
	content, err := s.cloud.GetFileContent(ctx, FileRequest{GistID: gistID, FileName: manifestFileName})
	if err != nil {
		s.observer.Record(action, time.Since(startedAt), false, 0, 0)
		return "", manifest{}, err
	}
	if empty(content) {
		data := manifest{Version: manifestVersion}
		s.observer.Record(action, time.Since(startedAt), true, estimateManifestSize(data), len(data.Snapshots))
		return gistID, data, nil
	}
	var data manifest
	if err = json.Unmarshal([]byte(content), &data); err != nil {
		s.observer.Record(action, time.Since(startedAt), false, len(content), 0)
		return "", manifest{}, fmt.Errorf("decode manifest: %w", err)
	}
	if data.Version == 0 {
		data.Version = manifestVersion
	}
	s.observer.Record(action, time.Since(startedAt), true, len(content), len(data.Snapshots))
	return gistID, data, nil
}

func (s *Service) saveManifest(ctx context.Context, gistID string, data manifest) error {
	startedAt := time.Now()
	action := "save_manifest"
	data.Version = manifestVersion
	raw, err := json.Marshal(data)
	if err != nil {
		s.observer.Record(action, time.Since(startedAt), false, 0, len(data.Snapshots))
		return fmt.Errorf("encode manifest: %w", err)
	}
	if err = s.cloud.UpsertFile(ctx, UpsertFileRequest{GistID: gistID, FileName: manifestFileName, Content: string(raw)}); err != nil {
		s.observer.Record(action, time.Since(startedAt), false, len(raw), len(data.Snapshots))
		return err
	}
	s.observer.Record(action, time.Since(startedAt), true, len(raw), len(data.Snapshots))
	return nil
}

func (m *manifest) upsertProfile(profile settings.Profile) {
	entry := manifestProfile{ID: profile.ID, Name: profile.Name, RestoreMode: profile.RestoreMode, RestoreRoot: profile.RestoreRoot}
	for _, item := range profile.Items {
		templatePath := pathmap.CompactHomePath(item.SourcePathTemplate)
		entry.Items = append(entry.Items, manifestProfileItem{
			ID:                 item.ID,
			SourcePathTemplate: templatePath,
			RelativePath:       resolveRelativePath(templatePath, item.RelativePath),
			Enabled:            item.Enabled,
		})
	}
	for i := range m.Profiles {
		if m.Profiles[i].ID == profile.ID {
			m.Profiles[i] = entry
			return
		}
	}
	m.Profiles = append(m.Profiles, entry)
}

func (m *manifest) findProfileAndSnapshot(profileID string, snapshotID string) (manifestProfile, manifestSnapshot, error) {
	profile, ok := m.findProfile(profileID)
	if !ok {
		return manifestProfile{}, manifestSnapshot{}, ErrProfileNotFound
	}
	snap, ok := m.findSnapshot(profileID, snapshotID)
	if !ok {
		return manifestProfile{}, manifestSnapshot{}, ErrSnapshotNotFound
	}
	return profile, snap, nil
}

func (m *manifest) findProfile(profileID string) (manifestProfile, bool) {
	for _, profile := range m.Profiles {
		if profile.ID == profileID {
			return profile, true
		}
	}
	return manifestProfile{}, false
}

func (m *manifest) findSnapshot(profileID string, snapshotID string) (manifestSnapshot, bool) {
	if !empty(snapshotID) {
		for _, snap := range m.Snapshots {
			if snap.ProfileID == profileID && snap.ID == snapshotID {
				return snap, true
			}
		}
		return manifestSnapshot{}, false
	}
	var latest manifestSnapshot
	found := false
	for _, snap := range m.Snapshots {
		if snap.ProfileID != profileID {
			continue
		}
		if !found || snap.CreatedAt > latest.CreatedAt {
			latest = snap
			found = true
		}
	}
	return latest, found
}

func estimateManifestSize(data manifest) int {
	raw, err := json.Marshal(data)
	if err != nil {
		return 0
	}
	return len(raw)
}
