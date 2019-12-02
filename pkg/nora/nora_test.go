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

var eventVeraTests = []eventNoraTest{
	{
		data: nora.Payload{
			Name:    "nora",
			Team:    "myteam",
			Cluster: "prod-fss",
			Zone:    "fss",
			Kilde:   "naiserator",
		},
		event: deployment.Event{
			Cluster:     "prod-fss",
			Namespace:   "default",
			Environment: deployment.Environment_development,
			Application: "nora",
			Team:"myteam",
			Version:     "1.2.3",
			Source:      deployment.System_naiserator,
		},
	},
	{
		data: nora.Payload{
			Environment:      "default:dev-fss",
			Application:      "app",
			Version:          "1.2.3",
			Deployer:         "naiserator (super-team)",
			Environmentclass: "q",
		},
		event: deployment.Event{
			Cluster:     "dev-fss",
			Namespace:   "default",
			Environment: deployment.Environment_development,
			Application: "app",
			Version:     "1.2.3",
			Team:        "super-team",
			Source:      deployment.System_naiserator,
		},
	},
	{
		data: nora.Payload{
			Environment:      "p",
			Application:      "app",
			Version:          "1.2.3",
			Deployer:         "naisd (ident)",
			Environmentclass: "p",
		},
		event: deployment.Event{
			SkyaEnvironment: "p",
			Environment:     deployment.Environment_production,
			Application:     "app",
			Version:         "1.2.3",
			Source:          deployment.System_naisd,
			Deployer: &deployment.Actor{
				Ident: "ident",
			}},
	},
	{
		data: nora.Payload{
			Environment:      "env",
			Application:      "app",
			Version:          "1.2.3",
			Deployer:         "aura (name)",
			Environmentclass: "q",
		},
		event: deployment.Event{
			SkyaEnvironment: "env",
			Application:     "app",
			Version:         "1.2.3",
			Environment:     deployment.Environment_development,
			Source:          deployment.System_aura,
			Deployer: &deployment.Actor{
				Name:  "name",
				Ident: "ident",
			}},
	},
	{
		data: nora.Payload{
			Environment:      "env",
			Application:      "app",
			Version:          "unknown",
			Deployer:         "aura (name)",
			Environmentclass: "q",
		},
		event: deployment.Event{
			SkyaEnvironment: "env",
			Application:     "app",
			Environment:     deployment.Environment_development,
			Source:          deployment.System_aura,
			Deployer: &deployment.Actor{
				Name:  "name",
				Ident: "ident",
			}},
	},
	{
		data: nora.Payload{
			Environment:      "p",
			Application:      "app",
			Version:          "1.2.3",
			Deployer:         "naiserator",
			Environmentclass: "p",
		},
		event: deployment.Event{
			Application: "app",
			Environment: deployment.Environment_production,
			Source:      deployment.System_naiserator,
			Version:     "1.2.3",
		},
	},
}

func TestVeraPayload(t *testing.T) {
	for _, test := range eventVeraTests {
		noraPayload := nora.BuildVeraEvent(&test.event)
		assert.Equal(t, test.data, noraPayload)
	}
}
