package broker

type Signal struct {
	PeerUpdated  *uint64
	PeersRemoved []uint64
	ACLUpdated   bool
	DNSUpdated   bool
}

type Listener chan *Signal

type Pubsub interface {
	Subscribe(tailnet uint64, listener Listener) (cancel func(), err error)
	Publish(tailnet uint64, message *Signal) error
	Close() error
}
