package vera

import (
	"encoding/json"
	"fmt"

	"github.com/navikt/deployment-event-relays/pkg/deployment"
)

// VeraPayload represents the JSON payload supported by the Vera API. All fields are required
type VeraPayload struct {
	Environment      string `json:"environment"`
	Application      string `json:"application"`
	Version          string `json:"version"`
	Deployer         string `json:"deployer"`
	Environmentclass string `json:"environmentclass"`
}

// BuildVeraEvent collects data from a deployment event and creates a valid payload for POSTing to the vera api.
func BuildVeraEvent(event *deployment.Event) VeraPayload {
	var deployer string

	if len(event.GetDeployer().GetName()) > 0 {
		deployer = fmt.Sprintf("%s (%s)", event.GetSource().String(), event.GetDeployer().GetName())
	} else if len(event.GetDeployer().GetIdent()) > 0 {
		deployer = fmt.Sprintf("%s (%s)", event.GetSource().String(), event.GetDeployer().GetIdent())
	} else {
		deployer = event.GetSource().String()
	}

	return VeraPayload{
		Environment:      getEnvironment(event),
		Application:      event.GetApplication(),
		Version:          event.GetVersion(),
		Deployer:         deployer,
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
