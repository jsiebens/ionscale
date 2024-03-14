package defaults

import (
	"github.com/jsiebens/ionscale/pkg/client/ionscale"
	ionscalev1 "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
)

func DefaultIAMPolicy() *ionscale.IAMPolicy {
	return &ionscale.IAMPolicy{}
}

func DefaultACLPolicy() *ionscale.ACLPolicy {
	return &ionscale.ACLPolicy{
		ACLs: []ionscale.ACLEntry{
			{
				Action:      "accept",
				Source:      []string{"*"},
				Destination: []string{"*:*"},
			},
		},
		SSH: []ionscale.ACLSSH{
			{
				Action:      "check",
				Source:      []string{"autogroup:member"},
				Destination: []string{"autogroup:self"},
				Users:       []string{"autogroup:nonroot", "root"},
			},
		},
	}
}

func DefaultDNSConfig() *ionscalev1.DNSConfig {
	return &ionscalev1.DNSConfig{
		MagicDns: true,
	}
}
