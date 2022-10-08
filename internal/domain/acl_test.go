package domain

import (
	"github.com/jsiebens/ionscale/internal/addr"
	"github.com/stretchr/testify/assert"
	"net/netip"
	"sort"
	"tailscale.com/tailcfg"
	"testing"
)

func TestACLPolicy_BuildFilterRulesWildcards(t *testing.T) {
	p1 := createMachine("john@example.com")
	p2 := createMachine("jane@example.com")

	policy := ACLPolicy{
		ACLs: []ACL{
			{
				Action: "accept",
				Src:    []string{"*"},
				Dst:    []string{"*:*"},
			},
		},
	}

	dst := createMachine("john@example.com")

	actualRules := policy.BuildFilterRules([]Machine{*p1, *p2}, dst)
	expectedRules := []tailcfg.FilterRule{
		{
			SrcIPs: []string{"*"},
			DstPorts: []tailcfg.NetPortRange{
				{
					IP: "*",
					Ports: tailcfg.PortRange{
						First: 0,
						Last:  65535,
					},
				},
			},
		},
	}

	assert.Equal(t, expectedRules, actualRules)
}

func TestACLPolicy_BuildFilterRulesWithGroups(t *testing.T) {
	p1 := createMachine("jane@example.com")
	p2 := createMachine("nick@example.com")
	p3 := createMachine("joe@example.com")

	policy := ACLPolicy{
		Groups: map[string][]string{
			"group:admin": []string{"jane@example.com"},
			"group:audit": []string{"nick@example.com"},
		},
		ACLs: []ACL{
			{
				Action: "accept",
				Src:    []string{"group:admin"},
				Dst:    []string{"*:22"},
			},
			{
				Action: "accept",
				Src:    []string{"group:audit"},
				Dst:    []string{"*:8000-8080"},
			},
		},
	}

	dst := createMachine("john@example.com")

	actualRules := policy.BuildFilterRules([]Machine{*p1, *p2, *p3}, dst)
	expectedRules := []tailcfg.FilterRule{
		{
			SrcIPs: []string{
				p1.IPv4.String(),
				p1.IPv6.String(),
			},
			DstPorts: []tailcfg.NetPortRange{
				{
					IP: "*",
					Ports: tailcfg.PortRange{
						First: 22,
						Last:  22,
					},
				},
			},
		},
		{
			SrcIPs: []string{
				p2.IPv4.String(),
				p2.IPv6.String(),
			},
			DstPorts: []tailcfg.NetPortRange{
				{
					IP: "*",
					Ports: tailcfg.PortRange{
						First: 8000,
						Last:  8080,
					},
				},
			},
		},
	}

	assert.Equal(t, expectedRules, actualRules)
}

func TestACLPolicy_BuildFilterRulesWithAutoGroupMembers(t *testing.T) {
	p1 := createMachine("jane@example.com")
	p2 := createMachine("nick@example.com")
	p3 := createMachine("joe@example.com", "tag:web")

	policy := ACLPolicy{
		ACLs: []ACL{
			{
				Action: "accept",
				Src:    []string{"autogroup:members"},
				Dst:    []string{"*:22"},
			},
		},
	}

	dst := createMachine("john@example.com")

	actualRules := policy.BuildFilterRules([]Machine{*p1, *p2, *p3}, dst)

	expectedSrcIPs := []string{
		p1.IPv4.String(), p1.IPv6.String(),
		p2.IPv4.String(), p2.IPv6.String(),
	}
	sort.Strings(expectedSrcIPs)

	expectedRules := []tailcfg.FilterRule{
		{
			SrcIPs: expectedSrcIPs,
			DstPorts: []tailcfg.NetPortRange{
				{
					IP: "*",
					Ports: tailcfg.PortRange{
						First: 22,
						Last:  22,
					},
				},
			},
		},
	}

	assert.Equal(t, expectedRules, actualRules)
}

