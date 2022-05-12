package domain

import (
	"fmt"
	"inet.af/netaddr"
	"strconv"
	"strings"
	"tailscale.com/tailcfg"
)

type ACLPolicy struct {
	Hosts map[string]string `json:"hosts,omitempty"`
	ACLs  []ACL             `json:"acls"`
}

type ACL struct {
	Action string   `json:"action"`
	Src    []string `json:"src"`
	Dst    []string `json:"dst"`
}

func defaultPolicy() ACLPolicy {
	return ACLPolicy{
		ACLs: []ACL{
			{
				Action: "accept",
				Src:    []string{"*"},
				Dst:    []string{"*:*"},
			},
		},
	}
}

type aclEngine struct {
	policy       *ACLPolicy
	expandedTags map[string][]string
}

func IsValidPeer(policy *ACLPolicy, src *Machine, dest *Machine) bool {
	f := &aclEngine{
		policy: policy,
	}
	return f.isValidPeer(src, dest)
}

func BuildFilterRules(policy *ACLPolicy, dst *Machine, peers []Machine) []tailcfg.FilterRule {
	f := &aclEngine{
		policy: policy,
	}
	return f.build(dst, peers)
}

func (a *aclEngine) isValidPeer(src *Machine, dest *Machine) bool {
	for _, acl := range a.policy.ACLs {
		allDestPorts := a.expandMachineToDstPorts(dest, acl.Dst)
		if len(allDestPorts) == 0 {
			continue
		}

		for _, alias := range acl.Src {
			if len(a.expandMachineAlias(src, alias)) != 0 {
				return true
			}
		}
	}
	return false
}

func (a *aclEngine) build(dst *Machine, peers []Machine) []tailcfg.FilterRule {
	var rules []tailcfg.FilterRule

	for _, acl := range a.policy.ACLs {
		allDestPorts := a.expandMachineToDstPorts(dst, acl.Dst)
		if len(allDestPorts) == 0 {
			continue
		}

		var allSrcIPs []string
		for _, src := range acl.Src {
			for _, peer := range peers {
				srcIPs := a.expandMachineAlias(&peer, src)
				allSrcIPs = append(allSrcIPs, srcIPs...)
			}
		}

		if len(allSrcIPs) == 0 {
			allSrcIPs = nil
		}

		rule := tailcfg.FilterRule{
			SrcIPs:   allSrcIPs,
			DstPorts: allDestPorts,
		}

		rules = append(rules, rule)
	}

	if len(rules) == 0 {
		return []tailcfg.FilterRule{{}}
	}

	return rules
}

func (a *aclEngine) expandMachineToDstPorts(m *Machine, ports []string) []tailcfg.NetPortRange {
	allDestRanges := []tailcfg.NetPortRange{}
	for _, d := range ports {
		ranges := a.expandMachineDestToNetPortRanges(m, d)
		allDestRanges = append(allDestRanges, ranges...)
	}
	return allDestRanges
}

func (a *aclEngine) expandMachineDestToNetPortRanges(m *Machine, dest string) []tailcfg.NetPortRange {
	tokens := strings.Split(dest, ":")
	if len(tokens) < 2 || len(tokens) > 3 {
		return nil
	}

	var alias string
	if len(tokens) == 2 {
		alias = tokens[0]
	} else {
		alias = fmt.Sprintf("%s:%s", tokens[0], tokens[1])
	}

	ports, err := a.expandValuePortToPortRange(tokens[len(tokens)-1])
	if err != nil {
		return nil
	}

	ips := a.expandMachineAlias(m, alias)
	if len(ips) == 0 {
		return nil
	}

	dests := []tailcfg.NetPortRange{}
	for _, d := range ips {
		for _, p := range ports {
			pr := tailcfg.NetPortRange{
				IP:    d,
				Ports: p,
			}
			dests = append(dests, pr)
		}
	}

	return dests
}

func (a *aclEngine) expandMachineAlias(m *Machine, src string) []string {
	if src == "*" {
		if src == "*" {
			return []string{"*"}
		}
	}

	machineIPs := []string{m.IPv4.String(), m.IPv6.String()}

	if strings.HasPrefix(src, "tag:") && m.HasTag(src[4:]) {
		return machineIPs
	}

	if h, ok := a.policy.Hosts[src]; ok {
		src = h
	}

	ip, err := netaddr.ParseIP(src)
	if err == nil && m.HasIP(ip) {
		return machineIPs
	}

	return []string{}
}

func (a *aclEngine) expandValuePortToPortRange(s string) ([]tailcfg.PortRange, error) {
	if s == "*" {
		return []tailcfg.PortRange{{First: 0, Last: 65535}}, nil
	}

	ports := []tailcfg.PortRange{}
	for _, p := range strings.Split(s, ",") {
		rang := strings.Split(p, "-")
		if len(rang) == 1 {
			pi, err := strconv.ParseUint(rang[0], 10, 16)
			if err != nil {
				return nil, err
			}
			ports = append(ports, tailcfg.PortRange{
				First: uint16(pi),
				Last:  uint16(pi),
			})
		} else if len(rang) == 2 {
			start, err := strconv.ParseUint(rang[0], 10, 16)
			if err != nil {
				return nil, err
			}
			last, err := strconv.ParseUint(rang[1], 10, 16)
			if err != nil {
				return nil, err
			}
			ports = append(ports, tailcfg.PortRange{
				First: uint16(start),
				Last:  uint16(last),
			})
		} else {
			return nil, fmt.Errorf("invalid format")
		}
	}
	return ports, nil
}
