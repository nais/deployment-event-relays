package influx

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/navikt/deployment-event-relays/pkg/deployment"
	log "github.com/sirupsen/logrus"
)

type Relay struct {
	URL      string
	Username string
	Password string
}

func (r *Relay) Process(event *deployment.Event) (retry bool, err error) {
	line := NewLine(event)
	payload, err := line.Marshal()
	if err != nil {
		return false, fmt.Errorf("marshal InfluxDB payload: %s", err)
	}

	body := bytes.NewReader(payload)
	request, err := http.NewRequest("POST", r.URL, body)
	if err != nil {
		return true, fmt.Errorf("unable to create new HTTP request object: %s", err)
	}

	if len(r.Username) > 0 && len(r.Password) > 0 {
		request.SetBasicAuth(r.Username, r.Password)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return true, fmt.Errorf("post to InfluxDB: %s", err)
	}
	bodyLoad, _ := ioutil.ReadAll(response.Body)

	defer response.Body.Close()

	if response.StatusCode > 299 {
		raw, _ := line.Marshal()
		log.WithFields(log.Fields{
			"raw_payload":    string(raw),
			"error_response": bodyLoad,
			"correlation_id": event.GetCorrelationID(),
		}).Debugf("NOTE: raw payload and output included here for %s", event.GetCorrelationID())

		return true, fmt.Errorf("POST %s: %s", r.URL, response.Status)
	}

	return false, nil
}
