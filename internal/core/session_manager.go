package core

import (
	"github.com/puzpuzpuz/xsync/v3"
	"slices"
	"sync"
	"time"
)

type Ping struct{}

type PollMapSessionManager interface {
	Register(tailnetID uint64, machineID uint64, ch chan<- *Ping)
	Deregister(tailnetID uint64, machineID uint64, ch chan<- *Ping)
	HasSession(tailnetID uint64, machineID uint64) bool
	NotifyAll(tailnetID uint64, ignoreMachineIDs ...uint64)
}

func NewPollMapSessionManager() PollMapSessionManager {
	return &pollMapSessionManager{
		tailnets: xsync.NewMapOf[uint64, *tailnetSessionManager](),
	}
}

type pollMapSessionManager struct {
	tailnets *xsync.MapOf[uint64, *tailnetSessionManager]
}

func (n *pollMapSessionManager) load(tailnetID uint64) *tailnetSessionManager {
	m, _ := n.tailnets.LoadOrCompute(tailnetID, func() *tailnetSessionManager {
		return &tailnetSessionManager{
			targets:  make(map[uint64]chan<- *Ping),
			timers:   make(map[uint64]*time.Timer),
			sessions: xsync.NewMapOf[uint64, bool](),
		}
	})
	return m
}

func (n *pollMapSessionManager) Register(tailnetID uint64, machineID uint64, ch chan<- *Ping) {
	n.load(tailnetID).Register(machineID, ch)
}

func (n *pollMapSessionManager) Deregister(tailnetID uint64, machineID uint64, ch chan<- *Ping) {
	n.load(tailnetID).Deregister(machineID, ch)
}

func (n *pollMapSessionManager) HasSession(tailnetID uint64, machineID uint64) bool {
	return n.load(tailnetID).HasSession(machineID)
}

func (n *pollMapSessionManager) NotifyAll(tailnetID uint64, ignoreMachineIDs ...uint64) {
	n.load(tailnetID).NotifyAll(ignoreMachineIDs...)
}

type tailnetSessionManager struct {
	sync.RWMutex
	targets  map[uint64]chan<- *Ping
	timers   map[uint64]*time.Timer
	sessions *xsync.MapOf[uint64, bool]
}

func (n *tailnetSessionManager) NotifyAll(ignoreMachineIDs ...uint64) {
	n.RLock()
	defer n.RUnlock()

	for i, p := range n.targets {
		if !slices.Contains(ignoreMachineIDs, i) {
			select {
			case p <- &Ping{}:
			default: // ignore, channel has a small buffer, failing to insert means there is already a ping pending
			}
		}
	}
}

func (n *tailnetSessionManager) Register(machineID uint64, ch chan<- *Ping) {
	n.Lock()
	defer n.Unlock()

	if curr, ok := n.targets[machineID]; ok {
		close(curr)
	}

	n.targets[machineID] = ch
	n.sessions.Store(machineID, true)

	t, ok := n.timers[machineID]
	if ok {
		t.Stop()
		delete(n.timers, machineID)
	}

	timer := time.NewTimer(5 * time.Second)
	go func() {
		<-timer.C
		if n.HasSession(machineID) {
			n.NotifyAll(machineID)
		}
	}()

	n.timers[machineID] = timer
}

func (n *tailnetSessionManager) Deregister(machineID uint64, ch chan<- *Ping) {
	n.Lock()
	defer n.Unlock()

	if curr, ok := n.targets[machineID]; ok && curr != ch {
		return
	}

	delete(n.targets, machineID)
	n.sessions.Store(machineID, false)

	t, ok := n.timers[machineID]
	if ok {
		t.Stop()
		delete(n.timers, machineID)
	}

	timer := time.NewTimer(10 * time.Second)
	go func() {
		<-timer.C
		if !n.HasSession(machineID) {
			n.NotifyAll()
		}
	}()

	n.timers[machineID] = timer
}

func (n *tailnetSessionManager) HasSession(machineID uint64) bool {
	v, ok := n.sessions.Load(machineID)
	return ok && v
}
