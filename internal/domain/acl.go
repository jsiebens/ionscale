package domain

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"net/netip"
	"sort"
	"strconv"
	"strings"
	"tailscale.com/tailcfg"
)

const (
	AutoGroupSelf     = "autogroup:self"
	AutoGroupMember   = "autogroup:member"
	AutoGroupMembers  = "autogroup:members"
	AutoGroupTagged   = "autogroup:tagged"
	AutoGroupInternet = "autogroup:internet"
)

type AutoApprovers struct {
	Routes   map[string][]string `json:"routes,omitempty"`
	ExitNode []string            `json:"exitNode,omitempty"`
}

type ACLPolicy struct {
	Groups        map[string][]string `json:"groups,omitempty"`
	Hosts         map[string]string   `json:"hosts,omitempty"`
	ACLs          []ACL               `json:"acls,omitempty"`
	TagOwners     map[string][]string `json:"tagowners,omitempty"`
	AutoApprovers *AutoApprovers      `json:"autoApprovers,omitempty"`
	SSHRules      []SSHRule           `json:"ssh,omitempty"`
	NodeAttrs     []NodeAttr          `json:"nodeAttrs,omitempty"`
}

type ACL struct {
	Action string   `json:"action"`
	Src    []string `json:"src"`
	Dst    []string `json:"dst"`
}

type SSHRule struct {
	Action      string   `json:"action"`
	Src         []string `json:"src"`
	Dst         []string `json:"dst"`
	Users       []string `json:"users"`
	CheckPeriod string   `json:"checkPeriod,omitempty"`
}

type NodeAttr struct {
	Target []string `json:"target"`
	Attr   []string `json:"attr"`
}

