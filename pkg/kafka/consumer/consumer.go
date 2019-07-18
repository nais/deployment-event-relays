package consumer

import (
	"crypto/tls"
	"fmt"
	"github.com/navikt/deployment-event-relays/pkg/kafka/config"
	"github.com/sirupsen/logrus"
	"sync"

	"github.com/Shopify/sarama"
	"github.com/bsm/sarama-cluster"
)

type Message struct {
	M   sarama.ConsumerMessage
	Ack func()
}

const (
	recvQueueSize = 32768
)

type client struct {
	recvQ    chan Message
	consumer *cluster.Consumer
	once     sync.Once
}

type Client interface {
	Consume() chan Message
}

func New(cfg config.Consumer) (Client, error) {
	var err error

	client := &client{
		recvQ: make (chan Message, recvQueueSize),
	}

	// Instantiate a Kafka client operating in consumer group mode,
	// starting from the oldest unread offset.

	client.consumer, err = cluster.NewConsumer(cfg.Brokers, cfg.GroupID, []string{cfg.Topic}, clusterConfig(cfg))
	if err != nil {
		return nil, fmt.Errorf("while setting up Kafka consumer: %s", err)
	}

	return client, nil
}

// Consume starts consuming messages from Kafka, and returns
// a channel on which the messages will arrive. The function
// call is idempotent.
//
// Callers may store their position on the Kafka topic by calling Ack()
// on the returned message.
func (c *client) Consume() chan Message {
	c.once.Do(func() { go c.main() })
	return c.recvQ
}

// Put messages on the queue until the channel is closed.
func (c *client) main() {
	for {
		msg, err := c.next()
		if err != nil {
			close(c.recvQ)
			return
		}
		c.recvQ <- *msg
	}
}

// Consume one message from Kafka, wrap it together with an Ack closure,
// and return it to caller.
func (c *client) next() (*Message, error) {
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

func (m *Message) LogFields() logrus.Fields {
	if m == nil {
		return logrus.Fields{}
	}
	return logrus.Fields{
		"kafka_topic":  m.M.Topic,
		"kafka_offset": m.M.Offset,
	}
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
	if cfg.TLS.Enabled {
		c.Net.TLS.Config = &tls.Config{
			InsecureSkipVerify: cfg.TLS.Insecure,
		}
	}
	return c
}
