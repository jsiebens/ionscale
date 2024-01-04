package sc

import (
	"slices"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tailcfg"
)

func PeerCount(expected int) func(*ipnstate.Status) bool {
	return func(status *ipnstate.Status) bool {
		return len(status.Peers()) == expected
	}
}

func HasCapability(capability tailcfg.NodeCapability) func(*ipnstate.Status) bool {
	return func(status *ipnstate.Status) bool {
		self := status.Self

		if self == nil {
			return false
		}

		if slices.Contains(self.Capabilities, capability) {
			return true
		}

		if _, ok := self.CapMap[capability]; ok {
			return true
		}

		return false
	}
}

func IsMissingCapability(capability tailcfg.NodeCapability) func(*ipnstate.Status) bool {
	return func(status *ipnstate.Status) bool {
		self := status.Self

		if slices.Contains(self.Capabilities, capability) {
			return false
		}

		if _, ok := self.CapMap[capability]; ok {
			return false
		}

		return true
	}
}
