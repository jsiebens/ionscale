package domain

import (
	"github.com/jsiebens/ionscale/pkg/client/ionscale"
	"net/netip"
	"strings"
	"tailscale.com/tailcfg"
)

func (a ACLPolicy) IsValidPeer(src *Machine, dest *Machine) bool {
	if !src.HasTags() && !dest.HasTags() && dest.HasUser(src.User.Name) {
		return true
	}

	for _, acl := range a.ACLs {
		selfDestPorts, allDestPorts := a.translateDestinationAliasesToMachineNetPortRanges(acl.Destination, dest)
		if len(selfDestPorts) != 0 {
			for _, alias := range acl.Source {
				if len(a.translateSourceAliasToMachineIPs(alias, src, &dest.User)) != 0 {
					return true
				}
			}
		}
		if len(allDestPorts) != 0 {
			for _, alias := range acl.Source {
				if len(a.translateSourceAliasToMachineIPs(alias, src, nil)) != 0 {
					return true
				}
			}
		}
	}

	for _, grant := range a.Grants {
		selfIps, otherIps := a.translateDestinationAliasesToMachineIPs(grant.Destination, dest)
		if len(selfIps) != 0 {
			for _, alias := range grant.Source {
				if len(a.translateSourceAliasToMachineIPs(alias, src, &dest.User)) != 0 {
					return true
				}
			}
		}
		if len(otherIps) != 0 {
			for _, alias := range grant.Source {
				if len(a.translateSourceAliasToMachineIPs(alias, src, nil)) != 0 {
					return true
				}
			}
		}
	}

	return false
}

func (a ACLPolicy) BuildFilterRules(peers []Machine, dst *Machine) []tailcfg.FilterRule {
	var rules = make([]tailcfg.FilterRule, 0)

	matchSourceAndAppendRule := func(rules []tailcfg.FilterRule, aliases []string, preparedRules []tailcfg.FilterRule, u *User) []tailcfg.FilterRule {
		if len(preparedRules) == 0 {
			return rules
		}

		var allSrcIPsSet = &StringSet{}
		for _, alias := range aliases {
			for _, peer := range peers {
				allSrcIPsSet.Add(a.translateSourceAliasToMachineIPs(alias, &peer, u)...)
			}
		}

		if allSrcIPsSet.Empty() {
			return rules
		}

		allSrcIPs := allSrcIPsSet.Items()

		if len(allSrcIPs) == 0 {
			return rules
		}

		for _, pr := range preparedRules {
			rules = append(rules, tailcfg.FilterRule{
				SrcIPs:   allSrcIPs,
				DstPorts: pr.DstPorts,
				IPProto:  pr.IPProto,
				CapGrant: pr.CapGrant,
			})
		}

		return rules
	}

	for _, acl := range a.ACLs {
		self, other := a.prepareFilterRulesFromACL(dst, acl)
		rules = matchSourceAndAppendRule(rules, acl.Source, self, &dst.User)
		rules = matchSourceAndAppendRule(rules, acl.Source, other, nil)
	}

	for _, acl := range a.Grants {
		self, other := a.prepareFilterRulesFromGrant(dst, acl)
		rules = matchSourceAndAppendRule(rules, acl.Source, self, &dst.User)
		rules = matchSourceAndAppendRule(rules, acl.Source, other, nil)
	}

	return rules
}

func (a ACLPolicy) prepareFilterRulesFromACL(candidate *Machine, acl ionscale.ACLEntry) ([]tailcfg.FilterRule, []tailcfg.FilterRule) {
	proto := parseProtocol(acl.Protocol)

	selfDstPorts, otherDstPorts := a.translateDestinationAliasesToMachineNetPortRanges(acl.Destination, candidate)

	var selfFilterRules []tailcfg.FilterRule
	var otherFilterRules []tailcfg.FilterRule

	if len(selfDstPorts) != 0 {
		selfFilterRules = append(selfFilterRules, tailcfg.FilterRule{IPProto: proto, DstPorts: selfDstPorts})
	}

	if len(otherDstPorts) != 0 {
		otherFilterRules = append(otherFilterRules, tailcfg.FilterRule{IPProto: proto, DstPorts: otherDstPorts})
	}

	return selfFilterRules, otherFilterRules
}

