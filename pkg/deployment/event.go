package deployment

import (
	log "github.com/sirupsen/logrus"
	"time"
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
		"timestamp":        time.Unix(m.GetTimestamp(), 0),
	}
}
