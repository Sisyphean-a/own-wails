package settings

import (
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"
)

type systemCredentialStore struct{}

func newSystemCredentialStore() CredentialStore {
	return &systemCredentialStore{}
}

func (s *systemCredentialStore) Put(ref CredentialRef, value string) error {
	if err := keyring.Set(ref.Service, ref.Account, value); err != nil {
		return fmt.Errorf("keyring set failed: %w", err)
	}
	return nil
}

func (s *systemCredentialStore) Get(ref CredentialRef) (string, error) {
	value, err := keyring.Get(ref.Service, ref.Account)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", ErrCredentialNotFound
	}
	if err != nil {
		return "", fmt.Errorf("keyring get failed: %w", err)
	}
	return value, nil
}
