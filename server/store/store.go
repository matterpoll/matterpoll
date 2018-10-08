package store

import (
	"errors"

	"github.com/mattermost/mattermost-server/plugin"
	"github.com/matterpoll/matterpoll/server/poll"
)

type Store struct {
	pollStore   PollStore
	systemStore SystemStore
}

func NewStore(api plugin.API) Store {
	store := Store{
		pollStore: PollStore{
			api: api,
		},
		systemStore: SystemStore{
			api: api,
		},
	}
	return store
}

func (s *Store) Poll() *PollStore     { return &s.pollStore }
func (s *Store) System() *SystemStore { return &s.systemStore }

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
	// err := s.api.KVSet(pollPrefix + poll.ID, poll.Encode())
	err := s.api.KVSet(poll.ID, poll.EncodeToByte())
	if err != nil {
		return err
	}
	return nil
}

func (s *PollStore) Delete(poll *poll.Poll) error {
	// err := s.api.KVDelete(pollPrefix + poll.ID)
	err := s.api.KVDelete(poll.ID)
	if err != nil {
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
