package domain

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"net/netip"
	"slices"
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
	Grants        []Grant             `json:"grants,omitempty"`
}

type ACL struct {
	Action string   `json:"action"`
	Proto  string   `json:"proto"`
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

type Grant struct {
	Src []string                 `json:"src"`
	Dst []string                 `json:"dst"`
	IP  []tailcfg.ProtoPortRange `json:"ip"`
	App tailcfg.PeerCapMap       `json:"app"`
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

	var autoApprovedIPs []netip.Prefix
	for route, autoApprovers := range a.AutoApprovers.Routes {
		candidate, err := netip.ParsePrefix(route)
		if err != nil {
			return nil
		}

		if matches(autoApprovers) {
			autoApprovedIPs = append(autoApprovedIPs, candidate)
		}
	}

	var result []netip.Prefix
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
		for _, alias := range tagOwners {
			if strings.HasPrefix(alias, "group:") {
				if group, ok := a.Groups[alias]; ok {
					return slices.Contains(group, p.Name)
				}
			} else {
				if alias == p.Name {
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

func (a ACLPolicy) parsePortRanges(s string) ([]tailcfg.PortRange, error) {
	if s == "*" {
		return []tailcfg.PortRange{tailcfg.PortRangeAny}, nil
	}

	var ports []tailcfg.PortRange
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

const (
	protocolICMP     = 1   // Internet Control Message
	protocolIGMP     = 2   // Internet Group Management
	protocolIPv4     = 4   // IPv4 encapsulation
	protocolTCP      = 6   // Transmission Control
	protocolEGP      = 8   // Exterior Gateway Protocol
	protocolIGP      = 9   // any private interior gateway (used by Cisco for their IGRP)
	protocolUDP      = 17  // User Datagram
	protocolGRE      = 47  // Generic Routing Encapsulation
	protocolESP      = 50  // Encap Security Payload
	protocolAH       = 51  // Authentication Header
	protocolIPv6ICMP = 58  // ICMP for IPv6
	protocolSCTP     = 132 // Stream Control Transmission Protocol
)

func parseProtocol(protocol string) []int {
	switch protocol {
	case "":
		return nil
	case "igmp":
		return []int{protocolIGMP}
	case "ipv4", "ip-in-ip":
		return []int{protocolIPv4}
	case "tcp":
		return []int{protocolTCP}
	case "egp":
		return []int{protocolEGP}
	case "igp":
		return []int{protocolIGP}
	case "udp":
		return []int{protocolUDP}
	case "gre":
		return []int{protocolGRE}
	case "esp":
		return []int{protocolESP}
	case "ah":
		return []int{protocolAH}
	case "sctp":
		return []int{protocolSCTP}
	case "icmp":
		return []int{protocolICMP, protocolIPv6ICMP}

	default:
		n, err := strconv.Atoi(protocol)
		if err != nil {
			return nil
		}
		return []int{n}
	}
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

func (s *StringSet) Empty() bool {
	return len(s.items) == 0
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
