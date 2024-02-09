package eventlog

import (
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/jsiebens/ionscale/internal/domain"
	"math/big"
)

const (
	tailnetCreated = "ionscale.tailnet.created"
	tailnetDeleted = "ionscale.tailnet.deleted"
	nodeCreated    = "ionscale.node.created"
)

func TailnetCreated(tailnet *domain.Tailnet, actor *domain.User) cloudevents.Event {
	data := &EventData{
		Tailnet: &Target{ID: idToStr(tailnet.ID), Name: tailnet.Name},
		Target:  &Target{ID: idToStr(tailnet.ID), Name: tailnet.Name},
		Actor:   system,
	}

	if actor != nil {
		data.Actor = Actor{ID: idToStr(actor.ID), Name: actor.Name}
	}

	event := cloudevents.NewEvent()
	event.SetType(tailnetCreated)
	_ = event.SetData(cloudevents.ApplicationJSON, data)

	return event
}

func MachineCreated(machine *domain.Machine, actor *domain.User) cloudevents.Event {
	data := &EventData{
		Tailnet: &Target{ID: idToStr(machine.Tailnet.ID), Name: machine.Tailnet.Name},
		Target:  &Target{ID: idToStr(machine.ID), Name: machine.CompleteName(), Addresses: machine.IPs()},
		Actor:   UserToActor(actor),
	}

	event := cloudevents.NewEvent()
	event.SetType(nodeCreated)
	_ = event.SetData(cloudevents.ApplicationJSON, data)

	return event
}

func UserToActor(actor *domain.User) Actor {
	if actor == nil {
		return system
	}

	switch actor.UserType {
	case domain.UserTypePerson:
		return Actor{ID: idToStr(actor.ID), Name: actor.Name}
	default:
		return system
	}
}

type EventData struct {
	Tailnet *Target `json:"tailnet,omitempty"`
	Target  *Target `json:"target,omitempty"`
	Actor   Actor   `json:"actor"`
}

type Target struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Addresses []string `json:"addresses,omitempty"`
}

type Actor struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`
}

func idToStr(id uint64) string {
	return big.NewInt(int64(id)).Text(10)
}

var system = Actor{ID: "", Name: "ionscale system"}