func DefaultACLPolicy() ACLPolicy {
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

func (a ACLPolicy) FindAutoApprovedIPs(routableIPs []netip.Prefix, tags []string, u *User) []netip.Prefix {
	if a.AutoApprovers == nil || len(routableIPs) == 0 {
		return nil
	}

	matches := func(values []string) bool {
		for _, alias := range values {
			if alias == u.Name {
				return true
			}

			group, ok := a.Groups[alias]
			if ok {
				for _, g := range group {
					if g == u.Name {
						return true
					}
				}
			}

			if strings.HasPrefix(alias, "tag:") {
				for _, tag := range tags {
					if alias == tag {
						return true
					}
				}
			}
		}
		return false
	}

	isAutoApproved := func(candidate netip.Prefix, approvedIPs []netip.Prefix) bool {
		for _, approvedIP := range approvedIPs {
			if candidate.Bits() >= approvedIP.Bits() && approvedIP.Contains(candidate.Masked().Addr()) {
				return true
			}
		}
		return false
	}

	autoApprovedIPs := []netip.Prefix{}
	for route, autoApprovers := range a.AutoApprovers.Routes {
		candidate, err := netip.ParsePrefix(route)
		if err != nil {
			return nil
		}

		if matches(autoApprovers) {
			autoApprovedIPs = append(autoApprovedIPs, candidate)
		}
	}

	result := []netip.Prefix{}
	for _, c := range routableIPs {
		if c.Bits() == 0 && matches(a.AutoApprovers.ExitNode) {
			result = append(result, c)
		}
		if isAutoApproved(c, autoApprovedIPs) {
			result = append(result, c)
		}
	}

	return result
}

func (a ACLPolicy) IsTagOwner(tags []string, p *User) bool {
	for _, t := range tags {
		if a.isTagOwner(t, p) {
			return true
		}
	}
	return false
}

func (a ACLPolicy) CheckTagOwners(tags []string, p *User) error {
	var result *multierror.Error
	for _, t := range tags {
		if ok := a.isTagOwner(t, p); !ok {
			result = multierror.Append(result, fmt.Errorf("tag [%s] is invalid or not permitted", t))
		}
	}
	return result.ErrorOrNil()
}

func (a ACLPolicy) isTagOwner(tag string, p *User) bool {
	if p.UserType == UserTypeService {
		return true
	}
	if tagOwners, ok := a.TagOwners[tag]; ok {
		return a.validateTagOwners(tagOwners, p)
	}
	return false
}

func (a ACLPolicy) validateTagOwners(tagOwners []string, p *User) bool {
	for _, alias := range tagOwners {
		if strings.HasPrefix(alias, "group:") {
			if group, ok := a.Groups[alias]; ok {
				for _, groupMember := range group {
					if groupMember == p.Name {
						return true
					}
				}
			}
		} else {
			if alias == p.Name {
				return true
			}
		}
	}
	return false
}

func (a ACLPolicy) IsValidPeer(src *Machine, dest *Machine) bool {
	if !src.HasTags() && !dest.HasTags() && dest.HasUser(src.User.Name) {
		return true
	}

	for _, acl := range a.ACLs {
		selfDestPorts, allDestPorts := a.expandMachineToDstPorts(dest, acl.Dst)
		if len(selfDestPorts) != 0 {
			for _, alias := range acl.Src {
				if len(a.expandMachineAlias(src, alias, true, &dest.User)) != 0 {
					return true
				}
			}
		}
		if len(allDestPorts) != 0 {
			for _, alias := range acl.Src {
				if len(a.expandMachineAlias(src, alias, true, nil)) != 0 {
					return true
				}
			}
		}
	}

	return false
}

func (a ACLPolicy) NodeCapabilities(m *Machine) []tailcfg.NodeCapability {
	var result = &StringSet{}

	matches := func(targets []string) bool {
		for _, alias := range targets {
			if alias == "*" {
				return true
			}

			if strings.Contains(alias, "@") && !m.HasTags() && m.HasUser(alias) {
				return true
			}

			if strings.HasPrefix(alias, "tag:") && m.HasTag(alias) {
				return true
			}

			if strings.HasPrefix(alias, "group:") && !m.HasTags() {
				for _, u := range a.Groups[alias] {
					if m.HasUser(u) {
						return true
					}
				}
			}
		}

		return false
	}

	for _, nodeAddr := range a.NodeAttrs {
		if matches(nodeAddr.Target) {
			result.Add(nodeAddr.Attr...)
		}
	}

	items := result.Items()
	caps := make([]tailcfg.NodeCapability, len(items))
	for i, c := range items {
		caps[i] = tailcfg.NodeCapability(c)
	}

	return caps
}

func (a ACLPolicy) BuildFilterRules(srcs []Machine, dst *Machine) []tailcfg.FilterRule {
	var rules []tailcfg.FilterRule

	transform := func(src []string, destPorts []tailcfg.NetPortRange, u *User) tailcfg.FilterRule {
		var allSrcIPsSet = &StringSet{}
		for _, alias := range src {
			for _, src := range srcs {
				srcIPs := a.expandMachineAlias(&src, alias, true, u)
				allSrcIPsSet.Add(srcIPs...)
			}
		}

		allSrcIPs := allSrcIPsSet.Items()

		if len(allSrcIPs) == 0 {
			allSrcIPs = nil
		}

		return tailcfg.FilterRule{
			SrcIPs:   allSrcIPs,
			DstPorts: destPorts,
		}
	}

	for _, acl := range a.ACLs {
		selfDestPorts, allDestPorts := a.expandMachineToDstPorts(dst, acl.Dst)
		if len(selfDestPorts) != 0 {
			rules = append(rules, transform(acl.Src, selfDestPorts, &dst.User))
		}
		if len(allDestPorts) != 0 {
			rules = append(rules, transform(acl.Src, allDestPorts, nil))
		}
	}

	if len(rules) == 0 {
		return []tailcfg.FilterRule{{}}
	}

	return rules
}

func (a ACLPolicy) expandMachineToDstPorts(m *Machine, ports []string) ([]tailcfg.NetPortRange, []tailcfg.NetPortRange) {
	selfDestRanges := []tailcfg.NetPortRange{}
	otherDestRanges := []tailcfg.NetPortRange{}
	for _, d := range ports {
		self, ranges := a.expandMachineDestToNetPortRanges(m, d)
		if self {
			selfDestRanges = append(selfDestRanges, ranges...)
		} else {
			otherDestRanges = append(otherDestRanges, ranges...)
		}
	}
	return selfDestRanges, otherDestRanges
}

func (a ACLPolicy) expandMachineDestToNetPortRanges(m *Machine, dest string) (bool, []tailcfg.NetPortRange) {
	lastInd := strings.LastIndex(dest, ":")
	if lastInd == -1 {
		return false, nil
	}

	alias := dest[:lastInd]
	portRange := dest[lastInd+1:]

	ports, err := a.expandValuePortToPortRange(portRange)
	if err != nil {
		return false, nil
	}

	ips := a.expandMachineAlias(m, alias, false, nil)
	if len(ips) == 0 {
		return false, nil
	}

	var netPortRanges []tailcfg.NetPortRange
	for _, d := range ips {
		for _, p := range ports {
			pr := tailcfg.NetPortRange{
				IP:    d,
				Ports: p,
			}
			netPortRanges = append(netPortRanges, pr)
		}
	}

	return alias == AutoGroupSelf, netPortRanges
}

func (a ACLPolicy) expandMachineAlias(m *Machine, alias string, src bool, u *User) []string {
	if u != nil && m.HasTags() {
		return []string{}
	}

	if u != nil && !m.HasUser(u.Name) {
		return []string{}
	}

	if alias == "*" && u != nil {
		return m.IPs()
	}

	if alias == "*" {
		return []string{"*"}
	}

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

	if strings.HasPrefix(alias, "group:") && !m.HasTags() {
		users, ok := a.Groups[alias]

		if !ok {
			return []string{}
		}

		for _, u := range users {
			if m.HasUser(u) {
				return m.IPs()
			}
		}

		return []string{}
	}

	if strings.HasPrefix(alias, "tag:") && m.HasTag(alias) {
		return m.IPs()
	}

	if h, ok := a.Hosts[alias]; ok {
		alias = h
	}

	if src {
		ip, err := netip.ParseAddr(alias)
		if err == nil && m.HasIP(ip) {
			return []string{ip.String()}
		}
	} else {
		ip, err := netip.ParseAddr(alias)
		if err == nil && m.IsAllowedIP(ip) {
			return []string{ip.String()}
		}

		prefix, err := netip.ParsePrefix(alias)
		if err == nil && m.IsAllowedIPPrefix(prefix) {
			return []string{prefix.String()}
		}
	}

	return []string{}
}

func (a ACLPolicy) expandValuePortToPortRange(s string) ([]tailcfg.PortRange, error) {
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

func (a ACLPolicy) isGroupMember(group string, m *Machine) bool {
	if m.HasTags() {
		return false
	}

	users, ok := a.Groups[group]
	if !ok {
		return false
	}

	for _, u := range users {
		if m.HasUser(u) {
			return true
		}
	}

	return false
}

func (i *ACLPolicy) Scan(destination interface{}) error {
	switch value := destination.(type) {
	case []byte:
		return json.Unmarshal(value, i)
	default:
		return fmt.Errorf("unexpected data type %T", destination)
	}
}

func (i ACLPolicy) Value() (driver.Value, error) {
	bytes, err := json.Marshal(i)
	return bytes, err
}

// GormDataType gorm common data type
func (ACLPolicy) GormDataType() string {
	return "json"
}

// GormDBDataType gorm db data type
func (ACLPolicy) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case "sqlite":
		return "JSON"
	}
	return ""
}

