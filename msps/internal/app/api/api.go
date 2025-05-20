package api

import "github.com/google/wire"

var ProviderSet = wire.NewSet(
	ProvideAgentSet,
	ProvideClientSet,
	ProvideMailQueue,
)

func ProvideAgentSet() *Agent {
	return &Agent{}
}

func ProvideClientSet() *Client {
	return &Client{}
}

func ProvideMailQueue() *MailQueue {
	return NewMailQueue(defaultQueueCapacity)
}
