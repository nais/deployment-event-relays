package vera

import (
	"encoding/json"
	"fmt"

	"github.com/navikt/deployment-event-relays/pkg/deployment"
)

// Payload represents the JSON payload supported by the Vera API. All fields are required
type Payload struct {
	Environment      string `json:"environment"`
	Application      string `json:"application"`
	Version          string `json:"version"`
	Deployer         string `json:"deployedBy"`
	Environmentclass string `json:"environmentClass"`
}

// BuildVeraEvent collects data from a deployment event and creates a valid payload for POSTing to the vera api.
func BuildVeraEvent(event *deployment.Event) Payload {

	return Payload{
		Environment:      getEnvironment(event),
		Application:      event.GetApplication(),
		Version:          event.GetVersion(),
		Deployer:         getDeployer(event),
		Environmentclass: getEnvironmentClass(event),
	}
}

func getEnvironment(event *deployment.Event) string {
	if event.GetSkyaEnvironment() != "" {
		return event.GetSkyaEnvironment()
	}
	return fmt.Sprintf("%s:%s", event.GetCluster(), event.GetNamespace())
}

func getDeployer(event *deployment.Event) string {
	if len(event.GetDeployer().GetName()) > 0 {
		return fmt.Sprintf("%s (%s)", event.GetSource().String(), event.GetDeployer().GetName())
	} else if len(event.GetDeployer().GetIdent()) > 0 {
		return fmt.Sprintf("%s (%s)", event.GetSource().String(), event.GetDeployer().GetIdent())
	} else {
		return event.GetSource().String()
	}
}

func getEnvironmentClass(event *deployment.Event) string {
	if event.GetEnvironment() == deployment.Environment_production {
		return "p"
	}
	return "q"

}

// Marshal VeraPayload struct to JSON
func (payload Payload) Marshal() ([]byte, error) {
	var marshaledPayload []byte
	var err error

	marshaledPayload, err = json.Marshal(payload)

	if err != nil {
		return nil, err
	}

	return marshaledPayload, nil
}
