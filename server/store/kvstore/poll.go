package kvstore

import (
	"errors"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"

	"github.com/matterpoll/matterpoll/server/poll"
)

// PollStore allows to access polls in the KV Store.
type PollStore struct {
	api plugin.API
}

const pollPrefix = "poll_"

// Get returns the poll for a given id. Returns an error if the poll doesn't exist or a KV Store error occurred.
func (s *PollStore) Get(id string) (*poll.Poll, error) {
	b, err := s.api.KVGet(pollPrefix + id)
	if err != nil {
		return nil, err
	}

	poll := poll.DecodePollFromByte(b)
	if poll == nil {
		return nil, errors.New("failed to decode poll")
	}

	return poll, nil
}

// Insert stores new a poll in the KV Store.
func (s *PollStore) Insert(poll *poll.Poll) error {
	opt := model.PluginKVSetOptions{
		Atomic:   true,
		OldValue: nil,
	}
	ok, err := s.api.KVSetWithOptions(pollPrefix+poll.ID, poll.EncodeToByte(), opt)
	if err != nil {
		return err
	}

	if !ok {
		return errors.New("poll already exists in database")
	}

	return nil
}

// Save stores a poll in the KV Store. Overwrittes any existing poll with the same id.
func (s *PollStore) Save(poll *poll.Poll) error {
	if err := s.api.KVSet(pollPrefix+poll.ID, poll.EncodeToByte()); err != nil {
		return err
	}

	return nil
}

// Update updates an existing a poll in the KV Store.
func (s *PollStore) Update(prev *poll.Poll, new *poll.Poll) error {
	opt := model.PluginKVSetOptions{
		Atomic:   true,
		OldValue: prev.EncodeToByte(),
	}
	ok, err := s.api.KVSetWithOptions(pollPrefix+prev.ID, new.EncodeToByte(), opt)
	if err != nil {
		return err
	}

	if !ok {
		return errors.New("poll already exists in database")
	}

	return nil
}

// Delete deletes a poll from the KV Store.
func (s *PollStore) Delete(poll *poll.Poll) error {
	if err := s.api.KVDelete(pollPrefix + poll.ID); err != nil {
		return err
	}

	return nil
}
