package rabbit

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"ride-hail-system/internal/logger"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Client struct {
	url string
	log logger.Logger

	mu   sync.RWMutex
	conn *amqp.Connection
	ch   *amqp.Channel

	once sync.Once
}

func New(url string, l logger.Logger) *Client {
	return &Client{url: url, log: l}
}

func (c *Client) ConnectAndSetup(ctx context.Context) error {
	conn, err := amqp.Dial(c.url)
	if err != nil {
		return err
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return err
	}
	if err := ch.Qos(20, 0, false); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return err
	}
	if err := declare(ch); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return err
	}

	c.mu.Lock()
	oldCh, oldConn := c.ch, c.conn
	c.ch, c.conn = ch, conn
	c.mu.Unlock()

	if oldCh != nil {
		_ = oldCh.Close()
	}
	if oldConn != nil {
		_ = oldConn.Close()
	}

	c.once.Do(func() { go c.reconnectLoop(ctx) })
	c.log.Info("mq_ready", "rabbitmq connected", "", "")
	return nil
}

func (c *Client) reconnectLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		c.mu.RLock()
		conn := c.conn
		c.mu.RUnlock()
		if conn == nil {
			time.Sleep(time.Second)
			continue
		}

		err := <-conn.NotifyClose(make(chan *amqp.Error, 1))
		if err == nil {
			return
		}
		c.log.Error("mq_disconnected", "rabbitmq disconnected", errors.New(err.Reason), "", "")

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			if e := c.ConnectAndSetup(ctx); e == nil {
				c.log.Info("mq_reconnected", "rabbitmq reconnected", "", "")
				break
			}
			time.Sleep(2 * time.Second)
		}
	}
}

func (c *Client) PublishJSON(ctx context.Context, exchange, routingKey string, payload any) error {
	c.mu.RLock()
	ch := c.ch
	c.mu.RUnlock()
	if ch == nil {
		return errors.New("mq not ready")
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return ch.PublishWithContext(ctx, exchange, routingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         b,
		Timestamp:    time.Now().UTC(),
	})
}

func (c *Client) Consume(queue string) (<-chan amqp.Delivery, error) {
	c.mu.RLock()
	ch := c.ch
	c.mu.RUnlock()
	if ch == nil {
		return nil, errors.New("mq not ready")
	}
	return ch.Consume(queue, "", false, false, false, false, nil)
}

func declare(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare("ride_topic", "topic", true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.ExchangeDeclare("driver_topic", "topic", true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.ExchangeDeclare("location_fanout", "fanout", true, false, false, false, nil); err != nil {
		return err
	}

	qs := []string{"ride_requests", "ride_status", "driver_matching", "driver_responses", "driver_status", "location_updates_ride"}
	for _, q := range qs {
		if _, err := ch.QueueDeclare(q, true, false, false, false, nil); err != nil {
			return err
		}
	}

	if err := ch.QueueBind("ride_requests", "ride.request.*", "ride_topic", false, nil); err != nil {
		return err
	}
	if err := ch.QueueBind("driver_matching", "ride.request.*", "ride_topic", false, nil); err != nil {
		return err
	}
	if err := ch.QueueBind("ride_status", "ride.status.*", "ride_topic", false, nil); err != nil {
		return err
	}
	if err := ch.QueueBind("driver_responses", "driver.response.*", "driver_topic", false, nil); err != nil {
		return err
	}
	if err := ch.QueueBind("driver_status", "driver.status.*", "driver_topic", false, nil); err != nil {
		return err
	}
	if err := ch.QueueBind("location_updates_ride", "", "location_fanout", false, nil); err != nil {
		return err
	}
	return nil
}
