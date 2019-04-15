package gosub

import (
	"cloud.google.com/go/iam"
	"cloud.google.com/go/pubsub"
	"context"
	"github.com/autom8ter/api/go/api"
	"github.com/autom8ter/gosub/driver"
	"github.com/jpillora/backoff"
	"github.com/pkg/errors"
	"golang.org/x/sync/semaphore"
	"sync"
	"time"
)

var (
	mutex = &sync.Mutex{}
)

// GoSub provides google cloud pubsub
type GoSub struct {
	client   *pubsub.Client
	topics   map[string]*pubsub.Topic
	subs     map[string]context.CancelFunc
	shutdown bool
}

// NewGoSub creates a new GoSub instace for a project
func NewGoSub(projectID string) (*GoSub, error) {
	c, err := pubsub.NewClient(context.Background(), projectID)
	if err != nil {
		return nil, err
	}

	return &GoSub{
		client: c,
		topics: map[string]*pubsub.Topic{},
		subs:   map[string]context.CancelFunc{},
	}, nil
}

// Publish implements Publish
func (g *GoSub) Publish(ctx context.Context, topic string, m *driver.Msg) error {
	t, err := g.GetTopic(topic)
	if err != nil {
		return err
	}

	res := t.Publish(context.Background(), &pubsub.Message{
		Data:       m.Message.Data,
		Attributes: m.Message.Meta,
	})

	_, err = res.Get(context.Background())
	if err != nil {
		api.Util.Entry().Error(errors.Wrap(err, "publish get failed"))
	} else {
		api.Util.Entry().Debug("Google Pubsub: Publish confirmed")
	}

	return err
}

// Subscribe implements Subscribe
func (g *GoSub) Subscribe(opts driver.HandlerOptions, h driver.MsgHandler) {
	g.subscribe(opts, h, make(chan bool, 1))
}

// Shutdown shuts down all subscribers gracefully
func (g *GoSub) Shutdown() {
	g.shutdown = true

	var wg sync.WaitGroup
	for k, v := range g.subs {
		wg.Add(1)
		api.Util.Entry().Infof("Shutting down sub for %s", k)
		go func(c context.CancelFunc) {
			c()
			wg.Done()
		}(v)
	}
	wg.Wait()
	return
}

func (g *GoSub) subscribe(opts driver.HandlerOptions, h driver.MsgHandler, ready chan<- bool) {
	go func() {
		var err error
		subName := opts.ServiceName + "." + opts.Name + "--" + opts.Topic
		sub := g.client.Subscription(subName)

		t, err := g.GetTopic(opts.Topic)
		if err != nil {
			api.Util.Entry().Panicf("Can't fetch topic: %s", err.Error())
		}

		ok, err := sub.Exists(context.Background())
		if err != nil {
			api.Util.Entry().Panicf("Can't connect to pubsub: %s", err.Error())
		}

		if !ok {
			sc := pubsub.SubscriptionConfig{
				Topic:       t,
				AckDeadline: opts.Deadline,
			}
			sub, err = g.client.CreateSubscription(context.Background(), subName, sc)
			if err != nil {
				api.Util.Entry().Panicf("Can't subscribe to topic: %s", err.Error())
			}
		}

		api.Util.Entry().Infof("Subscribed to topic %s with name %s", opts.Topic, subName)
		ready <- true

		b := &backoff.Backoff{
			//These are the defaults
			Min:    200 * time.Millisecond,
			Max:    600 * time.Second,
			Factor: 2,
			Jitter: true,
		}

		// create a semaphore, this is because Google PubSub will spam
		// your service if you can't process a message
		// and will also not handle
		sem := semaphore.NewWeighted(int64(opts.Concurrency))

		// Listen to messages and call the MsgHandler
		for {
			if g.shutdown {
				break
			}

			cctx, cancel := context.WithCancel(context.Background())
			err = sub.Receive(cctx, func(ctx context.Context, m *pubsub.Message) {
				if serr := sem.Acquire(ctx, 1); serr != nil {
					api.Util.Entry().Errorf(
						"pubsub: Failed to acquire worker semaphore: %v",
						serr,
					)
					return
				}
				defer sem.Release(1)

				b.Reset()
				msg := driver.Msg{
					Message: &api.Msg{
						Id:          m.ID,
						Meta:        m.Attributes,
						Data:        m.Data,
						PublishTime: m.PublishTime.String(),
					},
					Ack: func() {
						m.Ack()
					},
					Nack: func() {
						m.Nack()
					},
				}

				err = h(ctx, msg)
				if err != nil {
					return
				}

				if opts.AutoAck {
					m.Ack()
				}
			})

			if err != nil {
				d := b.Duration()
				api.Util.Entry().Errorf(
					"Subscription receive to topic %s failed, reconnecting in %v. Err: %v",
					opts.Topic, d, err,
				)
				time.Sleep(d)
			}

			g.subs[subName] = cancel
		}
	}()
}

