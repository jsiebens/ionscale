package core

import (
	"context"
	"github.com/jsiebens/ionscale/internal/domain"
	"time"
)

const (
	ticker            = 10 * time.Minute
	inactivityTimeout = 30 * time.Minute
)

func StartWorker(repository domain.Repository, sessionManager PollMapSessionManager) {
	r := &worker{
		sessionManager: sessionManager,
		repository:     repository,
	}

	go r.start()
}

type worker struct {
	sessionManager PollMapSessionManager
	repository     domain.Repository
}

func (r *worker) start() {
	r.deleteInactiveEphemeralNodes()
	t := time.NewTicker(ticker)
	for range t.C {
		r.deleteInactiveEphemeralNodes()
	}
}

func (r *worker) deleteInactiveEphemeralNodes() {
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
		for i, _ := range removedNodes {
			r.sessionManager.NotifyAll(i)
		}
	}
}
