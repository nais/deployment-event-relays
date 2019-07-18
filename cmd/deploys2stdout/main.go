package main

import (
	"github.com/navikt/deployment-event-relays/pkg/kafka/config"
	flag "github.com/spf13/pflag"
)

func main() {
	kafkaConfig := config.DefaultConsumer()
	config.SetupFlags(&kafkaConfig)

	flag.Parse()
}
