package store

import (
	"github.com/mattermost/mattermost-server/plugin"
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
