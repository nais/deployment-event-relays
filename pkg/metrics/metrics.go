package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type ProcessStatus string

const (
	labelStatus    = "status"
	labelSubsystem = "subsystem"

	LabelValueProcessedOK      ProcessStatus = "ok"
	LabelValueProcessedDropped ProcessStatus = "dropped"
	LabelValueProcessedError   ProcessStatus = "error"
	LabelValueProcessedRetry   ProcessStatus = "retry"
)

var (
	messages = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "deployment_event_relays",
		Name:      "messages",
		Help:      "Number of deployment event messages processed, labeled by status",
	}, []string{
		labelSubsystem,
		labelStatus,
	})
)

func Init(subsystem string) {
	messages.WithLabelValues(subsystem, string(LabelValueProcessedOK)).Add(0)
	messages.WithLabelValues(subsystem, string(LabelValueProcessedDropped)).Add(0)
	messages.WithLabelValues(subsystem, string(LabelValueProcessedError)).Add(0)
	messages.WithLabelValues(subsystem, string(LabelValueProcessedRetry)).Add(0)
}

func Process(subsystem string, status ProcessStatus) {
	messages.WithLabelValues(subsystem, string(status)).Inc()
}

func init() {
	prometheus.MustRegister(messages)
}