func TestACLPolicy_BuildFilterRulesAutogroupSelf(t *testing.T) {
	p1 := createMachine("john@example.com")
	p2 := createMachine("jane@example.com")

	policy := ACLPolicy{
		ACLs: []ACL{
			{
				Action: "accept",
				Src:    []string{"*"},
				Dst:    []string{"autogroup:self:*"},
			},
		},
	}

	dst := createMachine("john@example.com")

	actualRules := policy.BuildFilterRules([]Machine{*p1, *p2}, dst)
	expectedRules := []tailcfg.FilterRule{
		{
			SrcIPs: []string{
				p1.IPv4.String(),
				p1.IPv6.String(),
			},
			DstPorts: []tailcfg.NetPortRange{
				{
					IP: dst.IPv4.String(),
					Ports: tailcfg.PortRange{
						First: 0,
						Last:  65535,
					},
				},
				{
					IP: dst.IPv6.String(),
					Ports: tailcfg.PortRange{
						First: 0,
						Last:  65535,
					},
				},
			},
		},
	}

	assert.Equal(t, expectedRules, actualRules)
}

func TestACLPolicy_BuildFilterRulesAutogroupSelfAndTags(t *testing.T) {
	p1 := createMachine("john@example.com")
	p2 := createMachine("john@example.com", "tag:web")

	policy := ACLPolicy{
		ACLs: []ACL{
			{
				Action: "accept",
				Src:    []string{"*"},
				Dst:    []string{"autogroup:self:*"},
			},
		},
	}

	dst := createMachine("john@example.com")

	actualRules := policy.BuildFilterRules([]Machine{*p1, *p2}, dst)
	expectedRules := []tailcfg.FilterRule{
		{
			SrcIPs: []string{
				p1.IPv4.String(),
				p1.IPv6.String(),
			},
			DstPorts: []tailcfg.NetPortRange{
				{
					IP: dst.IPv4.String(),
					Ports: tailcfg.PortRange{
						First: 0,
						Last:  65535,
					},
				},
				{
					IP: dst.IPv6.String(),
					Ports: tailcfg.PortRange{
						First: 0,
						Last:  65535,
					},
				},
			},
		},
	}

	assert.Equal(t, expectedRules, actualRules)
}

func TestACLPolicy_BuildFilterRulesAutogroupSelfAndOtherDestinations(t *testing.T) {
	p1 := createMachine("john@example.com")
	p2 := createMachine("john@example.com", "tag:web")
	p3 := createMachine("jane@example.com")

	policy := ACLPolicy{
		ACLs: []ACL{
			{
				Action: "accept",
				Src:    []string{"*"},
				Dst:    []string{"autogroup:self:22", "john@example.com:80"},
			},
		},
	}

	dst := createMachine("john@example.com")

	actualRules := policy.BuildFilterRules([]Machine{*p1, *p2, *p3}, dst)
	expectedRules := []tailcfg.FilterRule{
		{
			SrcIPs: p1.IPs(),
			DstPorts: []tailcfg.NetPortRange{
				{
					IP: dst.IPv4.String(),
					Ports: tailcfg.PortRange{
						First: 22,
						Last:  22,
					},
				},
				{
					IP: dst.IPv6.String(),
					Ports: tailcfg.PortRange{
						First: 22,
						Last:  22,
					},
				},
			},
		},
		{
			SrcIPs: []string{"*"},
			DstPorts: []tailcfg.NetPortRange{
				{
					IP: dst.IPv4.String(),
					Ports: tailcfg.PortRange{
						First: 80,
						Last:  80,
					},
				},
				{
					IP: dst.IPv6.String(),
					Ports: tailcfg.PortRange{
						First: 80,
						Last:  80,
					},
				},
			},
		},
	}

	assert.Equal(t, expectedRules, actualRules)
}

func TestACLPolicy_BuildFilterRulesAutogroupMember(t *testing.T) {
	p1 := createMachine("jane@example.com")
	p2 := createMachine("jane@example.com", "tag:web")

	policy := ACLPolicy{
		ACLs: []ACL{
			{
				Action: "accept",
				Src:    []string{"autogroup:members"},
				Dst:    []string{"*:*"},
			},
		},
	}

	dst := createMachine("john@example.com")

	actualRules := policy.BuildFilterRules([]Machine{*p1, *p2}, dst)
	expectedRules := []tailcfg.FilterRule{
		{
			SrcIPs: []string{
				p1.IPv4.String(),
				p1.IPv6.String(),
			},
			DstPorts: []tailcfg.NetPortRange{
				{
					IP: "*",
					Ports: tailcfg.PortRange{
						First: 0,
						Last:  65535,
					},
				},
			},
		},
	}

	assert.Equal(t, expectedRules, actualRules)
}

