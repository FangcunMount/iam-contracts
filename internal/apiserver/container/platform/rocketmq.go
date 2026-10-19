package platform

import (
	"context"
	"errors"
	"time"

	"github.com/FangcunMount/iam/v5/internal/apiserver/infra/messaging"
	"github.com/FangcunMount/reliable-messaging/message"
	"github.com/FangcunMount/reliable-messaging/transport"
	nsqtransport "github.com/FangcunMount/reliable-messaging/transport/nsq"
	"github.com/FangcunMount/reliable-messaging/transport/rocketmq"
	"github.com/nsqio/go-nsq"
)

func newEventTransport(deps EventingDeps, eventing *Eventing) (ownedEventTransport, error) {
	if len(deps.Messaging.Transactions) > 0 {
		return nil, errors.New("transaction owners require transaction-bound event assembly; legacy Relay cannot publish them")
	}
	opts := deps.ReliableMessaging
	if deps.Messaging.SelectedProvider() == "rocketmq" {
		var topics []string
		for name := range eventing.Catalog.Config().Topics {
			topics = append(topics, eventing.Catalog.Config().Topics[name].Name)
		}
		config, err := deps.Messaging.RocketMQ(topics, opts.Concurrency, opts.PublishTimeout)
		if err != nil {
			return nil, err
		}
		p, err := rocketmq.NewPublisher(config)
		if err != nil {
			return nil, err
		}
		if err = p.Start(); err != nil {
			return nil, err
		}
		return &gatedTransport{ownedEventTransport: p, paused: !deps.Messaging.PublishEnabled}, nil
	}
	cfg := nsq.NewConfig()
	cfg.DialTimeout = opts.PublishTimeout
	cfg.ReadTimeout = opts.PublishTimeout
	cfg.WriteTimeout = opts.PublishTimeout
	cfg.HeartbeatInterval = min(opts.PublishTimeout/2, 30*time.Second)
	routes := map[string]string{}
	for _, topic := range eventing.Catalog.Config().Topics {
		routes[topic.Name] = topic.Name
	}
	return nsqtransport.NewManagedPublisher(nsqtransport.ManagedPublisherConfig{Address: deps.NSQAddress, Driver: cfg, Routes: routes, MaxInFlight: opts.Concurrency})
}

type gatedTransport struct {
	ownedEventTransport
	paused bool
}

func (p *gatedTransport) Publish(ctx context.Context, m message.Message) transport.Result {
	if p.paused {
		return transport.Result{Outcome: transport.Unknown}
	}
	return p.ownedEventTransport.Publish(ctx, m)
}
func (p *gatedTransport) PublishRaw(ctx context.Context, topic string, body []byte) transport.Result {
	if p.paused {
		return transport.Result{Outcome: transport.Unknown}
	}
	return p.ownedEventTransport.PublishRaw(ctx, topic, body)
}

// Gate before Run so maintenance never claims or spends an Outbox retry budget.
type publicationGate struct {
	runner messaging.ReliableRunner
	paused bool
}

func (g publicationGate) Run(ctx context.Context) error {
	if g.paused {
		<-ctx.Done()
		return ctx.Err()
	}
	return g.runner.Run(ctx)
}
