package main

import (
	"fmt"
	"github.com/navikt/deployment-event-relays/pkg/kafka/config"
	"github.com/navikt/deployment-event-relays/pkg/kafka/consumer"
	"github.com/navikt/deployment-event-relays/pkg/logging"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"os"
	"os/signal"
)

var signals chan os.Signal

func init() {
	signals = make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
}

func main() {
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

func run() error {
	kafkaConfig := config.DefaultConsumer()
	config.SetupFlags(&kafkaConfig)
	flag.Parse()

	kafkaLogger, err := logging.ConstLevel(kafkaConfig.Verbosity, "text")
	if err != nil {
		return err
	}
	config.SetLogger(kafkaLogger)

	err = logging.Apply(log.StandardLogger(), "INFO", "text")
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
			log.Info(data)
			msg.Ack()
		}
	}
}
