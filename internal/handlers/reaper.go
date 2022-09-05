package handlers

import (
	"context"
	"github.com/jsiebens/ionscale/internal/broker"
	"github.com/jsiebens/ionscale/internal/domain"
	"time"
)

const (
	ticker            = 10 * time.Minute
	inactivityTimeout = 30 * time.Minute
)

func NewReaper(brokers broker.Pubsub, repository domain.Repository) *Reaper {
	return &Reaper{
		pubsub:     brokers,
		repository: repository,
	}
}

type Reaper struct {
	pubsub     broker.Pubsub
	repository domain.Repository
}

func (r *Reaper) Start() {
	t := time.NewTicker(ticker)
	for range t.C {
		r.reapInactiveEphemeralNodes()
	}
}

func (r *Reaper) reapInactiveEphemeralNodes() {
	ctx := context.Background()

	now := time.Now().UTC()
	checkpoint := now.Add(-inactivityTimeout)
	machines, err := r.repository.ListInactiveEphemeralMachines(ctx, checkpoint)
	if err != nil {
		return
	}
	var removedNodes = make(map[uint64][]uint64)
	for _, m := range machines {
		if now.After(m.LastSeen.Add(inactivityTimeout)) {
			ok, err := r.repository.DeleteMachine(ctx, m.ID)
			if err != nil {
				continue
			}
			if ok {
				removedNodes[m.TailnetID] = append(removedNodes[m.TailnetID], m.ID)
			}
		}
	}

	if len(removedNodes) != 0 {
		for i, p := range removedNodes {
			r.pubsub.Publish(i, &broker.Signal{PeersRemoved: p})
		}
	}
}
