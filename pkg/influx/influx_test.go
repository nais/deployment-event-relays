package influx_test

import (
	"testing"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/navikt/deployment-event-relays/pkg/deployment"
	"github.com/navikt/deployment-event-relays/pkg/influx"
	"github.com/stretchr/testify/assert"
)

type eventLineTest struct {
	event deployment.Event
	data  string
	err   error
}

var eventLineTests = []eventLineTest{
	{
		data:  `nais.deployment,environment=production,platform_type=jboss,rollout_status=unknown source="aura" 0` + "\n",
		event: deployment.Event{},
	},
	{
		data: `nais.deployment,application=app,cluster=clust,environment=development,namespace=ns,platform_type=nais,rollout_status=complete,team=tea ` +
			`correlation_id="id",deployer_email="bar",deployer_ident="baz",deployer_name="foo",` +
			`image_hash="impossible",image_name="docker.io/foo/bar",image_tag="latest",skya_environment="skya",source="naiserator",version="1.2.3" 123456789000000000` + "\n",
		event: deployment.Event{
			Application:   "app",
			Cluster:       "clust",
			CorrelationID: "id",
			Deployer: &deployment.Actor{
				Name:  "foo",
				Email: "bar",
				Ident: "baz",
			},
			Environment: deployment.Environment_development,
			Image: &deployment.ContainerImage{
				Name: "docker.io/foo/bar",
				Tag:  "latest",
				Hash: "impossible",
			},
			Namespace: "ns",
			Platform: &deployment.Platform{
				Type: deployment.PlatformType_nais,
			},
			RolloutStatus:   deployment.RolloutStatus_complete,
			SkyaEnvironment: "skya",
			Source:          deployment.System_naiserator,
			Team:            "tea",
			Timestamp: &timestamp.Timestamp{
				Seconds: 123456789,
			},
			Version: "1.2.3",
		},
	},
}

func TestEventLineData(t *testing.T) {
	for _, test := range eventLineTests {
		line := influx.NewLine(&test.event)
		data, err := line.Marshal()
		assert.Equal(t, test.err, err)
		if err == nil {
			assert.Equal(t, test.data, string(data))
		}
	}
}
