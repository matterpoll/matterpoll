package kvstore

import (
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/matterpoll/matterpoll/server/store"
)

type Store struct {
	api         plugin.API
	pollStore   PollStore
	systemStore SystemStore
}

func NewStore(api plugin.API, pluginVersion string) (store.Store, error) {
	store := Store{
		api: api,
		pollStore: PollStore{
			api: api,
		},
		systemStore: SystemStore{
			api: api,
		},
	}
	err := store.UpdateDatabase(pluginVersion)
	if err != nil {
		return nil, err
	}

	return &store, nil
}

func (s *Store) Poll() store.PollStore     { return &s.pollStore }
func (s *Store) System() store.SystemStore { return &s.systemStore }