func TestACLPolicy_BuildFilterRulesAutogroupInternet(t *testing.T) {
	p1 := createMachine("nick@example.com")
	p2 := createMachine("jane@example.com")

	policy := ACLPolicy{
		ACLs: []ACL{
			{
				Action: "accept",
				Src:    []string{"nick@example.com"},
				Dst:    []string{"autogroup:internet:*"},
			},
		},
	}

	dst := createMachine("john@example.com")
	dst.AllowIPs = []netip.Prefix{
		netip.MustParsePrefix("0.0.0.0/0"),
	}

	expectedDstPorts := []tailcfg.NetPortRange{}
	for _, r := range autogroupInternetRanges() {
		expectedDstPorts = append(expectedDstPorts, tailcfg.NetPortRange{
			IP: r,
			Ports: tailcfg.PortRange{
				First: 0,
				Last:  65535,
			},
		})
	}

	actualRules := policy.BuildFilterRules([]Machine{*p1, *p2}, dst)
	expectedRules := []tailcfg.FilterRule{
		{
			SrcIPs: []string{
				p1.IPv4.String(),
				p1.IPv6.String(),
			},
			DstPorts: expectedDstPorts,
		},
	}

	assert.Equal(t, expectedRules, actualRules)
}

func TestWithUser(t *testing.T) {
	policy := ACLPolicy{
		ACLs: []ACL{
			{
				Action: "accept",
				Src:    []string{"*"},
				Dst:    []string{"john@example.com:*"},
			},
		},
	}

	src := createMachine("john@example.com")
	assert.True(t, policy.IsValidPeer(src, createMachine("john@example.com")))
	assert.False(t, policy.IsValidPeer(src, createMachine("john@example.com", "tag:web")))
	assert.False(t, policy.IsValidPeer(src, createMachine("jane@example.com")))
}

func TestWithGroup(t *testing.T) {
	policy := ACLPolicy{
		Groups: map[string][]string{
			"group:admin": {"john@example.com"},
		},
		ACLs: []ACL{
			{
				Action: "accept",
				Src:    []string{"*"},
				Dst:    []string{"group:admin:*"},
			},
		},
	}

	src := createMachine("john@example.com")
	assert.True(t, policy.IsValidPeer(src, createMachine("john@example.com")))
	assert.False(t, policy.IsValidPeer(src, createMachine("jane@example.com")))
}

func TestWithTags(t *testing.T) {
	policy := ACLPolicy{
		ACLs: []ACL{
			{
				Action: "accept",
				Src:    []string{"*"},
				Dst:    []string{"tag:web:*"},
			},
		},
	}

	src := createMachine("john@example.com")

	assert.True(t, policy.IsValidPeer(src, createMachine("john@example.com", "tag:web")))
	assert.False(t, policy.IsValidPeer(src, createMachine("john@example.com", "tag:ci")))
}

func TestWithHosts(t *testing.T) {
	dst1 := createMachine("john@example.com")
	dst2 := createMachine("john@example.com")

	policy := ACLPolicy{
		Hosts: map[string]string{
			"dst1": dst1.IPv4.String(),
		},
		ACLs: []ACL{

			{
				Action: "accept",
				Src:    []string{"*"},
				Dst:    []string{"dst1:*"},
			},
		},
	}

	src := createMachine("jane@example.com")

	assert.True(t, policy.IsValidPeer(src, dst1))
	assert.False(t, policy.IsValidPeer(src, dst2))
}

func createMachine(user string, tags ...string) *Machine {
	ipv4, ipv6, err := addr.SelectIP(func(addr netip.Addr) (bool, error) {
		return true, nil
	})
	if err != nil {
		return nil
	}
	return &Machine{
		IPv4: IP{ipv4},
		IPv6: IP{ipv6},
		User: User{
			Name: user,
		},
		Tags: tags,
	}
}

