package broker

import (
	"github.com/jsiebens/ionscale/internal/bind"
	"tailscale.com/tailcfg"
)

func NewClient(id uint64, channel chan *Signal) Client {
	return Client{
		id:      id,
		channel: channel,
	}
}

type Client struct {
	id     uint64
	binder bind.Binder
	node   *tailcfg.Node

	compress string
	channel  chan *Signal
}

func (c *Client) SignalUpdate(s *Signal) {
	c.channel <- s
}
