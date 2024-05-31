package domain

import (
	"github.com/jsiebens/ionscale/pkg/client/ionscale"
	"github.com/stretchr/testify/assert"
	"tailscale.com/tailcfg"
	"testing"
)

func TestACLPolicy_BuildSSHPolicy_(t *testing.T) {
	p1 := createMachine("john@example.com")
	p2 := createMachine("jane@example.com")

	policy := ACLPolicy{
		ionscale.ACLPolicy{
			SSH: []ionscale.ACLSSH{
				{
					Action:      "accept",
					Source:      []string{"autogroup:members"},
					Destination: []string{"autogroup:self"},
					Users:       []string{"autogroup:nonroot"},
				},
			},
		},
	}

	dst := createMachine("john@example.com")

	actualRules := policy.BuildSSHPolicy([]Machine{*p1, *p2}, dst)
	expectedRules := []*tailcfg.SSHRule{
		{
			Principals: []*tailcfg.SSHPrincipal{
				{NodeIP: p1.IPv4.String()},
				{NodeIP: p1.IPv6.String()},
			},
			SSHUsers: map[string]string{
				"*":    "=",
				"root": "",
			},
			Action: &tailcfg.SSHAction{
				Accept:                   true,
				AllowAgentForwarding:     true,
				AllowLocalPortForwarding: true,
			},
		},
	}

	assert.Equal(t, expectedRules, actualRules.Rules)
}

func TestACLPolicy_BuildSSHPolicy_WithGroup(t *testing.T) {
	p1 := createMachine("john@example.com")
	p2 := createMachine("jane@example.com")

	policy := ACLPolicy{
		ionscale.ACLPolicy{
			Groups: map[string][]string{
				"group:sre": {
					"john@example.com",
				},
			},
			SSH: []ionscale.ACLSSH{
				{
					Action:      "accept",
					Source:      []string{"group:sre"},
					Destination: []string{"tag:web"},
					Users:       []string{"autogroup:nonroot", "root"},
				},
			},
		},
	}

	dst := createMachine("john@example.com", "tag:web")

	actualRules := policy.BuildSSHPolicy([]Machine{*p1, *p2}, dst)
	expectedRules := []*tailcfg.SSHRule{
		{
			Principals: []*tailcfg.SSHPrincipal{
				{NodeIP: p1.IPv4.String()},
				{NodeIP: p1.IPv6.String()},
			},
			SSHUsers: map[string]string{
				"*":    "=",
				"root": "root",
			},
			Action: &tailcfg.SSHAction{
				Accept:                   true,
				AllowAgentForwarding:     true,
				AllowLocalPortForwarding: true,
			},
		},
	}

	assert.Equal(t, expectedRules, actualRules.Rules)
}

func TestACLPolicy_BuildSSHPolicy_WithMatchingUsers(t *testing.T) {
	p1 := createMachine("john@example.com")
	p2 := createMachine("jane@example.com")

	policy := ACLPolicy{
		ionscale.ACLPolicy{
			SSH: []ionscale.ACLSSH{
				{
					Action:      "accept",
					Source:      []string{"john@example.com"},
					Destination: []string{"john@example.com"},
					Users:       []string{"autogroup:nonroot", "root"},
				},
			},
		},
	}

	dst := createMachine("john@example.com")

	actualRules := policy.BuildSSHPolicy([]Machine{*p1, *p2}, dst)
	expectedRules := []*tailcfg.SSHRule{
		{
			Principals: sshPrincipalsFromMachines(*p1),
			SSHUsers: map[string]string{
				"*":    "=",
				"root": "root",
			},
			Action: &tailcfg.SSHAction{
				Accept:                   true,
				AllowAgentForwarding:     true,
				AllowLocalPortForwarding: true,
			},
		},
	}

	assert.Equal(t, expectedRules, actualRules.Rules)
}

