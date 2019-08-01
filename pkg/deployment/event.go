package deployment

import (
	"time"
)

func nonEmpty(data map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range data {
		if len(v) > 0 {
			result[k] = v
		}
	}
	return result
}

// Flatten returns all non-empty values as a key -> value hash.
// Nested data structures are flattened with key names joined using an underscore.
func (m *Event) Flatten() map[string]string {
	return nonEmpty(map[string]string{
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
		"skya_environment": m.GetSkyaEnvironment(),
		"namespace":        m.GetNamespace(),
		"cluster":          m.GetCluster(),
		"application":      m.GetApplication(),
		"version":          m.GetVersion(),
		"image_name":       m.GetImage().GetName(),
		"image_tag":        m.GetImage().GetTag(),
		"image_hash":       m.GetImage().GetHash(),
	})
}

func (m *Event) GetTimestampAsTime() time.Time {
	return time.Unix(m.GetTimestamp().GetSeconds(), int64(m.GetTimestamp().GetNanos()))
}
