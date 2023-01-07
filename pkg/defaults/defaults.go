package defaults

import ionscalev1 "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"

func DefaultACLPolicy() *ionscalev1.ACLPolicy {
	return &ionscalev1.ACLPolicy{
		Acls: []*ionscalev1.ACL{
			{
				Action: "accept",
				Src:    []string{"*"},
				Dst:    []string{"*:*"},
			},
		},
	}
}

func DefaultDNSConfig() *ionscalev1.DNSConfig {
	return &ionscalev1.DNSConfig{
		MagicDns: true,
	}
}