func (a ACLPolicy) prepareFilterRulesFromGrant(candidate *Machine, grant ionscale.ACLGrant) ([]tailcfg.FilterRule, []tailcfg.FilterRule) {
	selfIPs, otherIPs := a.translateDestinationAliasesToMachineIPs(grant.Destination, candidate)

	var selfFilterRules []tailcfg.FilterRule
	var otherFilterRules []tailcfg.FilterRule

	for _, ip := range grant.IP {
		if len(selfIPs) != 0 {
			ranges := make([]tailcfg.NetPortRange, len(selfIPs))
			for i, s := range selfIPs {
				ranges[i] = tailcfg.NetPortRange{IP: s, Ports: ip.Ports}
			}

			rule := tailcfg.FilterRule{DstPorts: ranges}
			if ip.Proto != 0 {
				rule.IPProto = []int{ip.Proto}
			}

			selfFilterRules = append(selfFilterRules, rule)
		}

		if len(otherIPs) != 0 {
			ranges := make([]tailcfg.NetPortRange, len(otherIPs))
			for i, s := range otherIPs {
				ranges[i] = tailcfg.NetPortRange{IP: s, Ports: ip.Ports}
			}

			rule := tailcfg.FilterRule{DstPorts: ranges}
			if ip.Proto != 0 {
				rule.IPProto = []int{ip.Proto}
			}

			otherFilterRules = append(otherFilterRules, rule)
		}
	}

	if len(grant.App) != 0 {
		selfPrefixes, otherPrefixes := appGrantDstIpsToPrefixes(candidate, selfIPs, otherIPs)
		if len(selfPrefixes) != 0 {
			rule := tailcfg.FilterRule{CapGrant: []tailcfg.CapGrant{{Dsts: selfPrefixes, CapMap: grant.App}}}
			selfFilterRules = append(selfFilterRules, rule)
		}

		if len(otherPrefixes) != 0 {
			rule := tailcfg.FilterRule{CapGrant: []tailcfg.CapGrant{{Dsts: otherPrefixes, CapMap: grant.App}}}
			otherFilterRules = append(otherFilterRules, rule)
		}
	}

	return selfFilterRules, otherFilterRules
}

func appGrantDstIpsToPrefixes(m *Machine, self []string, other []string) ([]netip.Prefix, []netip.Prefix) {
	translate := func(ips []string) []netip.Prefix {
		var prefixes []netip.Prefix
		for _, ip := range ips {
			if ip == "*" {
				prefixes = append(prefixes, netip.PrefixFrom(*m.IPv4.Addr, 32))
				prefixes = append(prefixes, netip.PrefixFrom(*m.IPv6.Addr, 128))
			} else {
				addr, err := netip.ParseAddr(ip)
				if err == nil && m.HasIP(addr) {
					if addr.Is4() {
						prefixes = append(prefixes, netip.PrefixFrom(addr, 32))
					} else {
						prefixes = append(prefixes, netip.PrefixFrom(addr, 128))
					}
				}
			}
		}
		return prefixes
	}

	return translate(self), translate(other)
}

func (a ACLPolicy) translateDestinationAliasesToMachineIPs(aliases []string, m *Machine) ([]string, []string) {
	var self = &StringSet{}
	var other = &StringSet{}
	for _, alias := range aliases {
		ips := a.translateDestinationAliasToMachineIPs(alias, m)
		if alias == AutoGroupSelf {
			self.Add(ips...)
		} else {
			other.Add(ips...)
		}
	}
	return self.Items(), other.Items()
}

