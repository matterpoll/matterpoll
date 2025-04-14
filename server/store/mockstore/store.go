package mockstore

import (
	"github.com/stretchr/testify/mock"

	"github.com/matterpoll/matterpoll/server/store"
)

// Store is a mock store
type Store struct {
	PollStore   PollStore
	SystemStore SystemStore
}

// Poll returns the Poll Store
func (s *Store) Poll() store.PollStore { return &s.PollStore }

// System returns the System Store
func (s *Store) System() store.SystemStore { return &s.SystemStore }

// AssertExpectations makes sure the expectations of all stores are meet
func (s *Store) AssertExpectations(t mock.TestingT) {
	s.PollStore.AssertExpectations(t)
	s.SystemStore.AssertExpectations(t)
}
