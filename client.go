package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"golang.org/x/sync/syncmap"
	"golang.skroutz.gr/skroutz/rafka/kafka"
)

type Client struct {
	id          string
	consumerGID string
	manager     *ConsumerManager
	log         *log.Logger
	ready       bool

	consumers map[ConsumerID]bool
	byTopic   map[string]ConsumerID
}

func NewClient(cm *ConsumerManager) *Client {
	c := Client{
		manager:   cm,
		consumers: make(map[ConsumerID]bool),
		byTopic:   make(map[string]ConsumerID),
		log:       log.New(os.Stderr, "[client] ", log.Ldate|log.Ltime)}

	return &c
}

// SetID sets the id for c.
//
// It returns an error if id is not in the form of "<group.id>:<client-name>".
func (c *Client) SetID(id string) error {
	parts := strings.SplitN(id, ":", 2)
	if len(parts) != 2 {
		return errors.New("Cannot parse group.id")
	}

	c.id = id
	c.consumerGID = parts[0]
	c.ready = true

	return nil
}

func (c *Client) String() string {
	return c.id
}

func (c *Client) Consumer(topics []string) (*kafka.Consumer, error) {
	if !c.ready {
		return nil, errors.New("Connection is not ready, please identify before using")
	}

	consumerID := ConsumerID(fmt.Sprintf("%s|%s", c.id, strings.Join(topics, ",")))

	// Check for topics that already have a consumer
	for _, topic := range topics {
		if existingID, ok := c.byTopic[topic]; ok {
			if existingID != consumerID {
				return nil, fmt.Errorf("Topic %s has another consumer", topic)
			}
		}
	}

	// Register the Consumer
	c.consumers[consumerID] = true
	for _, topic := range topics {
		c.byTopic[topic] = consumerID
	}

	return c.manager.Get(consumerID, c.consumerGID, topics), nil
}

func (c *Client) ConsumerByTopic(topic string) (*kafka.Consumer, error) {
	consumerID, ok := c.byTopic[topic]
	if !ok {
		return nil, fmt.Errorf("No consumer for topic %s", topic)
	}

	consumer, err := c.manager.ByID(consumerID)
	if err != nil {
		return nil, err
	}

	return consumer, nil
}

func (c *Client) Teardown(clientIDs *syncmap.Map) {
	for cid := range c.consumers {
		c.log.Printf("[%s] Scheduling teardown for %s", c.id, cid)
		c.manager.Delete(cid)
	}

	clientIDs.Delete(c.id)
}