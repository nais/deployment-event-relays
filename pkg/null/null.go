package null

import (
	"github.com/navikt/deployment-event-relays/pkg/deployment"
)

type Relay struct {
	URL string
}

func (r *Relay) Process(event *deployment.Event) (retry bool, err error) {
	return false, nil
}
