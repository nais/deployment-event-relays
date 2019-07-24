package main

import (
	"bytes"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/navikt/deployment-event-relays/pkg/deployment"
	"github.com/navikt/deployment-event-relays/pkg/influx"
	kafkaconfig "github.com/navikt/deployment-event-relays/pkg/kafka/config"
	"github.com/navikt/deployment-event-relays/pkg/kafka/consumer"
	"github.com/navikt/deployment-event-relays/pkg/logging"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type configuration struct {
	LogFormat         string
	LogVerbosity      string
	Ack               bool
	URL               string
	Username          string
	Password          string
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
		Help:      "Number of InfluxDB lines inserted",
		Namespace: "deployment",
		Subsystem: "deploys2influx",
	})

	discarded = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "discarded",
		Help:      "Number of events discarded due to unrecoverable errors",
		Namespace: "deployment",
		Subsystem: "deploys2influx",
	})

	transientErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "transient_errors",
		Help:      "Number of recoverable errors encountered during InfluxDB communication",
		Namespace: "deployment",
		Subsystem: "deploys2influx",
	})

	queueSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:      "queue_size",
		Help:      "Kafka messages read but not committed to InfluxDB",
		Namespace: "deployment",
		Subsystem: "deploys2influx",
	})
)

func defaultConfig() configuration {
	return configuration{
		LogFormat:         "text",
		LogVerbosity:      "trace",
		Ack:               false,
		URL:               "http://localhost:8086/write?db=default",
		Username:          os.Getenv("INFLUX_USERNAME"),
		Password:          os.Getenv("INFLUX_PASSWORD"),
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
	flag.StringVar(&cfg.URL, "influxdb-url", cfg.URL, "Full URL to InfluxDB 1.7 write endpoint.")
	flag.StringVar(&cfg.Username, "influxdb-username", cfg.Username, "InfluxDB username.")
	flag.StringVar(&cfg.Password, "influxdb-password", cfg.Password, "InfluxDB password.")
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

func prepare(url, username, password string, event deployment.Event) (postCallback, error) {
	line := influx.NewLine(&event)
	payload, err := line.Marshal()
	if err != nil {
		return nil, fmt.Errorf("unable to marshal InfluxDB payload: %s", err)
	}

	body := bytes.NewReader(payload)
	request, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("unable to create new HTTP request object: %s", err)
	}

	if len(username) > 0 && len(password) > 0 {
		request.SetBasicAuth(username, password)
	}

	// Create a callback function wrapping recoverable network errors.
	// This callback will be run as many times as necessary in order to
	// ensure the data is fully written to InfluxDB.
	post := func() error {
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			return fmt.Errorf("post to InfluxDB: %s", err)
		}

		if response.StatusCode > 299 {
			return fmt.Errorf("POST %s: %s", url, response.Status)
		}

		return nil
	}

	return post, nil
}

func ack(msg consumer.Message) {
	if cfg.Ack {
		msg.Ack()
	}
}

// Execute incoming work (post data to InfluxDB) and retry indefinitely.
func worker(work chan Workload) {
	var err error
	for workload := range work {
		retry := time.Second * 1
		for {
			err = workload.Callback()
			if err == nil {
				workload.Logger.Infof("Wrote data to InfluxDB")
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

	// Create a goroutine that will do the actual posting to InfluxDB.
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

				// Prepare a closure that will post our data to InfluxDB.
				// An error here is also non-recoverable.
				callback, err = prepare(cfg.URL, cfg.Username, cfg.Password, *event)
			}

			// Discard the message permanently.
			if err != nil {
				discarded.Inc()
				logger.Errorf("Discarding incoming message due to unrecoverable error: %s", err)
				ack(msg)
				continue
			}

			// Create a callback function that submits data to InfluxDB and acknowledges
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
