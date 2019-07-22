package influx_test

import (
	"github.com/navikt/deployment-event-relays/pkg/deployment"
	"github.com/navikt/deployment-event-relays/pkg/influx"
	"github.com/stretchr/testify/assert"
	"testing"
)

type eventLineTest struct {
	event deployment.Event
	data  string
	err   error
}

var eventLineTests = []eventLineTest{
	{
		data: "nais.deployment,application=app,cluster=cluster,environment=development,platform_type=nais,rollout_status=complete,team=team source=naiserator,version=1.2.3 123456789000000000\n",
		event: deployment.Event{
			Application: "app",
			Cluster:     "cluster",
			Environment: deployment.Environment_development,
			Source:      deployment.System_naiserator,
			Team:        "team",
			Platform: &deployment.Platform{
				Type: deployment.PlatformType_nais,
			},
			RolloutStatus: deployment.RolloutStatus_complete,
			Version:       "1.2.3",
			Timestamp:     123456789,
		},
	},
}

func TestEventLineData(t *testing.T) {
	for _, test := range eventLineTests {
		line := influx.NewLine(&test.event)
		data, err := line.Marshal()
		assert.Equal(t, test.data, string(data))
		assert.Equal(t, test.err, err)
	}
}
