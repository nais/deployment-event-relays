package nora

import (
	"encoding/json"
	"strings"

	"github.com/navikt/deployment-event-relays/pkg/deployment"
)

// Payload represents the JSON payload supported by the nora API. All fields are required
type Payload struct {
	Name    string `json:"name"`
	Team    string `json:"team"`
	Cluster string `json:"cluster"`
	Zone    string `json:"zone"`
	Kilde   string `json:"kilde"`
}

// BuildEvent collects data from a deployment event and creates a valid payload for POSTing to the nora api.
func BuildEvent(event *deployment.Event) Payload {
	return Payload{
		Name:    event.GetApplication(),
		Team:    event.GetTeam(),
		Cluster: event.GetCluster(),
		Zone:    getZone(event),
		Kilde:   event.GetSource().String(),
	}
}

func getZone(event *deployment.Event) string {
	tokens := strings.Split(event.GetCluster(), "-")
	// expect "prod-sbs", "dev-gcp", etc.
	if len(tokens) != 2 {
		return ""
	}
	return tokens[1]
}

// Marshal Payload struct to JSON
func (payload Payload) Marshal() ([]byte, error) {
	var marshaledPayload []byte
	var err error

	marshaledPayload, err = json.Marshal(payload)

	if err != nil {
		return nil, err
	}

	return marshaledPayload, nil
}
