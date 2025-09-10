package kvstore

import (
	"github.com/mattermost/mattermost/server/public/plugin"
)

// SystemStore allows to access system informations in the KV Store.
type SystemStore struct {
	api plugin.API
}

const versionKey = "version"

// GetVersion returns the db schema version.
func (s *SystemStore) GetVersion() (string, error) {
	b, err := s.api.KVGet(versionKey)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// SaveVersion sets the db schema version.
func (s *SystemStore) SaveVersion(version string) error {
	err := s.api.KVSet(versionKey, []byte(version))
	if err != nil {
		return err
	}
	return nil
}
