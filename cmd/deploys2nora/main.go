package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/navikt/deployment-event-relays/pkg/deployment"
	kafkaconfig "github.com/navikt/deployment-event-relays/pkg/kafka/config"
	"github.com/navikt/deployment-event-relays/pkg/kafka/consumer"
	"github.com/navikt/deployment-event-relays/pkg/logging"
	"github.com/navikt/deployment-event-relays/pkg/nora"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

const (
	MetricNamespace = "deployment"
	MetricSubsystem = "deploys2nora"
)

type configuration struct {
	LogFormat         string
	LogVerbosity      string
	Ack               bool
	URL               string
	MetricsListenAddr string
	Kafka             kafkaconfig.Consumer
}

type postCallback func() error

type Workload struct {
	Callback postCallback
	Logger   log.Entry
}

var (
	signals   = make(chan os.Signal, 1)
	workloads = make(chan Workload, 4096)
	cfg       = defaultConfig()

	written = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "written",
		Help:      "Number of Nora events created",
		Namespace: MetricNamespace,
		Subsystem: MetricSubsystem,
	})

	discarded = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "discarded",
		Help:      "Number of nora events discarded due to unrecoverable errors",
		Namespace: MetricNamespace,
		Subsystem: MetricSubsystem,
	})

	transientErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "transient_errors",
		Help:      "Number of recoverable errors encountered during Nora communication",
		Namespace: MetricNamespace,
		Subsystem: MetricSubsystem,
	})

	queueSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:      "queue_size",
		Help:      "Kafka messages read but not committed to Nora",
		Namespace: MetricNamespace,
		Subsystem: MetricSubsystem,
	})
)

func defaultConfig() configuration {
	return configuration{
		LogFormat:         "text",
		LogVerbosity:      "trace",
		Ack:               false,
		URL:               "http://localhost:8080/api/apps",
		MetricsListenAddr: "127.0.0.1:8080",
		Kafka:             kafkaconfig.DefaultConsumer(),
	}
}

func init() {
	// Trap interrupts
	signal.Notify(signals, os.Interrupt)

	flag.StringVar(&cfg.LogFormat, "log-format", cfg.LogFormat, "Log format, either 'json' or 'text'.")
	flag.StringVar(&cfg.LogVerbosity, "log-verbosity", cfg.LogVerbosity, "Logging verbosity level.")
	flag.BoolVar(&cfg.Ack, "kafka-ack", cfg.Ack, "Acknowledge messages in Kafka queue, i.e. store consumer group position.")
	flag.StringVar(&cfg.URL, "nora-url", cfg.URL, "Full URL to Nora api")
	flag.StringVar(&cfg.MetricsListenAddr, "metrics-listen-addr", cfg.MetricsListenAddr, "Serve metrics on this address.")

	kafkaconfig.SetupFlags(&cfg.Kafka)

	prometheus.MustRegister(queueSize)
	prometheus.MustRegister(written)
	prometheus.MustRegister(discarded)
	prometheus.MustRegister(transientErrors)
}

func main() {
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

func extract(msg consumer.Message) (*deployment.Event, *log.Entry, error) {
	logger := log.WithFields(msg.LogFields())

	event := &deployment.Event{}
	err := proto.Unmarshal(msg.M.Value, event)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to unmarshal incoming message: %s", err)
	}

	for k, v := range event.Flatten() {
		logger = logger.WithField(k, v)
	}
	logger = logger.WithField("timestamp", event.GetTimestampAsTime().String())

	return event, logger, nil
}

func prepare(url string, event deployment.Event) (postCallback, error) {
	if event.GetEnvironment() != deployment.Environment_production {
		return nil, fmt.Errorf("event does not belong to production")
	}

	noraEvent := nora.BuildEvent(&event)
	payload, err := noraEvent.Marshal()
	log.Infof("Posting payload to Nora %s", payload)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal Nora payload: %s", err)
	}

	body := bytes.NewReader(payload)
	request, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("unable to create new HTTP request object: %s", err)
	}
	request.Header.Set("Content-Type", "application/json")

	// Create a callback function wrapping recoverable network errors.
	// This callback will be run as many times as necessary in order to
	// ensure the data is fully written to Nora.
	post := func() error {
		body.Reset(payload)
		response, err := http.DefaultClient.Do(request)
		defer response.Body.Close()
		if err != nil {
			return fmt.Errorf("post to Nora: %s", err)
		}

		switch response.StatusCode {
		case http.StatusOK: // Application updated
			return nil
		case http.StatusCreated: // Application created
			return nil
		case http.StatusUnprocessableEntity: // Application is already registered
			return nil
		default:
			return fmt.Errorf("POST %s: %s", url, response.Status)
		}
	}

	return post, nil
}

func ack(msg consumer.Message) {
	if cfg.Ack {
		log.Infof("Ack offset %d", msg.M.Offset)
		msg.Ack()
	}
}

// Execute incoming work (post data to Nora) and retry indefinitely.
func worker(work chan Workload) {
	var err error
	for workload := range work {
		retry := time.Second * 1
		for {
			err = workload.Callback()
			if err == nil {
				workload.Logger.Infof("Wrote data to Nora")
				break
			}
			transientErrors.Inc()
			retry *= 2
			workload.Logger.Errorf("%s; retrying in %s", err, retry.String())
			time.Sleep(retry)
		}
	}
}

func run() error {
	var err error

	flag.Parse()

	log.SetOutput(os.Stderr)

	kafkaLogger, err := logging.ConstLevel(cfg.Kafka.Verbosity, cfg.LogFormat)
	if err != nil {
		return err
	}
	kafkaconfig.SetLogger(kafkaLogger)

	err = logging.Apply(log.StandardLogger(), cfg.LogVerbosity, cfg.LogFormat)
	if err != nil {
		return err
	}

	kafka, err := consumer.New(cfg.Kafka)
	if err != nil {
		return err
	}

	// Create a goroutine that will do the actual posting to Nora.
	go worker(workloads)

	// Serve metrics
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		err := http.ListenAndServe(cfg.MetricsListenAddr, nil)
		if err != nil {
			log.Errorf("fatal error in HTTP handler: %s", err)
			log.Error("terminating application")
			signals <- syscall.SIGTERM
		}
	}()

	for {
		select {
		case sig := <-signals:
			log.Infof("Caught signal '%s'; exiting.", sig.String())
			return nil

		case msg, ok := <-kafka.Consume():
			if !ok {
				return fmt.Errorf("kafka consumer has shut down")
			}

			var callback postCallback

			// Extract event data from the message.
			// Errors are non-recoverable.
			event, logger, err := extract(msg)
			if err == nil {

				if event.GetRolloutStatus() != deployment.RolloutStatus_complete {
					logger.Infof("Discarding message because rollout status is != complete")
					ack(msg)
				}

				// Prepare a closure that will post our data to Nora.
				// An error here is also non-recoverable.
				callback, err = prepare(cfg.URL, *event)
			}

			// Discard the message permanently.
			if err != nil {
				discarded.Inc()
				logger.Errorf("Discarding incoming message due to unrecoverable error: %s", err)
				ack(msg)
				continue
			}

			// Create a callback function that submits data to Nora and acknowledges
			// our queue position in Kafka if successful.
			workloadCallback := func() error {
				err := callback()
				if err == nil {
					queueSize.Dec()
					written.Inc()
					ack(msg)
				}
				return err
			}

			// Submit the callback function to the worker goroutine.
			workloads <- Workload{
				Callback: workloadCallback,
				Logger:   *logger,
			}
			queueSize.Inc()
		}
	}
}
