package config

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nais/liberator/pkg/tlsutil"
	"github.com/navikt/deployment-event-relays/pkg/kafka/consumer"
	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

func DefaultGroupName() string {
	var suffix string
	if hostname, err := os.Hostname(); err == nil {
		suffix = hostname
	} else {
		suffix = strconv.Itoa(rand.Int())
	}
	return fmt.Sprintf("%s-%s", os.Args[0], suffix)
}

func DefaultConfig() *consumer.Config {
	tlsConfig, err := tlsutil.TLSConfigFromFiles(
		os.Getenv("KAFKA_CERTIFICATE_PATH"),
		os.Getenv("KAFKA_PRIVATE_KEY_PATH"),
		os.Getenv("KAFKA_CA_PATH"),
	)
	if err != nil {
		logrus.Error(err)
	}
	return &consumer.Config{
		Brokers:           strings.Split(os.Getenv("KAFKA_BROKERS"), ","),
		Callback:          nil,
		GroupID:           DefaultGroupName(),
		MaxProcessingTime: time.Second * 3,
		Logger:            logrus.StandardLogger(),
		RetryInterval:     time.Second * 30,
		TlsConfig:         tlsConfig,
		Topic:             os.Getenv("KAFKA_TOPIC"),
	}
}

func SetupFlags(cfg *consumer.Config) {
	flag.StringSliceVar(&cfg.Brokers, "kafka-brokers", cfg.Brokers, "Comma-separated list of Kafka brokers, HOST:PORT.")
	flag.StringVar(&cfg.Topic, "kafka-topic", cfg.Topic, "Dev-rapid/deployment message Kafka topic.")
	flag.StringVar(&cfg.GroupID, "kafka-group-id", cfg.GroupID, "Kafka consumer group ID.")
	flag.DurationVar(&cfg.MaxProcessingTime, "kafka-max-processing-time", cfg.MaxProcessingTime, "Max time to use per incoming Kafka message.")
	flag.DurationVar(&cfg.RetryInterval, "kafka-retry-interval", cfg.RetryInterval, "Time between retries when message processing fails.")
}
