package vera

import (
	"encoding/json"
	"fmt"

	"github.com/navikt/deployment-event-relays/pkg/deployment"
)

// VeraPayload represents the JSON payload supported by the Vera API. All fields are required
type VeraPayload struct {
	Environment      string
	Application      string
	Version          string
	Deployer         string
	Environmentclass string
}

// BuildVeraEvent collects data from a deployment event and creates a valid payload for POSTing to the vera api.
func BuildVeraEvent(event *deployment.Event) VeraPayload {

	return VeraPayload{
		Environment:      getEnvironment(event),
		Application:      event.GetApplication(),
		Version:          event.GetVersion(),
		Deployer:         fmt.Sprintf("%s ( %s %s )", event.GetSource().String(), event.GetDeployer().GetName(), event.GetDeployer().GetIdent()),
		Environmentclass: event.GetEnvironment().String(),
	}
}

func getEnvironment(event *deployment.Event) string {
	if event.GetSkyaEnvironment() != "" {
		return event.GetSkyaEnvironment()
	}
	return fmt.Sprintf("%s (%s)", event.GetCluster(), event.GetNamespace())
}

// Marshal VeraPayload struct to JSON
func (payload VeraPayload) Marshal() ([]byte, error) {
	var marshaledPayload []byte
	var err error

	marshaledPayload, err = json.Marshal(payload)

	if err != nil {
		return nil, err
	}

	return marshaledPayload, nil
}
