package store

import "github.com/matterpoll/matterpoll/server/poll"

type Store interface {
	Poll() PollStore
	System() SystemStore
}

type PollStore interface {
	Get(id string) (*poll.Poll, error)
	Save(poll *poll.Poll) error
	Delete(poll *poll.Poll) error
}

type SystemStore interface {
	GetVersion() (string, error)
	SaveVersion(version string) error
}
