package vera_test

import (
	"testing"

	"github.com/navikt/deployment-event-relays/pkg/deployment"
	"github.com/navikt/deployment-event-relays/pkg/vera"
	"github.com/stretchr/testify/assert"
)

type eventVeraTest struct {
	event deployment.Event
	data  vera.VeraPayload
	err   error
}

var eventVeraTests = []eventVeraTest{
	{
		data: vera.VeraPayload{
			Environment:      "dev-fss (default)",
			Application:      "app",
			Version:          "1.2.3",
			Deployer:         "naiserator",
			Environmentclass: "development",
		},
		event: deployment.Event{
			Cluster:     "dev-fss",
			Namespace:   "default",
			Environment: deployment.Environment_development,
			Application: "app",
			Version:     "1.2.3",
			Source:      deployment.System_naiserator,
		},
	},
	{
		data: vera.VeraPayload{
			Environment:      "p",
			Application:      "app",
			Version:          "1.2.3",
			Deployer:         "naisd (ident)",
			Environmentclass: "production",
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
		data: vera.VeraPayload{
			Environment:      "env",
			Application:      "app",
			Version:          "1.2.3",
			Deployer:         "aura (name)",
			Environmentclass: "development",
		},
		event: deployment.Event{
			SkyaEnvironment: "env",
			Application:     "app",
			Version:         "1.2.3",
			Environment:     deployment.Environment_development,
			Source:          deployment.System_aura,
			Deployer: &deployment.Actor{
				Name: "name",
				Ident: "ident",
			}},
	},
}

func TestEventLineData(t *testing.T) {
	for _, test := range eventVeraTests {
		veraPayload := vera.BuildVeraEvent(&test.event)
		assert.Equal(t, test.data, veraPayload)
	}
}