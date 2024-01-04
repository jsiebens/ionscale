package core

import (
	"slices"
	"sync"
	"time"
)

type Ping struct{}

type PollMapSessionManager interface {
	Register(tailnetID uint64, machineID uint64, ch chan *Ping)
	Deregister(tailnetID uint64, machineID uint64)
	HasSession(tailnetID uint64, machineID uint64) bool
	NotifyAll(tailnetID uint64, ignoreMachineIDs ...uint64)
}

func NewPollMapSessionManager() PollMapSessionManager {
	return &pollMapSessionManager{
		data:   map[uint64]map[uint64]chan *Ping{},
		timers: map[uint64]*time.Timer{},
	}
}

type pollMapSessionManager struct {
	sync.RWMutex
	data   map[uint64]map[uint64]chan *Ping
	timers map[uint64]*time.Timer
}

func (n *pollMapSessionManager) Register(tailnetID uint64, machineID uint64, ch chan *Ping) {
	n.Lock()
	defer n.Unlock()

	if ss := n.data[tailnetID]; ss == nil {
		n.data[tailnetID] = map[uint64]chan *Ping{machineID: ch}
	} else {
		ss[machineID] = ch
	}

	t, ok := n.timers[machineID]
	if ok {
		t.Stop()
		delete(n.timers, machineID)
	}
}

func (n *pollMapSessionManager) Deregister(tailnetID uint64, machineID uint64) {
	n.Lock()
	defer n.Unlock()

	if ss := n.data[tailnetID]; ss != nil {
		delete(ss, machineID)
	}

	t, ok := n.timers[machineID]
	if ok {
		t.Stop()
		delete(n.timers, machineID)
	}

	timer := time.NewTimer(10 * time.Second)
	go func() {
		<-timer.C
		if !n.HasSession(tailnetID, machineID) {
			n.NotifyAll(tailnetID)
		}
	}()

	n.timers[machineID] = timer
}

func (n *pollMapSessionManager) HasSession(tailnetID uint64, machineID uint64) bool {
	n.RLock()
	defer n.RUnlock()

	if ss := n.data[tailnetID]; ss != nil {
		if _, ok := ss[machineID]; ok {
			return true
		}
	}

	return false
}

func (n *pollMapSessionManager) NotifyAll(tailnetID uint64, ignoreMachineIDs ...uint64) {
	n.RLock()
	defer n.RUnlock()

	if ss := n.data[tailnetID]; ss != nil {
		for i, p := range ss {
			if !slices.Contains(ignoreMachineIDs, i) {
				p <- &Ping{}
			}
		}
	}
}
