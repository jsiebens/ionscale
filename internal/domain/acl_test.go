package domain

import (
	"encoding/json"
	"github.com/jsiebens/ionscale/internal/addr"
	"github.com/jsiebens/ionscale/pkg/client/ionscale"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/netip"
	"sort"
	"tailscale.com/tailcfg"
	"testing"
)

func TestACLPolicy_NodeAttributesWithWildcards(t *testing.T) {
	p1 := createMachine("john@example.com")

	policy := ACLPolicy{
		ionscale.ACLPolicy{
			NodeAttrs: []ionscale.ACLNodeAttrGrant{
				{
					Target: []string{"*"},
					Attr: []string{
						"attr1",
						"attr2",
					},
				},
				{
					Target: []string{"*"},
					Attr: []string{
						"attr3",
					},
				},
			},
		},
	}

	actualAttrs := policy.NodeCapabilities(p1)
	expectedAttrs := []tailcfg.NodeCapability{
		tailcfg.NodeCapability("attr1"),
		tailcfg.NodeCapability("attr2"),
		tailcfg.NodeCapability("attr3"),
	}

	assert.Equal(t, expectedAttrs, actualAttrs)
}

func TestACLPolicy_NodeAttributesWithUserAndGroups(t *testing.T) {
	p1 := createMachine("john@example.com")

	policy := ACLPolicy{
		ionscale.ACLPolicy{
			Groups: map[string][]string{
				"group:admins": []string{"john@example.com"},
			},
			NodeAttrs: []ionscale.ACLNodeAttrGrant{
				{
					Target: []string{"john@example.com"},
					Attr: []string{
						"attr1",
						"attr2",
					},
				},
				{
					Target: []string{"jane@example.com", "group:analytics", "group:admins"},
					Attr: []string{
						"attr3",
					},
				},
			},
		},
	}

	actualAttrs := policy.NodeCapabilities(p1)
	expectedAttrs := []tailcfg.NodeCapability{
		tailcfg.NodeCapability("attr1"),
		tailcfg.NodeCapability("attr2"),
		tailcfg.NodeCapability("attr3"),
	}

	assert.Equal(t, expectedAttrs, actualAttrs)
}

func TestACLPolicy_NodeAttributesWithUserAndTags(t *testing.T) {
	p1 := createMachine("john@example.com", "tag:web")

	policy := ACLPolicy{
		ionscale.ACLPolicy{
			Groups: map[string][]string{
				"group:admins": []string{"john@example.com"},
			},
			NodeAttrs: []ionscale.ACLNodeAttrGrant{
				{
					Target: []string{"john@example.com"},
					Attr: []string{
						"attr1",
						"attr2",
					},
				},
				{
					Target: []string{"jane@example.com", "tag:web"},
					Attr: []string{
						"attr3",
					},
				},
			},
		},
	}

	actualAttrs := policy.NodeCapabilities(p1)
	expectedAttrs := []tailcfg.NodeCapability{tailcfg.NodeCapability("attr3")}

	assert.Equal(t, expectedAttrs, actualAttrs)
}

func TestACLPolicy_BuildFilterRulesEmptyACL(t *testing.T) {
	p1 := createMachine("john@example.com")
	p2 := createMachine("jane@example.com")

	policy := ACLPolicy{
		ionscale.ACLPolicy{
			ACLs: []ionscale.ACLEntry{},
		},
	}

	dst := createMachine("john@example.com")

	actualRules := policy.BuildFilterRules([]Machine{*p1, *p2}, dst)
	expectedRules := []tailcfg.FilterRule{}

	assert.Equal(t, expectedRules, actualRules)
}