func TestACLPolicy_BuildSSHPolicy_WithMatchingUsersInGroup(t *testing.T) {
	p1 := createMachine("john@example.com")
	p2 := createMachine("jane@example.com")

	policy := ACLPolicy{
		ionscale.ACLPolicy{
			Groups: map[string][]string{
				"group:sre": {"jane@example.com", "john@example.com"},
			},
			SSH: []ionscale.ACLSSH{
				{
					Action:      "accept",
					Source:      []string{"group:sre"},
					Destination: []string{"john@example.com"},
					Users:       []string{"autogroup:nonroot", "root"},
				},
			},
		},
	}

	dst := createMachine("john@example.com")

	actualRules := policy.BuildSSHPolicy([]Machine{*p1, *p2}, dst)
	expectedRules := []*tailcfg.SSHRule{
		{
			Principals: sshPrincipalsFromMachines(*p1),
			SSHUsers: map[string]string{
				"*":    "=",
				"root": "root",
			},
			Action: &tailcfg.SSHAction{
				Accept:                   true,
				AllowAgentForwarding:     true,
				AllowLocalPortForwarding: true,
			},
		},
	}

	assert.Equal(t, expectedRules, actualRules.Rules)
}

func TestACLPolicy_BuildSSHPolicy_WithNoMatchingUsers(t *testing.T) {
	p1 := createMachine("john@example.com")
	p2 := createMachine("jane@example.com")

	policy := ACLPolicy{
		ionscale.ACLPolicy{
			SSH: []ionscale.ACLSSH{
				{
					Action:      "accept",
					Source:      []string{"jane@example.com"},
					Destination: []string{"john@example.com"},
					Users:       []string{"autogroup:nonroot", "root"},
				},
			},
		},
	}

	dst := createMachine("john@example.com")

	actualRules := policy.BuildSSHPolicy([]Machine{*p1, *p2}, dst)

	assert.Nil(t, actualRules.Rules)
}

func TestACLPolicy_BuildSSHPolicy_WithTags(t *testing.T) {
	p1 := createMachine("john@example.com")
	p2 := createMachine("nick@example.com")
	p3 := createMachine("nick@example.com", "tag:web")

	policy := ACLPolicy{
		ionscale.ACLPolicy{
			SSH: []ionscale.ACLSSH{
				{
					Action:      "accept",
					Source:      []string{"john@example.com", "tag:web"},
					Destination: []string{"tag:web"},
					Users:       []string{"ubuntu"},
				},
			},
		},
	}

	dst := createMachine("john@example.com", "tag:web")

	actualRules := policy.BuildSSHPolicy([]Machine{*p1, *p2, *p3}, dst)
	expectedRules := []*tailcfg.SSHRule{
		{
			Principals: sshPrincipalsFromMachines(*p1, *p3),
			SSHUsers: map[string]string{
				"ubuntu": "ubuntu",
			},
			Action: &tailcfg.SSHAction{
				Accept:                   true,
				AllowAgentForwarding:     true,
				AllowLocalPortForwarding: true,
			},
		},
	}

	assert.Equal(t, expectedRules, actualRules.Rules)
}

func TestACLPolicy_BuildSSHPolicy_WithTagsInDstAndAutogroupMemberInSrc(t *testing.T) {
	p1 := createMachine("john@example.com")
	p2 := createMachine("nick@example.com")
	p3 := createMachine("nick@example.com", "tag:web")

	policy := ACLPolicy{
		ionscale.ACLPolicy{
			SSH: []ionscale.ACLSSH{
				{
					Action:      "accept",
					Source:      []string{"autogroup:members"},
					Destination: []string{"tag:web"},
					Users:       []string{"ubuntu"},
				},
			},
		},
	}

	dst := createMachine("john@example.com", "tag:web")

	actualRules := policy.BuildSSHPolicy([]Machine{*p1, *p2, *p3}, dst)
	expectedRules := []*tailcfg.SSHRule{
		{
			Principals: sshPrincipalsFromMachines(*p1, *p2),
			SSHUsers: map[string]string{
				"ubuntu": "ubuntu",
			},
			Action: &tailcfg.SSHAction{
				Accept:                   true,
				AllowAgentForwarding:     true,
				AllowLocalPortForwarding: true,
			},
		},
	}

	assert.Equal(t, expectedRules, actualRules.Rules)
}

