package config

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"

	"github.com/spf13/pflag"
)

type Log struct {
	Format    string `json:"format"`
	Verbosity string `json:"verbosity"`
}

type InfluxDB struct {
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type Vera struct {
	URL string `json:"url"`
}

type Nora struct {
	URL string `json:"url"`
}

type KafkaTLS struct {
	CAPath          string `json:"ca-path"`
	CertificatePath string `json:"certificate-path"`
	PrivateKeyPath  string `json:"private-key-path"`
}

type Kafka struct {
	Brokers       []string `json:"brokers"`
	TLS           KafkaTLS `json:"tls"`
	Topic         string   `json:"topic"`
	GroupIDPrefix string   `json:"group-id-prefix"`
}

type Config struct {
	Log      Log      `json:"log"`
	InfluxDB InfluxDB `json:"influxdb"`
	Nora     Nora     `json:"nora"`
	Vera     Vera     `json:"vera"`
	Kafka    Kafka    `json:"kafka"`
}

func DefaultConfig() *Config {
	return &Config{
		Log: Log{
			Format:    "text",
			Verbosity: "trace",
		},
	}
}

func BindFlags(cfg *Config) {
	pflag.StringSliceVar(&cfg.Kafka.Brokers, "kafka.brokers", cfg.Kafka.Brokers, "")
	pflag.StringVar(&cfg.Kafka.Topic, "kafka.topic", cfg.Kafka.Topic, "")
	pflag.StringVar(&cfg.Kafka.GroupIDPrefix, "kafka.group-id-prefix", cfg.Kafka.GroupIDPrefix, defaultGroupIDPrefix())
	pflag.StringVar(&cfg.Kafka.TLS.CAPath, "kafka.tls.ca-path", cfg.Kafka.TLS.CAPath, "")
	pflag.StringVar(&cfg.Kafka.TLS.CertificatePath, "kafka.tls.certificate-path", cfg.Kafka.TLS.CAPath, "")
	pflag.StringVar(&cfg.Kafka.TLS.PrivateKeyPath, "kafka.tls.private-key-path", cfg.Kafka.TLS.PrivateKeyPath, "")

	pflag.StringVar(&cfg.Log.Format, "log.format", cfg.Log.Format, "text")
	pflag.StringVar(&cfg.Log.Verbosity, "log.verbosity", cfg.Log.Verbosity, "trace")

	pflag.StringVar(&cfg.InfluxDB.URL, "influxdb.url", cfg.InfluxDB.URL, "")
	pflag.StringVar(&cfg.InfluxDB.Username, "influxdb.username", cfg.InfluxDB.Username, "")
	pflag.StringVar(&cfg.InfluxDB.Password, "influxdb.password", cfg.InfluxDB.Password, "")

	pflag.StringVar(&cfg.Vera.URL, "vera.url", cfg.Vera.URL, "")

	pflag.StringVar(&cfg.Nora.URL, "nora.url", cfg.Nora.URL, "")
}

func defaultGroupIDPrefix() string {
	var suffix string
	if hostname, err := os.Hostname(); err == nil {
		suffix = hostname
	} else {
		suffix = strconv.Itoa(rand.Int())
	}
	return fmt.Sprintf("%s-%s", os.Args[0], suffix)
}
