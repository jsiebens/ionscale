package eventlog

import (
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/jsiebens/ionscale/internal/domain"
	"math/big"
)

const (
	tailnetCreated          = "ionscale.tailnet.create"
	tailnetIamUpdated       = "ionscale.tailnet.iam.update"
	tailnetAclUpdated       = "ionscale.tailnet.acl.update"
	tailnetDNSConfigUpdated = "ionscale.tailnet.dns_config.update"
	nodeCreated             = "ionscale.node.create"
)

func TailnetCreated(tailnet *domain.Tailnet, actor ActorOpts) cloudevents.Event {
	data := &EventData[any]{
		Tailnet: &Target{ID: idToStr(tailnet.ID), Name: tailnet.Name},
		Target:  &Target{ID: idToStr(tailnet.ID), Name: tailnet.Name},
		Actor:   actor(),
	}

	event := cloudevents.NewEvent()
	event.SetType(tailnetCreated)
	_ = event.SetData(cloudevents.ApplicationJSON, data)

	return event
}

func TailnetIAMUpdated(tailnet *domain.Tailnet, old *domain.IAMPolicy, actor ActorOpts) cloudevents.Event {
	data := &EventData[*domain.IAMPolicy]{
		Tailnet: &Target{ID: idToStr(tailnet.ID), Name: tailnet.Name},
		Target:  &Target{ID: idToStr(tailnet.ID), Name: tailnet.Name},
		Actor:   actor(),
		Attr: &Attr[*domain.IAMPolicy]{
			New: &tailnet.IAMPolicy,
			Old: old,
		},
	}

	event := cloudevents.NewEvent()
	event.SetType(tailnetIamUpdated)
	_ = event.SetData(cloudevents.ApplicationJSON, data)

	return event
}

func TailnetACLUpdated(tailnet *domain.Tailnet, old *domain.ACLPolicy, actor ActorOpts) cloudevents.Event {
	data := &EventData[*domain.ACLPolicy]{
		Tailnet: &Target{ID: idToStr(tailnet.ID), Name: tailnet.Name},
		Target:  &Target{ID: idToStr(tailnet.ID), Name: tailnet.Name},
		Actor:   actor(),
		Attr: &Attr[*domain.ACLPolicy]{
			New: &tailnet.ACLPolicy,
			Old: old,
		},
	}

	event := cloudevents.NewEvent()
	event.SetType(tailnetAclUpdated)
	_ = event.SetData(cloudevents.ApplicationJSON, data)

	return event
}

func TailnetDNSConfigUpdated(tailnet *domain.Tailnet, old *domain.DNSConfig, actor ActorOpts) cloudevents.Event {
	data := &EventData[*domain.DNSConfig]{
		Tailnet: &Target{ID: idToStr(tailnet.ID), Name: tailnet.Name},
		Target:  &Target{ID: idToStr(tailnet.ID), Name: tailnet.Name},
		Actor:   actor(),
		Attr: &Attr[*domain.DNSConfig]{
			New: &tailnet.DNSConfig,
			Old: old,
		},
	}

	event := cloudevents.NewEvent()
	event.SetType(tailnetDNSConfigUpdated)
	_ = event.SetData(cloudevents.ApplicationJSON, data)

	return event
}

func MachineCreated(machine *domain.Machine, actor ActorOpts) cloudevents.Event {
	data := &EventData[any]{
		Tailnet: &Target{ID: idToStr(machine.Tailnet.ID), Name: machine.Tailnet.Name},
		Target:  &Target{ID: idToStr(machine.ID), Name: machine.CompleteName()},
		Actor:   actor(),
	}

	event := cloudevents.NewEvent()
	event.SetType(nodeCreated)
	_ = event.SetData(cloudevents.ApplicationJSON, data)

	return event
}

type ActorOpts func() Actor

func User(u *domain.User) ActorOpts {
	if u == nil {
		return SystemAdmin()
	}

	switch u.UserType {
	case domain.UserTypePerson:
		return func() Actor {
			return Actor{ID: idToStr(u.ID), Name: u.Name}
		}
	default:
		return SystemAdmin()
	}
}

func SystemAdmin() ActorOpts {
	return func() Actor {
		return Actor{ID: "", Name: "system admin"}
	}
}

type EventData[T any] struct {
	Tailnet *Target  `json:"tailnet,omitempty"`
	Target  *Target  `json:"target,omitempty"`
	Attr    *Attr[T] `json:"attr,omitempty"`
	Actor   Actor    `json:"actor"`
}

type Target struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Actor struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`
}

type Attr[T any] struct {
	New T `json:"new"`
	Old T `json:"old,omitempty"`
}

func idToStr(id uint64) string {
	return big.NewInt(int64(id)).Text(10)
}
