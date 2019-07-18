package main

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/navikt/deployment-event-relays/pkg/deployment"
	kafkaconfig "github.com/navikt/deployment-event-relays/pkg/kafka/config"
	"github.com/navikt/deployment-event-relays/pkg/kafka/consumer"
	"github.com/navikt/deployment-event-relays/pkg/logging"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"os"
	"os/signal"
)

type configuration struct {
	LogFormat    string
	LogVerbosity string
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
	}
}

func init() {
	signals = make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	flag.StringVar(&cfg.LogFormat, "log-format", cfg.LogFormat, "Log format, either 'json' or 'text'.")
	flag.StringVar(&cfg.LogVerbosity, "log-verbosity", cfg.LogVerbosity, "Logging verbosity level.")

	kafkaconfig.SetupFlags(&kafkaConfig)
}

func main() {
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

func greenField(fields log.Fields) log.Fields {
	newFields := log.Fields{}
	for k, v := range fields {
		switch x := v.(type) {
		case string:
			if len(x) > 0 {
				newFields[k] = x
			}
		default:
			newFields[k] = v
		}
	}
	return newFields
}

func run() error {
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

	messages := make(chan consumer.Message)
	go func() {
		for {
			msg, err := kafka.Next()
			if err != nil {
				close(messages)
				return
			}
			messages <- *msg
		}
	}()

	for {
		select {
		case sig := <-signals:
			log.Infof("Caught signal '%s'; exiting.", sig.String())
			return nil

		case msg, ok := <-messages:
			if !ok {
				return fmt.Errorf("kafka consumer has shut down")
			}

			data := msg.M.Value
			logger := log.WithFields(greenField(msg.LogFields()))

			event := &deployment.Event{}
			err = proto.Unmarshal(data, event)
			if err != nil {
				logger.Errorf("unable to unmarshal incoming message: %s", err)
				continue
			}

			logger = logger.WithFields(greenField(event.LogFields()))
			logger.Infof("Deployment %s", event.RolloutStatus.String())
		}
	}
}
