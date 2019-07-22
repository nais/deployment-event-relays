package influx

import (
	"github.com/navikt/deployment-event-relays/pkg/deployment"
)

func NewLine(event *deployment.Event) Line {
	return Line{
		Measurement: "deployment",
		Tags: TagField{
			"application":    event.GetApplication(),
			"cluster":        event.GetCluster(),
			"environment":    event.GetEnvironment().String(),
			"team":           event.GetTeam(),
			"platform":       event.GetPlatform().GetType().String(),
			"rollout_status": event.GetRolloutStatus().String(),
		},
		Fields: TagField{
			"version": event.GetVersion(),
		},
		Timestamp: event.GetTimestampAsTime(),
	}
}

