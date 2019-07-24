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
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"net/http"
	"os"
	"os/signal"
	"time"
)

type configuration struct {
	LogFormat    string
	LogVerbosity string
	Ack          bool
	URL          string
}

type postCallback func() error

type Workload struct {
	Callback postCallback
	Logger   log.Entry
}

var (
	signals     chan os.Signal
	cfg         = defaultConfig()
	kafkaConfig = kafkaconfig.DefaultConsumer()
)

func defaultConfig() configuration {
	return configuration{
		LogFormat:    "text",
		LogVerbosity: "trace",
		Ack:          false,
		URL:          "http://localhost:8086/write?db=default",
	}
}

func init() {
	signals = make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	flag.StringVar(&cfg.LogFormat, "log-format", cfg.LogFormat, "Log format, either 'json' or 'text'.")
	flag.StringVar(&cfg.LogVerbosity, "log-verbosity", cfg.LogVerbosity, "Logging verbosity level.")
	flag.BoolVar(&cfg.Ack, "ack", cfg.Ack, "Acknowledge messages in Kafka queue, i.e. store consumer group position.")
	flag.StringVar(&cfg.URL, "influxdb-url", cfg.URL, "URL to InfluxDB 1.7 write endpoint.")

	kafkaconfig.SetupFlags(&kafkaConfig)
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
			retry *= 2
			workload.Logger.Errorf("%s; retrying in %s", err, retry.String())
			time.Sleep(retry)
		}
	}
}

func run() error {
	var err error

	flag.Parse()

	log.SetOutput(os.Stdout)

	kafkaLogger, err := logging.ConstLevel(kafkaConfig.Verbosity, cfg.LogFormat)
	if err != nil {
		return err
	}
	kafkaconfig.SetLogger(kafkaLogger)

	err = logging.Apply(log.StandardLogger(), cfg.LogVerbosity, cfg.LogFormat)
	if err != nil {
		return err
	}

	kafka, err := consumer.New(kafkaConfig)
	if err != nil {
		return err
	}

	// Create a goroutine that will do the actual posting to InfluxDB.
	workloads := make(chan Workload, 4096)
	go worker(workloads)

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
				callback, err = prepare(cfg.URL, *event)
			}

			// Discard the message permanently.
			if err != nil {
				logger.Errorf("Discarding incoming message due to unrecoverable error: %s")
				ack(msg)
				continue
			}

			// Create a callback function that submits data to InfluxDB and acknowledges
			// our queue position in Kafka if successful.
			workloadCallback := func() error {
				err := callback()
				if err == nil {
					ack(msg)
				}
				return err
			}

			// Submit the callback function to the worker goroutine.
			workloads <- Workload{
				Callback: workloadCallback,
				Logger:   *logger,
			}
		}
	}
}