func (g *GoSub) GetTopic(name string) (*pubsub.Topic, error) {
	mutex.Lock()
	defer mutex.Unlock()

	if g.topics[name] != nil {
		return g.topics[name], nil
	}

	var err error
	t := g.client.Topic(name)
	ok, err := t.Exists(context.Background())
	if err != nil {
		return nil, err
	}

	if !ok {
		t, err = g.client.CreateTopic(context.Background(), name)
		if err != nil {
			return nil, err
		}
	}

	g.topics[name] = t

	return t, nil
}

func (g *GoSub) DeleteTopic(name string) error {
	t, err := g.GetTopic(name)
	if err != nil {
		return err
	}

	return t.Delete(context.Background())
}

func (g *GoSub) TopicExists(name string) (bool, error) {
	t, err := g.GetTopic(name)
	if err != nil {
		return false, err
	}

	return t.Exists(context.Background())
}


func (g *GoSub) TopicIAM(name string) (*iam.Handle, error) {
	t, err := g.GetTopic(name)
	if err != nil {
		return nil, err
	}

	return t.IAM(), nil
}


func (g *GoSub) TopicSubscriptions(name string, ctx context.Context) (*pubsub.SubscriptionIterator, error) {
	t, err := g.GetTopic(name)
	if err != nil {
		return nil, err
	}
	return t.Subscriptions(ctx), nil
}

func (g *GoSub) TopicConfig(name string, ctx context.Context) (pubsub.TopicConfig, error) {
	t, err := g.GetTopic(name)
	if err != nil {
		return pubsub.TopicConfig{}, err
	}
	cfg, err := t.Config(ctx)
	if err != nil {
		return pubsub.TopicConfig{}, err
	}
	return cfg, nil
}

func (g *GoSub) UpdateTopicConfig(name string, ctx context.Context, labels map[string]string) (pubsub.TopicConfig, error) {
	t, err := g.GetTopic(name)
	if err != nil {
		return pubsub.TopicConfig{}, err
	}
	cfg, err := t.Update(ctx, pubsub.TopicConfigToUpdate{
		Labels: labels,
	})
	if err != nil {
		return pubsub.TopicConfig{}, err
	}
	return cfg, nil
}


type SubscriberWare func(opts driver.HandlerOptions, next driver.MsgHandler) driver.MsgHandler
type PublisherWare func(serviceName string, next driver.PublishHandler) driver.PublishHandler

type MiddlewareFunctions struct {
	SubscriberWare SubscriberWare
	PublisherWare  PublisherWare
}

func NewMiddlewareFunctions(subscriberWare SubscriberWare, publisherWare PublisherWare) *MiddlewareFunctions {
	return &MiddlewareFunctions{SubscriberWare: subscriberWare, PublisherWare: publisherWare}
}

func (m MiddlewareFunctions) SubscribeInterceptor(opts driver.HandlerOptions, next driver.MsgHandler) driver.MsgHandler {
	return m.SubscriberWare(opts, next)
}

func (m MiddlewareFunctions) PublisherMsgInterceptor(serviceName string, next driver.PublishHandler) driver.PublishHandler {
	return m.PublisherWare(serviceName, next)
}
