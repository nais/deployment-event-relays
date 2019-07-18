package deployment

import (
	log "github.com/sirupsen/logrus"
	"time"
)

const (
	millisPerSecond     = int64(time.Second / time.Millisecond)
	nanosPerMillisecond = int64(time.Millisecond / time.Nanosecond)
	distantFuture       = int64(15000000000)
)

func (m *Event) LogFields() log.Fields {
	return log.Fields{
		"correlation_id":   m.GetCorrelationID(),
		"platform_type":    m.GetPlatform().GetType().String(),
		"platform_variant": m.GetPlatform().GetVariant(),
		"source":           m.GetSource().String(),
		"deployer_name":    m.GetDeployer().GetName(),
		"deployer_email":   m.GetDeployer().GetEmail(),
		"deployer_ident":   m.GetDeployer().GetIdent(),
		"team":             m.GetTeam(),
		"rollout_status":   m.GetRolloutStatus().String(),
		"environment":      m.GetEnvironment().String(),
		"namespace":        m.GetNamespace(),
		"cluster":          m.GetCluster(),
		"application":      m.GetApplication(),
		"version":          m.GetVersion(),
		"image_name":       m.GetImage().GetName(),
		"image_tag":        m.GetImage().GetTag(),
		"image_hash":       m.GetImage().GetHash(),
		"timestamp":        m.GetTimestampAsTime(),
	}
}

// Some timestamps are milliseconds and some timestamps are seconds.
// The cutoff `distantFuture` is at 2445-05-01 02:40:00 +0000 UTC.
func (m Event) GetTimestampAsTime() time.Time {
	timestamp := m.GetTimestamp()
	if timestamp > distantFuture {
		return time.Unix(timestamp/millisPerSecond, (timestamp%millisPerSecond)*nanosPerMillisecond)
	}
	return time.Unix(timestamp, 0)
}
