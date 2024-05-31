package ionscale

import (
	"encoding/json"
	"tailscale.com/tailcfg"
)

type IAMPolicy struct {
	Subs    []string          `json:"subs,omitempty" hujson:"Subs,omitempty"`
	Emails  []string          `json:"emails,omitempty" hujson:"Emails,omitempty"`
	Filters []string          `json:"filters,omitempty" hujson:"Filters,omitempty"`
	Roles   map[string]string `json:"roles,omitempty" hujson:"Roles,omitempty"`
}

func (a IAMPolicy) Marshal() string {
	indent, _ := json.MarshalIndent(&a, "", "  ")
	return string(indent)
}

type ACLPolicy struct {
	Groups        map[string][]string `json:"groups,omitempty" hujson:"Groups,omitempty"`
	Hosts         map[string]string   `json:"hosts,omitempty" hujson:"Hosts,omitempty"`
	ACLs          []ACLEntry          `json:"acls,omitempty" hujson:"ACLs,omitempty"`
	TagOwners     map[string][]string `json:"tagOwners,omitempty" hujson:"TagOwners,omitempty"`
	AutoApprovers *ACLAutoApprovers   `json:"autoApprovers,omitempty" hujson:"AutoApprovers,omitempty"`
	SSH           []ACLSSH            `json:"ssh,omitempty" hujson:"SSH,omitempty"`
	NodeAttrs     []ACLNodeAttrGrant  `json:"nodeAttrs,omitempty" hujson:"NodeAttrs,omitempty"`
	Grants        []ACLGrant          `json:"grants,omitempty" hujson:"Grants,omitempty"`
}

func (a ACLPolicy) Marshal() string {
	indent, _ := json.MarshalIndent(&a, "", "  ")
	return string(indent)
}

type ACLAutoApprovers struct {
	Routes   map[string][]string `json:"routes,omitempty" hujson:"Routes,omitempty"`
	ExitNode []string            `json:"exitNode,omitempty" hujson:"ExitNode,omitempty"`
}

type ACLEntry struct {
	Action      string   `json:"action,omitempty" hujson:"Action,omitempty"`
	Protocol    string   `json:"proto,omitempty" hujson:"Proto,omitempty"`
	Source      []string `json:"src,omitempty" hujson:"Src,omitempty"`
	Destination []string `json:"dst,omitempty" hujson:"Dst,omitempty"`
}

type ACLSSH struct {
	Action          string   `json:"action,omitempty" hujson:"Action,omitempty"`
	Source          []string `json:"src,omitempty" hujson:"Src,omitempty"`
	Destination     []string `json:"dst,omitempty" hujson:"Dst,omitempty"`
	Users           []string `json:"users,omitempty" hujson:"Users,omitempty"`
	CheckPeriod     string   `json:"checkPeriod,omitempty" hujson:"CheckPeriod,omitempty"`
	Recorder        []string `json:"recorder,omitempty" hujson:"Recorder,omitempty"`
	EnforceRecorder bool     `json:"enforceRecorder,omitempty" hujson:"EnforceRecorder,omitempty"`
}

type ACLNodeAttrGrant struct {
	Target []string `json:"target,omitempty" hujson:"Target,omitempty"`
	Attr   []string `json:"attr,omitempty" hujson:"Attr,omitempty"`
}

type ACLGrant struct {
	Source      []string                 `json:"src,omitempty" hujson:"Src,omitempty"`
	Destination []string                 `json:"dst,omitempty" hujson:"Dst,omitempty"`
	IP          []tailcfg.ProtoPortRange `json:"ip,omitempty" hujson:"Ip,omitempty"`
	App         tailcfg.PeerCapMap       `json:"app,omitempty" hujson:"App,omitempty"`
}
