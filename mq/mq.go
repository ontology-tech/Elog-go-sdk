package mq

import (
	"errors"

	"github.com/streadway/amqp"
	"github.com/orange-protocol/Elog-go-sdk/utils"
)

type Consumer struct {
	conn *amqp.Connection
	channel *amqp.Channel
	topics map[string]struct{}
}

func NewConsumer(addr string) *Consumer {
	conn, err := amqp.Dial(addr)
	if err != nil{
		panic("get rabbit client conn fail")
	}
	//创建一个Channel，所有的连接都是通过Channel管理的
	channel, err := conn.Channel()
	if err != nil{
		panic("get rabbit client channel fail")
	}
	return &Consumer{
		conn: conn,
		channel: channel,
		topics: make(map[string]struct{}),
	}
}

// one contract get one channel
func (consumer *Consumer) RegisterTopic(topic string) (<-chan amqp.Delivery, error) {
	if _, ok := consumer.topics[topic]; ok {
		return nil, utils.ErrTopicHasRegistered
	}
	queue, err := consumer.channel.QueueDeclare(topic, true, false, false, false, nil)
	if err != nil{
		return nil, errors.New("get queue fail")
	}
	msgs, err := consumer.channel.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		return nil, err
	}
	consumer.topics[topic] = struct{}{}
	return msgs, nil
}

func (consumer *Consumer) UnregisterTopic(topic string) {
	if _, ok := consumer.topics[topic]; ok {
		delete(consumer.topics, topic)
	}
}