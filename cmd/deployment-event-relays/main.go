package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/Shopify/sarama"
	"github.com/nais/liberator/pkg/conftools"
	"github.com/nais/liberator/pkg/tlsutil"
	"github.com/navikt/deployment-event-relays/pkg/config"
	"github.com/navikt/deployment-event-relays/pkg/deployment"
	"github.com/navikt/deployment-event-relays/pkg/influx"
	"github.com/navikt/deployment-event-relays/pkg/kafka/consumer"
	"github.com/navikt/deployment-event-relays/pkg/logging"
	"github.com/navikt/deployment-event-relays/pkg/metrics"
	"github.com/navikt/deployment-event-relays/pkg/nora"
	"github.com/navikt/deployment-event-relays/pkg/null"
	"github.com/navikt/deployment-event-relays/pkg/vera"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

type Processor interface {
	Process(event *deployment.Event) (retry bool, err error)
}

func kafkaConfig(cfg *config.Config, subsystem string, callback consumer.Callback) (*consumer.Config, error) {
	tlsConfig, err := tlsutil.TLSConfigFromFiles(
		cfg.Kafka.TLS.CertificatePath,
		cfg.Kafka.TLS.PrivateKeyPath,
		cfg.Kafka.TLS.CAPath,
	)
	if err != nil {
		return nil, err
	}
	return &consumer.Config{
		Brokers:           cfg.Kafka.Brokers,
		Callback:          callback,
		GroupID:           cfg.Kafka.GroupIDPrefix + "/" + subsystem,
		MaxProcessingTime: time.Second * 3,
		Logger:            log.StandardLogger(),
		RetryInterval:     time.Second * 30,
		TlsConfig:         tlsConfig,
		Topic:             cfg.Kafka.Topic,
	}, nil
}

func main() {
	err := run()
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.DefaultConfig()
	config.BindFlags(cfg)

	conftools.Initialize("DER")
	config.BindNAIS()
	err := conftools.Load(cfg)
	if err != nil {
		return err
	}

	pflag.Parse()

	err = logging.Apply(log.StandardLogger(), cfg.Log.Verbosity, cfg.Log.Format)
	if err != nil {
		return err
	}

	log.Infof("deployment-event-relays starting up")
	log.Infof("--- configuration ---")

	disallowedKeys := []string{
		"influxdb.password",
	}
	for _, configLine := range conftools.Format(disallowedKeys) {
		log.Info(configLine)
	}

	log.Infof("--- end configuration ---")

	sarama.Logger = log.StandardLogger()

	go func() {
		err := http.ListenAndServe(cfg.Metrics.BindAddress, promhttp.Handler())
		if err != nil {
			log.Errorf("Serve metrics: %s", err)
			os.Exit(2)
		}
	}()

	subsystems := make(map[string]Processor)

	if len(cfg.InfluxDB.URL) > 0 {
		subsystems["influxdb"] = &influx.Relay{
			URL:      cfg.InfluxDB.URL,
			Username: cfg.InfluxDB.Username,
			Password: cfg.InfluxDB.Password,
		}
	}

	if len(cfg.Nora.URL) > 0 {
		subsystems["nora"] = &nora.Relay{
			URL: cfg.InfluxDB.URL,
		}
	}

	if len(cfg.Vera.URL) > 0 {
		subsystems["vera"] = &vera.Relay{
			URL: cfg.Vera.URL,
		}
	}

	if cfg.Null.Enabled {
		subsystems["null"] = &null.Relay{}
	}

	if len(subsystems) == 0 {
		return fmt.Errorf("no subsystems enabled")
	}

	setup := func(key string, relayer Processor) error {
		callback := func(message *sarama.ConsumerMessage, logger *log.Entry) (retry bool, err error) {
			event := &deployment.Event{}
			any := &anypb.Any{}
			err = proto.Unmarshal(message.Value, any)
			if err != nil {
				// unknown types are dropped silently
				metrics.Process(key, metrics.LabelValueProcessedDropped, message.Offset+1)
				return false, nil
			}

			logger = logger.WithFields(log.Fields{
				"subsystem":      key,
				"correlation_id": event.GetCorrelationID(),
			})

			js, err := json.Marshal(event)
			if err != nil {
				logger.Errorf("Incoming message, but unable to render: %s", err)
			} else {
				logger.Tracef("Incoming message: %s", js)
			}
			retry, err = relayer.Process(event)
			if err == nil {
				logger.Infof("Successfully processed message")
				metrics.Process(key, metrics.LabelValueProcessedOK, message.Offset+1)
			} else {
				if retry {
					metrics.Process(key, metrics.LabelValueProcessedRetry, message.Offset)
				} else {
					metrics.Process(key, metrics.LabelValueProcessedError, message.Offset+1)
				}
			}
			return
		}
		metrics.Init(key)
		kafkacfg, err := kafkaConfig(cfg, key, callback)
		if err != nil {
			return fmt.Errorf("initialize configuration: %w", err)
		}
		_, err = consumer.New(*kafkacfg)
		if err != nil {
			return fmt.Errorf("initialize Kafka for subsystem %w", err)
		}
		return nil
	}

	for key, relayer := range subsystems {
		err := setup(key, relayer)
		if err != nil {
			return fmt.Errorf("setup subsystem '%s': %w", key, err)
		}
		log.Infof("Enabled subsystem '%s'", key)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	sig := <-sigs

	log.Infof("Received %s, shutting down.", sig)

	return nil
}
