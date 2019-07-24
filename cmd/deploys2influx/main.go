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
	"net/url"
	"os"
	"os/signal"
)

type configuration struct {
	LogFormat    string
	LogVerbosity string
	Ack          bool
	URL          string
	Database     string
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
		URL:          "http://localhost:8086",
		Database:     "default",
	}
}

func init() {
	signals = make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	flag.StringVar(&cfg.LogFormat, "log-format", cfg.LogFormat, "Log format, either 'json' or 'text'.")
	flag.StringVar(&cfg.LogVerbosity, "log-verbosity", cfg.LogVerbosity, "Logging verbosity level.")
	flag.BoolVar(&cfg.Ack, "ack", cfg.Ack, "Acknowledge messages in Kafka queue, i.e. store consumer group position.")
	flag.StringVar(&cfg.URL, "influxdb-url", cfg.URL, "Root URL to InfluxDB 1.7 write endpoint.")
	flag.StringVar(&cfg.Database, "influxdb-database", cfg.Database, "InfluxDB database name.")

	kafkaconfig.SetupFlags(&kafkaConfig)
}

func main() {
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

func prepare(msg consumer.Message) (*deployment.Event, *log.Entry, error) {
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

func writeurl(baseURL string) (string, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("parsing InfluxDB base url: %s", err)
	}
	queryParams := url.Values{}
	queryParams.Set("db", cfg.Database)
	base.Path = "/write"
	base.RawQuery = queryParams.Encode()
	return base.String(), nil
}

func request(url string, event deployment.Event) error {
	line := influx.NewLine(&event)
	payload, err := line.Marshal()
	if err != nil {
		return fmt.Errorf("unable to marshal InfluxDB payload: %s", err)
	}

	body := bytes.NewReader(payload)
	request, err := http.NewRequest("POST", url, body)
	if err != nil {
		return fmt.Errorf("unable to create new HTTP request object: %s", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf("post to InfluxDB: %s", err)
	}

	if response.StatusCode > 299 {
		return fmt.Errorf("POST %s: %s", url, response.Status)
	}

	return nil
}

func run() error {
	var err error

	flag.Parse()

	log.SetOutput(os.Stdout)

	influxURL, err := writeurl(cfg.URL)
	if err != nil {
		return err
	}

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

	for {
		select {
		case sig := <-signals:
			log.Infof("Caught signal '%s'; exiting.", sig.String())
			return nil

		case msg, ok := <-kafka.Consume():
			if !ok {
				return fmt.Errorf("kafka consumer has shut down")
			}

			// Extract event data from the message.
			// Errors are non-recoverable and can only be logged.
			event, logger, err := prepare(msg)
			if err == nil {

				// Perform the request against InfluxDB.
				// These errors are possibly recoverable.
				err = request(influxURL, *event)
			}

			if err != nil {
				logger.Error(err)
			} else {
				logger.Info("Successfully inserted line data into InfluxDB")
			}

			// Mark the message as processed.
			if cfg.Ack {
				msg.Ack()
			}
		}
	}
}
