package syncflow

import (
	"context"
	"encoding/json"
	"fmt"

	"GistSync/internal/security"
)

// Flow: read one blob from the newest snapshot without creating or modifying cloud data.
func (s *Service) ValidateMasterPassword(ctx context.Context, password string) (MasterPasswordValidation, error) {
	if empty(password) {
		return MasterPasswordValidation{}, ErrEmptyPassword
	}
	gistID, err := s.cloud.FindManifestGist(ctx)
	if err != nil {
		return MasterPasswordValidation{}, err
	}
	if empty(gistID) {
		return MasterPasswordValidation{}, nil
	}
	content, err := s.cloud.GetFileContent(ctx, FileRequest{GistID: gistID, FileName: manifestFileName})
	if err != nil {
		return MasterPasswordValidation{}, err
	}
	if empty(content) {
		return MasterPasswordValidation{}, nil
	}
	var data manifest
	if err = json.Unmarshal([]byte(content), &data); err != nil {
		return MasterPasswordValidation{}, fmt.Errorf("decode manifest: %w", err)
	}
	snapshot, ok := data.latestSnapshotWithItems()
	if !ok {
		return MasterPasswordValidation{}, nil
	}
	ciphertext, err := s.cloud.GetFileContent(ctx, FileRequest{GistID: gistID, FileName: snapshot.Items[0].BlobFile})
	if err != nil {
		return MasterPasswordValidation{}, err
	}
	_, err = security.DecryptString(ciphertext, password)
	return MasterPasswordValidation{Valid: err == nil, HasSnapshot: true}, nil
}

func (m manifest) latestSnapshotWithItems() (manifestSnapshot, bool) {
	var latest manifestSnapshot
	found := false
	for _, snapshot := range m.Snapshots {
		if len(snapshot.Items) == 0 || (found && snapshot.CreatedAt <= latest.CreatedAt) {
			continue
		}
		latest = snapshot
		found = true
	}
	return latest, found
}
