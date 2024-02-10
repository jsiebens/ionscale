package tsn

import (
	"slices"
	"strings"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tailcfg"
	"tailscale.com/types/views"
)

type Condition = func(*ipnstate.Status) bool

func Connected() Condition {
	return func(status *ipnstate.Status) bool {
		return status.CurrentTailnet != nil
	}
}

func HasTailnet(tailnet string) Condition {
	return func(status *ipnstate.Status) bool {
		return status.CurrentTailnet != nil && status.CurrentTailnet.Name == tailnet
	}
}

func HasTag(tag string) Condition {
	return func(status *ipnstate.Status) bool {
		return status.Self != nil && status.Self.Tags != nil && views.SliceContains[string](*status.Self.Tags, tag)
	}
}

func HasName(name string) Condition {
	return func(status *ipnstate.Status) bool {
		return status.Self != nil && strings.HasPrefix(status.Self.DNSName, name)
	}
}

func NeedsMachineAuth() Condition {
	return func(status *ipnstate.Status) bool {
		return status.BackendState == "NeedsMachineAuth"
	}
}

func IsRunning() Condition {
	return func(status *ipnstate.Status) bool {
		return status.BackendState == "Running"
	}
}

func HasUser(email string) Condition {
	return func(status *ipnstate.Status) bool {
		if status.Self == nil {
			return false
		}
		userID := status.Self.UserID
		if u, ok := status.User[userID]; ok {
			return u.LoginName == email
		}

		return false
	}
}

func PeerCount(expected int) Condition {
	return func(status *ipnstate.Status) bool {
		return len(status.Peers()) == expected
	}
}

func HasExpiredPeer(name string) Condition {
	return func(status *ipnstate.Status) bool {
		for _, peer := range status.Peer {
			if strings.HasPrefix(peer.DNSName, name) {
				return peer.Expired
			}
		}
		return false
	}
}

func HasCapability(capability tailcfg.NodeCapability) Condition {
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

func IsMissingCapability(capability tailcfg.NodeCapability) Condition {
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