func TestACLPolicy_BuildFilterRulesWildcards(t *testing.T) {
	p1 := createMachine("john@example.com")
	p2 := createMachine("jane@example.com")

	policy := ACLPolicy{
		ionscale.ACLPolicy{
			ACLs: []ionscale.ACLEntry{
				{
					Action:      "accept",
					Source:      []string{"*"},
					Destination: []string{"*:*"},
				},
			},
		},
	}

	dst := createMachine("john@example.com")

	actualRules := policy.BuildFilterRules([]Machine{*p1, *p2}, dst)
	expectedRules := []tailcfg.FilterRule{
		{
			SrcIPs: expectedSourceIPs(p1, p2),
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

func TestACLPolicy_BuildFilterRulesProto(t *testing.T) {
	p1 := createMachine("john@example.com")
	p2 := createMachine("jane@example.com")

	policy := ACLPolicy{
		ionscale.ACLPolicy{
			ACLs: []ionscale.ACLEntry{
				{
					Action:      "accept",
					Source:      []string{"*"},
					Destination: []string{"*:22"},
				},
				{
					Action:      "accept",
					Source:      []string{"*"},
					Destination: []string{"*:*"},
					Protocol:    "igmp",
				},
			},
		},
	}

	dst := createMachine("john@example.com")

	actualRules := policy.BuildFilterRules([]Machine{*p1, *p2}, dst)
	expectedRules := []tailcfg.FilterRule{
		{
			SrcIPs: expectedSourceIPs(p1, p2),
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
			SrcIPs: expectedSourceIPs(p1, p2),
			DstPorts: []tailcfg.NetPortRange{
				{
					IP: "*",
					Ports: tailcfg.PortRange{
						First: 0,
						Last:  65535,
					},
				},
			},
			IPProto: []int{protocolIGMP},
		},
	}

	assert.Equal(t, expectedRules, actualRules)
}

func TestACLPolicy_BuildFilterRulesWithGroups(t *testing.T) {
	p1 := createMachine("jane@example.com")
	p2 := createMachine("nick@example.com")
	p3 := createMachine("joe@example.com")

	policy := ACLPolicy{
		ionscale.ACLPolicy{
			Groups: map[string][]string{
				"group:admin": []string{"jane@example.com"},
				"group:audit": []string{"nick@example.com"},
			},
			ACLs: []ionscale.ACLEntry{
				{
					Action:      "accept",
					Source:      []string{"group:admin"},
					Destination: []string{"*:22"},
				},
				{
					Action:      "accept",
					Source:      []string{"group:audit"},
					Destination: []string{"*:8000-8080"},
				},
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
		ionscale.ACLPolicy{
			ACLs: []ionscale.ACLEntry{
				{
					Action:      "accept",
					Source:      []string{"autogroup:members"},
					Destination: []string{"*:22"},
				},
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

func TestACLPolicy_BuildFilterRulesWithAutoGroupMember(t *testing.T) {
	p1 := createMachine("jane@example.com")
	p2 := createMachine("nick@example.com")
	p3 := createMachine("joe@example.com", "tag:web")

	policy := ACLPolicy{
		ionscale.ACLPolicy{
			ACLs: []ionscale.ACLEntry{
				{
					Action:      "accept",
					Source:      []string{"autogroup:member"},
					Destination: []string{"*:22"},
				},
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

func TestACLPolicy_BuildFilterRulesWithAutoGroupTagged(t *testing.T) {

	p1 := createMachine("jane@example.com")
	p2 := createMachine("nick@example.com")
	p3 := createMachine("joe@example.com", "tag:web")

	policy := ACLPolicy{
		ionscale.ACLPolicy{
			ACLs: []ionscale.ACLEntry{
				{
					Action:      "accept",
					Source:      []string{"autogroup:tagged"},
					Destination: []string{"*:22"},
				},
			},
		},
	}

	dst := createMachine("john@example.com")

	actualRules := policy.BuildFilterRules([]Machine{*p1, *p2, *p3}, dst)

	expectedSrcIPs := []string{
		p3.IPv4.String(), p3.IPv6.String(),
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
		ionscale.ACLPolicy{
			ACLs: []ionscale.ACLEntry{
				{
					Action:      "accept",
					Source:      []string{"*"},
					Destination: []string{"autogroup:self:*"},
				},
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
		ionscale.ACLPolicy{
			ACLs: []ionscale.ACLEntry{
				{
					Action:      "accept",
					Source:      []string{"*"},
					Destination: []string{"autogroup:self:*"},
				},
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
		ionscale.ACLPolicy{
			ACLs: []ionscale.ACLEntry{
				{
					Action:      "accept",
					Source:      []string{"*"},
					Destination: []string{"autogroup:self:22", "john@example.com:80"},
				},
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
			SrcIPs: expectedSourceIPs(p1, p2, p3),
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

func TestACLPolicy_BuildFilterRulesAutogroupInternet(t *testing.T) {
	p1 := createMachine("nick@example.com")
	p2 := createMachine("jane@example.com")

	policy := ACLPolicy{
		ionscale.ACLPolicy{
			ACLs: []ionscale.ACLEntry{
				{
					Action:      "accept",
					Source:      []string{"nick@example.com"},
					Destination: []string{"autogroup:internet:*"},
				},
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

func TestACLPolicy_BuildFilterRulesAutogroupDangerAll(t *testing.T) {
	p1 := createMachine("nick@example.com")
	p2 := createMachine("jane@example.com")

	policy := ACLPolicy{
		ionscale.ACLPolicy{
			ACLs: []ionscale.ACLEntry{
				{
					Action:      "accept",
					Source:      []string{"autogroup:danger-all"},
					Destination: []string{"*:*"},
				},
			},
		},
	}

	dst := createMachine("john@example.com")

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
				"0.0.0.0/0", "::/0",
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

func TestWithUser(t *testing.T) {
	policy := ACLPolicy{
		ionscale.ACLPolicy{
			ACLs: []ionscale.ACLEntry{
				{
					Action:      "accept",
					Source:      []string{"*"},
					Destination: []string{"john@example.com:*"},
				},
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
		ionscale.ACLPolicy{
			Groups: map[string][]string{
				"group:admin": {"john@example.com"},
			},
			ACLs: []ionscale.ACLEntry{
				{
					Action:      "accept",
					Source:      []string{"*"},
					Destination: []string{"group:admin:*"},
				},
			},
		},
	}

	src := createMachine("john@example.com")
	assert.True(t, policy.IsValidPeer(src, createMachine("john@example.com")))
	assert.False(t, policy.IsValidPeer(src, createMachine("jane@example.com")))
}

func TestWithTags(t *testing.T) {
	policy := ACLPolicy{
		ionscale.ACLPolicy{
			ACLs: []ionscale.ACLEntry{
				{
					Action:      "accept",
					Source:      []string{"*"},
					Destination: []string{"tag:web:*"},
				},
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
		ionscale.ACLPolicy{
			Hosts: map[string]string{
				"dst1": dst1.IPv4.String(),
			},
			ACLs: []ionscale.ACLEntry{

				{
					Action:      "accept",
					Source:      []string{"*"},
					Destination: []string{"dst1:*"},
				},
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

func expectedSourceIPs(m ...*Machine) []string {
	x := &StringSet{}
	for _, m := range m {
		x = x.Add(m.IPv4.String(), m.IPv6.String())
	}
	return x.Items()
}

func TestACLPolicy_IsTagOwner(t *testing.T) {
	policy := ACLPolicy{
		ionscale.ACLPolicy{
			Groups: map[string][]string{
				"group:engineers": {"jane@example.com"},
			},
			TagOwners: map[string][]string{
				"tag:web": {"john@example.com", "group:engineers"},
			}}}

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

func TestACLPolicy_FindAutoApprovedIPsWhenNoAutoapproversAreSet(t *testing.T) {
	route1 := netip.MustParsePrefix("10.160.0.0/20")
	route2 := netip.MustParsePrefix("10.161.0.0/20")
	route3 := netip.MustParsePrefix("10.162.0.0/20")

	policy := ACLPolicy{}
	assert.Nil(t, policy.FindAutoApprovedIPs([]netip.Prefix{route1, route2, route3}, nil, nil))
}

func TestACLPolicy_FindAutoApprovedIPs(t *testing.T) {
	route1 := netip.MustParsePrefix("10.160.0.0/20")
	route2 := netip.MustParsePrefix("10.161.0.0/20")
	route3 := netip.MustParsePrefix("10.162.0.0/20")

	policy := ACLPolicy{
		ionscale.ACLPolicy{
			Groups: map[string][]string{
				"group:admins": {"jane@example.com"},
			},
			AutoApprovers: &ionscale.ACLAutoApprovers{
				Routes: map[string][]string{
					route1.String(): {"group:admins"},
					route2.String(): {"john@example.com", "tag:router"},
				},
				ExitNode: []string{"nick@example.com"},
			},
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
			expected:    nil,
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
			expected:    nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualAllowedIPs := policy.FindAutoApprovedIPs(tc.routableIPs, tc.tag, &User{Name: tc.userName})
			assert.Equal(t, tc.expected, actualAllowedIPs)
		})
	}
}

func TestACLPolicy_BuildFilterRulesWithAdvertisedRoutes(t *testing.T) {
	route1 := netip.MustParsePrefix("fd7a:115c:a1e0:b1a:0:1:a3c:0/120")
	p1 := createMachine("john@example.com", "tag:trusted")

	policy := ACLPolicy{
		ionscale.ACLPolicy{
			ACLs: []ionscale.ACLEntry{
				{
					Action:      "accept",
					Source:      []string{"tag:trusted"},
					Destination: []string{"fd7a:115c:a1e0:b1a:0:1:a3c:0/120:*"},
				},
			},
		},
	}

	dst := createMachine("john@example.com")
	dst.AllowIPs = []netip.Prefix{route1}

	actualRules := policy.BuildFilterRules([]Machine{*p1}, dst)
	expectedRules := []tailcfg.FilterRule{
		{
			SrcIPs: p1.IPs(),
			DstPorts: []tailcfg.NetPortRange{
				{
					IP: route1.String(),
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

func TestACLPolicy_BuildFilterRulesWildcardGrants(t *testing.T) {
	ranges, err := tailcfg.ParseProtoPortRanges([]string{"*"})
	require.NoError(t, err)

	p1 := createMachine("john@example.com")
	p2 := createMachine("jane@example.com")

	policy := ACLPolicy{
		ionscale.ACLPolicy{
			Grants: []ionscale.ACLGrant{
				{
					Source:      []string{"*"},
					Destination: []string{"*"},
					IP:          ranges,
				},
			},
		},
	}

	dst := createMachine("john@example.com")

	actualRules := policy.BuildFilterRules([]Machine{*p1, *p2}, dst)
	expectedRules := []tailcfg.FilterRule{
		{
			SrcIPs: expectedSourceIPs(p1, p2),
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

func TestACLPolicy_BuildFilterRulesWithAppGrants(t *testing.T) {
	p1 := createMachine("john@example.com")
	p2 := createMachine("jane@example.com")

	dst := createMachine("john@example.com")

	mycap := map[string]interface{}{
		"channel": "alpha",
		"ids":     []string{"1", "2", "3"},
	}

	marshal, _ := json.Marshal(mycap)

	policy := ACLPolicy{
		ionscale.ACLPolicy{
			Grants: []ionscale.ACLGrant{
				{
					Source:      []string{"*"},
					Destination: []string{"*"},
					App: map[tailcfg.PeerCapability][]tailcfg.RawMessage{
						tailcfg.PeerCapability("localtest.me/cap/test"): {tailcfg.RawMessage(marshal)},
					},
				},
			},
		},
	}

	actualRules := policy.BuildFilterRules([]Machine{*p1, *p2}, dst)
	expectedRules := []tailcfg.FilterRule{
		{
			SrcIPs: expectedSourceIPs(p1, p2),
			CapGrant: []tailcfg.CapGrant{
				{
					Dsts: []netip.Prefix{
						netip.PrefixFrom(*dst.IPv4.Addr, 32),
						netip.PrefixFrom(*dst.IPv6.Addr, 128),
					},
					CapMap: map[tailcfg.PeerCapability][]tailcfg.RawMessage{
						tailcfg.PeerCapability("localtest.me/cap/test"): {tailcfg.RawMessage(marshal)},
					},
				},
			},
		},
	}

	assert.Equal(t, expectedRules, actualRules)
}
