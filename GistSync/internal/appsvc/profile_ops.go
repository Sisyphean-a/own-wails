package appsvc

import (
	"context"

	"GistSync/internal/settings"
)

func (s *Service) LoadSettings(ctx context.Context) (settings.Data, error) {
	return s.profiles.LoadSettings(ctx)
}

func (s *Service) SaveSettings(ctx context.Context, data settings.Data) error {
	return s.profiles.SaveSettings(ctx, data)
}

func (s *Service) PullProfilesFromCloud(ctx context.Context) (int, error) {
	return s.profiles.PullProfilesFromCloud(ctx)
}

func (s *Service) CreateProfile(ctx context.Context, name string) (settings.Profile, error) {
	return s.profiles.CreateProfile(ctx, name)
}

func (s *Service) DeleteProfile(ctx context.Context, profileID string) error {
	return s.profiles.DeleteProfile(ctx, profileID)
}

func (s *Service) SetActiveProfile(ctx context.Context, profileID string) error {
	return s.profiles.SetActiveProfile(ctx, profileID)
}

func (s *Service) AddFilesToProfile(ctx context.Context, profileID string, paths []string) error {
	return s.profiles.AddFilesToProfile(ctx, profileID, paths)
}

func (s *Service) RemoveProfileItems(ctx context.Context, profileID string, itemIDs []string) error {
	return s.profiles.RemoveProfileItems(ctx, profileID, itemIDs)
}

func (s *Service) SetProfileItemsEnabled(ctx context.Context, profileID string, itemIDs []string, enabled bool) error {
	return s.profiles.SetProfileItemsEnabled(ctx, profileID, itemIDs, enabled)
}