func (a ACLPolicy) translateDestinationAliasesToMachineNetPortRanges(aliases []string, m *Machine) ([]tailcfg.NetPortRange, []tailcfg.NetPortRange) {
	var self []tailcfg.NetPortRange
	var other []tailcfg.NetPortRange
	for _, alias := range aliases {
		ranges := a.translationDestinationAliasToMachineNetPortRanges(alias, m)
		if strings.HasPrefix(alias, AutoGroupSelf) {
			self = append(self, ranges...)
		} else {
			other = append(other, ranges...)
		}
	}
	return self, other
}

func (a ACLPolicy) translationDestinationAliasToMachineNetPortRanges(alias string, m *Machine) []tailcfg.NetPortRange {
	lastInd := strings.LastIndex(alias, ":")
	if lastInd == -1 {
		return nil
	}

	ports := alias[lastInd+1:]
	alias = alias[:lastInd]

	portRanges, err := a.parsePortRanges(ports)
	if err != nil {
		return nil
	}

	ips := a.translateDestinationAliasToMachineIPs(alias, m)
	if len(ips) == 0 {
		return nil
	}

	var netPortRanges []tailcfg.NetPortRange
	for _, d := range ips {
		for _, p := range portRanges {
			pr := tailcfg.NetPortRange{
				IP:    d,
				Ports: p,
			}
			netPortRanges = append(netPortRanges, pr)
		}
	}

	return netPortRanges
}

func (a ACLPolicy) translateDestinationAliasToMachineIPs(alias string, m *Machine) []string {
	f := func(alias string, m *Machine) []string {
		ip, err := netip.ParseAddr(alias)
		if err == nil && m.IsAllowedIP(ip) {
			return []string{ip.String()}
		}

		prefix, err := netip.ParsePrefix(alias)
		if err == nil && m.IsAllowedIPPrefix(prefix) {
			return []string{prefix.String()}
		}

		return make([]string, 0)
	}

	if alias == "*" {
		return []string{"*"}
	}

	return a.translateAliasToMachineIPs(alias, m, f)
}

func (a ACLPolicy) translateSourceAliasToMachineIPs(alias string, m *Machine, u *User) []string {
	f := func(alias string, m *Machine) []string {
		ip, err := netip.ParseAddr(alias)
		if err == nil && m.HasIP(ip) {
			return []string{ip.String()}
		}

		return make([]string, 0)
	}

	if u != nil && m.HasTags() {
		return []string{}
	}

	if u != nil && !m.HasUser(u.Name) {
		return []string{}
	}

	if alias == "*" {
		return append(m.IPs(), m.AllowedPrefixes()...)
	}

	if alias == AutoGroupDangerAll {
		return []string{"0.0.0.0/0", "::/0"}
	}

	return a.translateAliasToMachineIPs(alias, m, f)
}

func (a ACLPolicy) translateAliasToMachineIPs(alias string, m *Machine, f func(string, *Machine) []string) []string {
	if alias == AutoGroupMember || alias == AutoGroupMembers || alias == AutoGroupSelf {
		if !m.HasTags() {
			return m.IPs()
		} else {
			return []string{}
		}
	}

	if alias == AutoGroupTagged {
		if m.HasTags() {
			return m.IPs()
		} else {
			return []string{}
		}
	}

	if alias == AutoGroupInternet && m.IsExitNode() {
		return autogroupInternetRanges()
	}

	if strings.Contains(alias, "@") && !m.HasTags() && m.HasUser(alias) {
		return m.IPs()
	}

	if strings.HasPrefix(alias, "group:") && !m.HasTags() && a.isGroupMember(alias, m) {
		return m.IPs()
	}

	if strings.HasPrefix(alias, "tag:") && m.HasTag(alias) {
		return m.IPs()
	}

	if h, ok := a.Hosts[alias]; ok {
		alias = h
	}

	return f(alias, m)
}
