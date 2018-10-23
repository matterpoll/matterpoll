package store

import "github.com/mattermost/mattermost-server/plugin"

type SystemStore struct {
	api plugin.API
}

const versionKey = "version"

func (s *SystemStore) GetVersion() (string, error) {
	b, err := s.api.KVGet(versionKey)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (s *SystemStore) SaveVersion(version string) error {
	err := s.api.KVSet(versionKey, []byte(version))
	if err != nil {
		return err
	}
	return nil
}
