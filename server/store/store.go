package store

import (
	"errors"
	"fmt"

	"github.com/blang/semver"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/matterpoll/matterpoll/server/poll"
)

const (
	oldestVersion = "0.1.0"
)

type Store struct {
	api         plugin.API
	pollStore   PollStore
	systemStore SystemStore
}

func NewStore(api plugin.API) (*Store, error) {
	store := Store{
		api: api,
		pollStore: PollStore{
			api: api,
		},
		systemStore: SystemStore{
			api: api,
		},
	}
	err := store.UpdateDatabase()
	if err != nil {
		return nil, err
	}
	return &store, nil
}

func (s *Store) Poll() *PollStore     { return &s.pollStore }
func (s *Store) System() *SystemStore { return &s.systemStore }

func (s *Store) UpdateDatabase() error {
	v, err := s.System().GetVersion()
	if err != nil {
		return err
	}
	// If no version is set, set to to the newest version
	if v == "" {
		v = PluginVersion
	}

	// TODO: Uncomment following condition when version 1.0.0 is released
	/*
		currentSchemaVersion := semver.MustParse(v)
		if err := s.UpgradeDatabaseToVersion10(currentSchemaVersion); err != nil {
			return err
		}
	*/

	return nil
}

func (s *Store) shouldPerformUpgrade(currentSchemaVersion semver.Version, expectedSchemaVersion semver.Version) bool {
	if currentSchemaVersion.LT(expectedSchemaVersion) {
		s.api.LogWarn(fmt.Sprintf("The database schema version of %v appears to be out of date", currentSchemaVersion.String()))
		s.api.LogWarn(fmt.Sprintf("Attempting to upgrade the database schema version to %v", expectedSchemaVersion.String()))
		return true
	}
	return false
}

func (s *Store) UpgradeDatabaseToVersion10(currentSchemaVersion semver.Version) error {
	if s.shouldPerformUpgrade(currentSchemaVersion, semver.MustParse("1.0.0")) {
		s.api.LogWarn("Update complete")
		// Do migration
		if err := s.System().SaveVersion("1.0.0"); err != nil {
			return err
		}
	}
	return nil
}

type PollStore struct {
	api plugin.API
}

const pollPrefix = "poll_"

func (s *PollStore) Get(id string) (*poll.Poll, error) {
	// b, err := s.api.KVGet(pollPrefix + id)
	b, err := s.api.KVGet(id)
	if err != nil {
		return nil, err
	}
	poll := poll.DecodePollFromByte(b)
	if poll == nil {
		return nil, errors.New("failed to decode poll")
	}
	return poll, nil
}

func (s *PollStore) Save(poll *poll.Poll) error {
	// err := s.api.KVSet(pollPrefix+poll.ID, poll.Encode())
	if err := s.api.KVSet(poll.ID, poll.EncodeToByte()); err != nil {
		return err
	}
	return nil
}

func (s *PollStore) Delete(poll *poll.Poll) error {
	// err := s.api.KVDelete(pollPrefix + poll.ID)
	if err := s.api.KVDelete(poll.ID); err != nil {
		return err
	}
	return nil
}

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
