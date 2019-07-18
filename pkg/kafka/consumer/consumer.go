package consumer

import (
	"crypto/tls"
	"fmt"
	"github.com/navikt/deployment-event-relays/pkg/kafka/config"

	"github.com/Shopify/sarama"
	"github.com/bsm/sarama-cluster"
)

type Message struct {
	M   sarama.ConsumerMessage
	Ack func()
}

type client struct {
	queue    chan sarama.ConsumerMessage
	consumer *cluster.Consumer
}

type Client interface {
	Next() (*Message, error)
}

func (c *client) Next() (*Message, error) {
	m, ok := <-c.consumer.Messages()
	if !ok {
		return nil, fmt.Errorf("consumer has shut down")
	}
	return &Message{
		M: *m,
		Ack: func() {
			c.consumer.MarkOffset(m, "")
		},
	}, nil
}

func clusterConfig(cfg config.Consumer) *cluster.Config {
	c := cluster.NewConfig()
	c.ClientID = cfg.ClientID
	c.Consumer.Offsets.Initial = sarama.OffsetOldest
	c.Net.SASL.Enable = cfg.SASL.Enabled
	c.Net.SASL.User = cfg.SASL.Username
	c.Net.SASL.Password = cfg.SASL.Password
	c.Net.SASL.Handshake = cfg.SASL.Handshake
	c.Net.TLS.Enable = cfg.TLS.Enabled
	c.Net.TLS.Config = &tls.Config{
		InsecureSkipVerify: cfg.TLS.Insecure,
	}
	return c
}

func New(cfg config.Consumer) (Client, error) {
	var err error

	client := &client{}

	// Instantiate a Kafka client operating in consumer group mode,
	// starting from the oldest unread offset.

	client.consumer, err = cluster.NewConsumer(cfg.Brokers, cfg.GroupID, []string{cfg.Topic}, clusterConfig(cfg))
	if err != nil {
		return nil, fmt.Errorf("while setting up Kafka consumer: %s", err)
	}

	return client, nil
}
