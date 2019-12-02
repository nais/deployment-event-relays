package nora_test

import (
	"testing"

	"github.com/navikt/deployment-event-relays/pkg/deployment"
	"github.com/navikt/deployment-event-relays/pkg/nora"
	"github.com/stretchr/testify/assert"
)

type eventNoraTest struct {
	event deployment.Event
	data  nora.Payload
	err   error
}

var eventNoraTests = []eventNoraTest{
	{
		data: nora.Payload{
			Cluster: "prod-fss",
			Kilde:   "naiserator",
			Name:    "nora",
			Team:    "myteam",
			Zone:    "fss",
		},
		event: deployment.Event{
			Application: "nora",
			Cluster:     "prod-fss",
			Source:      deployment.System_naiserator,
			Team:        "myteam",
		},
	},
	{
		data: nora.Payload{
			Cluster: "foobar-gcp",
			Kilde:   "naisd",
			Name:    "foo",
			Team:    "bar",
			Zone:    "gcp",
		},
		event: deployment.Event{
			Application: "foo",
			Cluster:     "foobar-gcp",
			Source:      deployment.System_naisd,
			Team:        "bar",
		},
	},
	{
		data: nora.Payload{
			Cluster: "",
			Kilde:   "aura",
			Name:    "foo",
			Team:    "",
			Zone:    "",
		},
		event: deployment.Event{
			Application: "foo",
			Source:      deployment.System_aura,
		},
	},
}

func TestNoraPayload(t *testing.T) {
	for _, test := range eventNoraTests {
		noraPayload := nora.BuildEvent(&test.event)
		assert.Equal(t, test.data, noraPayload)
	}
}
