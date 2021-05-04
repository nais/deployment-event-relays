package nora

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/navikt/deployment-event-relays/pkg/deployment"
	log "github.com/sirupsen/logrus"
)

var (
	ErrNotProduction = errors.New("event does not belong to production")
)

// Payload represents the JSON payload supported by the nora API. All fields are required
type Payload struct {
	Name    string `json:"name"`
	Team    string `json:"team"`
	Cluster string `json:"cluster"`
	Zone    string `json:"zone"`
	Kilde   string `json:"kilde"`
}

type Relay struct {
	URL string
}

func (r *Relay) Process(event *deployment.Event) (retry bool, err error) {
	if event.GetEnvironment() != deployment.Environment_production {
		return false, ErrNotProduction
	}

	noraEvent := BuildEvent(event)
	payload, err := noraEvent.Marshal()
	log.Infof("Posting payload to Nora: %s", payload)
	if err != nil {
		return false, fmt.Errorf("marshal Nora payload: %s", err)
	}

	body := bytes.NewReader(payload)
	request, err := http.NewRequest("POST", r.URL, body)
	if err != nil {
		return true, fmt.Errorf("create new HTTP request object: %s", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return true, fmt.Errorf("post to Nora: %s", err)
	}

	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK: // Application created
		return false, nil
	case http.StatusCreated: // Application created
		return false, nil
	case http.StatusForbidden: // Writing null team name to entry with team registered
		return false, nil
	case http.StatusUnprocessableEntity: // Application is already registered
		return false, nil
	default:
		return true, fmt.Errorf("POST %s: %s", r.URL, response.Status)
	}
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