type StringSet struct {
	items map[string]bool
}

func (s *StringSet) Add(t ...string) *StringSet {
	if s.items == nil {
		s.items = make(map[string]bool)
	}

	for _, v := range t {
		s.items[v] = true
	}

	return s
}

func (s *StringSet) Items() []string {
	items := []string{}
	for i := range s.items {
		items = append(items, i)
	}
	sort.Strings(items)
	return items
}

func autogroupInternetRanges() []string {
	return []string{
		"0.0.0.0/5",
		"8.0.0.0/7",
		"11.0.0.0/8",
		"12.0.0.0/6",
		"16.0.0.0/4",
		"32.0.0.0/3",
		"64.0.0.0/3",
		"96.0.0.0/6",
		"100.0.0.0/10",
		"100.128.0.0/9",
		"101.0.0.0/8",
		"102.0.0.0/7",
		"104.0.0.0/5",
		"112.0.0.0/4",
		"128.0.0.0/3",
		"160.0.0.0/5",
		"168.0.0.0/8",
		"169.0.0.0/9",
		"169.128.0.0/10",
		"169.192.0.0/11",
		"169.224.0.0/12",
		"169.240.0.0/13",
		"169.248.0.0/14",
		"169.252.0.0/15",
		"169.255.0.0/16",
		"170.0.0.0/7",
		"172.0.0.0/12",
		"172.32.0.0/11",
		"172.64.0.0/10",
		"172.128.0.0/9",
		"173.0.0.0/8",
		"174.0.0.0/7",
		"176.0.0.0/4",
		"192.0.0.0/9",
		"192.128.0.0/11",
		"192.160.0.0/13",
		"192.169.0.0/16",
		"192.170.0.0/15",
		"192.172.0.0/14",
		"192.176.0.0/12",
		"192.192.0.0/10",
		"193.0.0.0/8",
		"194.0.0.0/7",
		"196.0.0.0/6",
		"200.0.0.0/5",
		"208.0.0.0/4",
		"224.0.0.0/3",
		"2000::/3",
	}
}
