package mockstore

import (
	"github.com/stretchr/testify/mock"

	"github.com/matterpoll/matterpoll/server/store"
	"github.com/matterpoll/matterpoll/server/store/mockstore/mocks"
)

// Store is a mock store
type Store struct {
	PollStore   mocks.PollStore
	SystemStore mocks.SystemStore
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