func TestACLPolicy_IsTagOwner(t *testing.T) {
	policy := ACLPolicy{
		Groups: map[string][]string{
			"group:engineers": {"jane@example.com"},
		},
		TagOwners: map[string][]string{
			"tag:web": {"john@example.com", "group:engineers"},
		}}

	testCases := []struct {
		name      string
		tag       string
		userName  string
		userType  UserType
		expectErr bool
	}{
		{
			name:      "system admin is always a valid owner",
			tag:       "tag:web",
			userName:  "system admin",
			userType:  UserTypeService,
			expectErr: false,
		},
		{
			name:      "system admin is always a valid owner",
			tag:       "tag:unknown",
			userName:  "system admin",
			userType:  UserTypeService,
			expectErr: false,
		},
		{
			name:      "direct tag owner",
			tag:       "tag:web",
			userName:  "john@example.com",
			userType:  UserTypePerson,
			expectErr: false,
		},
		{
			name:      "owner by group",
			tag:       "tag:web",
			userName:  "jane@example.com",
			userType:  UserTypePerson,
			expectErr: false,
		},
		{
			name:      "unknown owner",
			tag:       "tag:web",
			userName:  "nick@example.com",
			userType:  UserTypePerson,
			expectErr: true,
		},
		{
			name:      "unknown tag",
			tag:       "tag:unknown",
			userName:  "jane@example.com",
			userType:  UserTypePerson,
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := policy.CheckTagOwners([]string{tc.tag}, &User{Name: tc.userName, UserType: tc.userType})
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestACLPolicy_FindAutoApprovedIPs(t *testing.T) {
	route1 := netip.MustParsePrefix("10.160.0.0/20")
	route2 := netip.MustParsePrefix("10.161.0.0/20")
	route3 := netip.MustParsePrefix("10.162.0.0/20")

	policy := ACLPolicy{
		Groups: map[string][]string{
			"group:admins": {"jane@example.com"},
		},
		AutoApprovers: AutoApprovers{
			Routes: map[string][]string{
				route1.String(): {"group:admins"},
				route2.String(): {"john@example.com", "tag:router"},
			},
			ExitNode: []string{"nick@example.com"},
		},
	}

	testCases := []struct {
		name        string
		tag         []string
		userName    string
		routableIPs []netip.Prefix
		expected    []netip.Prefix
	}{
		{
			name:        "nil",
			tag:         []string{},
			userName:    "john@example.com",
			routableIPs: nil,
			expected:    nil,
		},
		{
			name:        "empty",
			tag:         []string{},
			userName:    "john@example.com",
			routableIPs: []netip.Prefix{},
			expected:    nil,
		},
		{
			name:        "by user",
			tag:         []string{},
			userName:    "john@example.com",
			routableIPs: []netip.Prefix{route1, route2, route3},
			expected:    []netip.Prefix{route2},
		},
		{
			name:        "partial by user",
			tag:         []string{},
			userName:    "john@example.com",
			routableIPs: []netip.Prefix{netip.MustParsePrefix("10.161.4.0/22")},
			expected:    []netip.Prefix{netip.MustParsePrefix("10.161.4.0/22")},
		},
		{
			name:        "by tag",
			tag:         []string{"tag:router"},
			routableIPs: []netip.Prefix{route1, route2, route3},
			expected:    []netip.Prefix{route2},
		},
		{
			name:        "by group",
			userName:    "jane@example.com",
			routableIPs: []netip.Prefix{route1, route2, route3},
			expected:    []netip.Prefix{route1},
		},
		{
			name:        "no match",
			userName:    "nick@example.com",
			routableIPs: []netip.Prefix{route1, route2, route3},
			expected:    []netip.Prefix{},
		},
		{
			name:        "exit",
			userName:    "nick@example.com",
			routableIPs: []netip.Prefix{netip.MustParsePrefix("0.0.0.0/0")},
			expected:    []netip.Prefix{netip.MustParsePrefix("0.0.0.0/0")},
		},
		{
			name:        "exit no match",
			userName:    "john@example.com",
			routableIPs: []netip.Prefix{netip.MustParsePrefix("0.0.0.0/0")},
			expected:    []netip.Prefix{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualAllowedIPs := policy.FindAutoApprovedIPs(tc.routableIPs, tc.tag, &User{Name: tc.userName})
			assert.Equal(t, tc.expected, actualAllowedIPs)
		})
	}
}
