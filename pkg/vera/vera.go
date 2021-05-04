package vera

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/navikt/deployment-event-relays/pkg/deployment"
	log "github.com/sirupsen/logrus"
)

// Payload represents the JSON payload supported by the Vera API. All fields are required
type Payload struct {
	Environment      string `json:"environment"`
	Application      string `json:"application"`
	Version          string `json:"version"`
	Deployer         string `json:"deployedBy"`
	Environmentclass string `json:"environmentClass"`
}

type Relay struct {
	URL string
}

func (r *Relay) Process(event *deployment.Event) (retry bool, err error) {
	if event.GetRolloutStatus() != deployment.RolloutStatus_complete {
		return false, fmt.Errorf("discarding message because rollout status is != complete")
	}

	veraEvent := BuildVeraEvent(event)
	payload, err := veraEvent.Marshal()
	log.Infof("Posting payload to Vera: %s", payload)
	if err != nil {
		return false, fmt.Errorf("marshal Vera payload: %s", err)
	}

	body := bytes.NewReader(payload)
	request, err := http.NewRequest("POST", r.URL, body)
	if err != nil {
		return true, fmt.Errorf("unable to create new HTTP request object: %s", err)
	}
	request.Header.Set("Content-Type", "application/json")

	// Create a callback function wrapping recoverable network errors.
	// This callback will be run as many times as necessary in order to
	// ensure the data is fully written to Vera.
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return true, fmt.Errorf("post to Vera: %s", err)
	}

	if response.StatusCode > 299 {
		return true, fmt.Errorf("POST %s: %s", r.URL, response.Status)
	}

	return false, nil
}

// BuildVeraEvent collects data from a deployment event and creates a valid payload for POSTing to the vera api.
func BuildVeraEvent(event *deployment.Event) Payload {
	return Payload{
		Environment:      getEnvironment(event),
		Application:      event.GetApplication(),
		Version:          getVersion(event),
		Deployer:         getDeployer(event),
		Environmentclass: getEnvironmentClass(event),
	}
}

func getEnvironment(event *deployment.Event) string {
	if event.GetSkyaEnvironment() != "" {
		return event.GetSkyaEnvironment()
	} else if event.GetEnvironment() == deployment.Environment_production && event.GetSource() == deployment.System_naiserator {
		return "p"
	}

	return fmt.Sprintf("%s:%s", event.GetNamespace(), event.GetCluster())
}

func getVersion(event *deployment.Event) string {
	if len(event.GetVersion()) > 0 {
		return event.GetVersion()
	} else {
		return "unknown"
	}
}

func getDeployer(event *deployment.Event) string {
	source := event.GetSource().String()

	if len(event.GetDeployer().GetName()) > 0 {
		return fmt.Sprintf("%s (%s)", source, event.GetDeployer().GetName())
	} else if len(event.GetDeployer().GetIdent()) > 0 {
		return fmt.Sprintf("%s (%s)", source, event.GetDeployer().GetIdent())
	} else if len(event.GetTeam()) > 0 {
		return fmt.Sprintf("%s (%s)", source, event.GetTeam())
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