func TestACLPolicy_BuildSSHPolicy_WithUserInDstAndNonMatchingSrc(t *testing.T) {
	p1 := createMachine("john@example.com")
	p2 := createMachine("jane@example.com")

	policy := ACLPolicy{
		ionscale.ACLPolicy{
			SSH: []ionscale.ACLSSH{
				{
					Action:      "accept",
					Source:      []string{"jane@example.com"},
					Destination: []string{"john@example.com"},
					Users:       []string{"autogroup:nonroot"},
				},
			},
		},
	}

	dst := createMachine("john@example.com")

	actualRules := policy.BuildSSHPolicy([]Machine{*p1, *p2}, dst)

	assert.Nil(t, actualRules.Rules)
}

func TestACLPolicy_BuildSSHPolicy_WithUserInDstAndAutogroupMembersSrc(t *testing.T) {
	p1 := createMachine("john@example.com")
	p2 := createMachine("jane@example.com")

	policy := ACLPolicy{
		ionscale.ACLPolicy{
			SSH: []ionscale.ACLSSH{
				{
					Action:      "accept",
					Source:      []string{"autogroup:members"},
					Destination: []string{"john@example.com"},
					Users:       []string{"autogroup:nonroot"},
				},
			},
		},
	}

	dst := createMachine("john@example.com")

	actualRules := policy.BuildSSHPolicy([]Machine{*p1, *p2}, dst)
	expectedRules := []*tailcfg.SSHRule{
		{
			Principals: sshPrincipalsFromMachines(*p1),
			SSHUsers: map[string]string{
				"*":    "=",
				"root": "",
			},
			Action: &tailcfg.SSHAction{
				Accept:                   true,
				AllowAgentForwarding:     true,
				AllowLocalPortForwarding: true,
			},
		},
	}

	assert.Equal(t, expectedRules, actualRules.Rules)
}

func TestACLPolicy_BuildSSHPolicy_WithAutogroupSelfAndTagSrc(t *testing.T) {
	p1 := createMachine("john@example.com")
	p2 := createMachine("jane@example.com", "tag:web")

	policy := ACLPolicy{
		ionscale.ACLPolicy{
			SSH: []ionscale.ACLSSH{
				{
					Action:      "accept",
					Source:      []string{"tag:web"},
					Destination: []string{"autogroup:self"},
					Users:       []string{"autogroup:nonroot"},
				},
			},
		},
	}

	dst := createMachine("john@example.com")

	actualRules := policy.BuildSSHPolicy([]Machine{*p1, *p2}, dst)

	assert.Nil(t, actualRules.Rules)
}

func TestACLPolicy_BuildSSHPolicy_WithTagsAndActionCheck(t *testing.T) {
	p1 := createMachine("john@example.com")
	p2 := createMachine("jane@example.com", "tag:web")

	policy := ACLPolicy{
		ionscale.ACLPolicy{
			SSH: []ionscale.ACLSSH{
				{
					Action:      "check",
					Source:      []string{"tag:web"},
					Destination: []string{"tag:web"},
					Users:       []string{"autogroup:nonroot"},
				},
			},
		},
	}

	dst := createMachine("john@example.com", "tag:web")

	actualRules := policy.BuildSSHPolicy([]Machine{*p1, *p2}, dst)

	assert.Nil(t, actualRules.Rules)
}

func sshPrincipalsFromMachines(machines ...Machine) []*tailcfg.SSHPrincipal {
	x := StringSet{}
	for _, m := range machines {
		x.Add(m.IPv4.String(), m.IPv6.String())
	}

	var result = []*tailcfg.SSHPrincipal{}

	for _, i := range x.Items() {
		result = append(result, &tailcfg.SSHPrincipal{NodeIP: i})
	}

	return result
}
