package kvstore

import (
	"errors"

	"github.com/mattermost/mattermost-server/plugin"
	"github.com/matterpoll/matterpoll/server/poll"
)

type PollStore struct {
	api plugin.API
}

const pollPrefix = "poll_"

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

func (s *PollStore) Save(poll *poll.Poll) error {
	if err := s.api.KVSet(pollPrefix+poll.ID, poll.EncodeToByte()); err != nil {
		return err
	}
	return nil
}

func (s *PollStore) Delete(poll *poll.Poll) error {
	if err := s.api.KVDelete(pollPrefix + poll.ID); err != nil {
		return err
	}
	return nil
}
