package driver

import (
	"context"
	"encoding/json"
	"github.com/autom8ter/api/go/api"
	"github.com/golang/protobuf/proto"
	"reflect"
	"time"
)

var wait = make(chan bool)

// Subscribe starts a run loop with a Subscriber that listens to topics and
// waits for a syscall.SIGINT or syscall.SIGTERM
func Subscribe(s Subscriber) {
	s.Setup(client)
	<-wait
}

func Shutdown() {
	wait <- true
	client.Provider.Shutdown()
}

// HandlerOptions defines the options for a subscriber handler
type HandlerOptions struct {
	// The topic to subscribe to
	Topic string
	// The name of this subscriber/function
	Name string
	// The name of this subscriber/function's service
	ServiceName string
	// The function to invoke
	Handler Handler
	// A message deadline/timeout
	Deadline time.Duration
	// Concurrency sets the maximum number of msgs to be run concurrently
	// default: 20
	Concurrency int
	// Auto Ack the message automatically if return err == nil
	AutoAck bool
	// Decode JSON objects from pubsub instead of protobuf
	JSON bool
}

// On takes HandlerOptions and subscribes to a topic, waiting for a protobuf message
// calling the function when a message is received
func (c Client) On(opts HandlerOptions) {
	if opts.Topic == "" {
		panic("pubsub: topic must be set")
	}

	if opts.Name == "" {
		panic("pubsub: name must be set")
	}

	if opts.ServiceName == "" {
		opts.ServiceName = c.ServiceName
	}

	if opts.Handler == nil {
		panic("pubsub: handler cannot be nil")
	}

	// Set some default options
	if opts.Deadline == 0 {
		opts.Deadline = 10 * time.Second
	}

	// Set some default concurrency
	if opts.Concurrency == 0 {
		opts.Concurrency = 20
	}

	// Reflection is slow, but this is done only once on subscriber setup
	hndlr := reflect.TypeOf(opts.Handler)
	if hndlr.Kind() != reflect.Func {
		panic("pubsub: handler needs to be a func")
	}

	if hndlr.NumIn() != 3 {
		panic(`pubsub: handler should be of format
		func(ctx context.Context, obj *proto.Message, msg *Msg) error
		but didn't receive enough args`)
	}

	if hndlr.In(0) != reflect.TypeOf((*context.Context)(nil)).Elem() {
		panic(`pubsub: handler should be of format
		func(ctx context.Context, obj *proto.Message, msg *Msg) error
		but first arg was not context.Context`)
	}

	if !opts.JSON {
		if !hndlr.In(1).Implements(reflect.TypeOf((*proto.Message)(nil)).Elem()) {
			panic(`pubsub: handler should be of format
		func(ctx context.Context, obj *proto.Message, msg *Msg) error
		but second arg does not implement proto.Message interface`)
		}
	}

	if hndlr.In(2) != reflect.TypeOf(&Msg{}) {
		panic(`pubsub: handler should be of format
		func(ctx context.Context, obj *proto.Message, msg *Msg) error
		but third arg was not pubsub.Msg`)
	}

	if !hndlr.Out(0).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		panic(`pubsub: handler should be of format
		func(ctx context.Context, obj *proto.Message, msg *Msg) error
		but output type is not error`)
	}

	fn := reflect.ValueOf(opts.Handler)

	cb := func(ctx context.Context, m Msg) error {
		var err error
		obj := reflect.New(hndlr.In(1).Elem()).Interface()
		if opts.JSON {
			err = json.Unmarshal(m.Data, obj)
		} else {
			err = proto.Unmarshal(m.Data, obj.(proto.Message))
		}

		if err != nil {
			return api.Util.WrapErr(err, "pubsub: could not unmarshal message")
		}

		rtrn := fn.Call([]reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(obj),
			reflect.ValueOf(&m),
		})
		if len(rtrn) == 0 {
			return nil
		}

		erri := rtrn[0].Interface()
		if erri != nil {
			err = erri.(error)
		}

		return err
	}

	mw := chainSubscriberMiddleware(c.Middleware...)
	c.Provider.Subscribe(opts, mw(opts, cb))
}

func chainSubscriberMiddleware(mw ...Middleware) func(opts HandlerOptions, next MsgHandler) MsgHandler {
	return func(opts HandlerOptions, final MsgHandler) MsgHandler {
		return func(ctx context.Context, m Msg) error {
			last := final
			for i := len(mw) - 1; i >= 0; i-- {
				last = mw[i].SubscribeInterceptor(opts, last)
			}
			return last(ctx, m)
		}
	}
}

// Publish published on the client
func (c *Client) Publish(ctx context.Context, topic string, msg interface{}, isJSON bool) error {
	var b []byte
	var err error
	if isJSON {
		b, err = json.Marshal(msg)
	} else {
		b, err = proto.Marshal(msg.(proto.Message))
	}

	if err != nil {
		return err
	}

	m := &Msg{Data: b}

	mw := chainPublisherMiddleware(c.Middleware...)
	return mw(c.ServiceName, func(ctx context.Context, topic string, m *Msg) error {
		return c.Provider.Publish(ctx, topic, m)
	})(ctx, topic, m)
}

// A PublishResult holds the result from a call to Publish.
type PublishResult struct {
	Ready chan struct{}
	Err   error
}

// Publish is a convenience message which publishes to the
// current (global) publisher as protobuf
func Publish(ctx context.Context, topic string, msg proto.Message) *PublishResult {
	pr := &PublishResult{Ready: make(chan struct{})}
	go func() {
		err := client.Publish(ctx, topic, msg, false)
		pr.Err = err
		close(pr.Ready)
	}()
	return pr
}

// PublishJSON is a convenience message which publishes to the
// current (global) publisher as JSON
func PublishJSON(ctx context.Context, topic string, obj interface{}) *PublishResult {
	pr := &PublishResult{Ready: make(chan struct{})}
	go func() {
		err := client.Publish(ctx, topic, obj, true)
		pr.Err = err
		close(pr.Ready)
	}()
	return pr
}

func chainPublisherMiddleware(mw ...Middleware) func(serviceName string, next PublishHandler) PublishHandler {
	return func(serviceName string, final PublishHandler) PublishHandler {
		return func(ctx context.Context, topic string, m *Msg) error {
			last := final
			for i := len(mw) - 1; i >= 0; i-- {
				last = mw[i].PublisherMsgInterceptor(serviceName, last)
			}
			return last(ctx, topic, m)
		}
	}
}

var (
	client = &Client{Provider: NoopProvider{}}
)

// Client holds a reference to a Provider
type Client struct {
	ServiceName string
	Provider    Provider
	Middleware  []Middleware
}

// SetClient sets the global pubsub client, useful in tests
func SetClient(cli *Client) {
	client = cli
}

// Provider is generic interface for a pub sub provider
type Provider interface {
	Publish(ctx context.Context, topic string, m *Msg) error
	Subscribe(opts HandlerOptions, handler MsgHandler)
	Shutdown()
}

// Subscriber is a service that listens to events and registers handlers
// for those events
type Subscriber interface {
	// Setup is a required method that allows the subscriber service to add handlers
	// and perform any setup if required, this is usually called by pubsub upon start
	Setup(*Client)
}

// Msg is a representation of a pub sub message
type Msg struct {
	ID          string
	Metadata    map[string]string
	Data        []byte
	PublishTime *time.Time

	Ack  func()
	Nack func()
}

// Handler is a specific callback used for Subscribe in the format of..
// func(ctx context.Context, obj proto.Message, msg *Msg) error
// for example, you can unmarshal a custom type..
// func(ctx context.Context, accounts accounts.Account, msg *Msg) error
// you can also unmarshal a JSON object by supplying any type of interface{}
// func(ctx context.Context, accounts models.SomeJSONAccount, msg *Msg) error
type Handler interface{}

// MsgHandler is the internal or raw message handler
type MsgHandler func(ctx context.Context, m Msg) error

// PublishHandler wraps a call to publish, for interception
type PublishHandler func(ctx context.Context, topic string, m *Msg) error

// Middleware is an interface to provide subscriber and publisher interceptors
type Middleware interface {
	SubscribeInterceptor(opts HandlerOptions, next MsgHandler) MsgHandler
	PublisherMsgInterceptor(serviceName string, next PublishHandler) PublishHandler
}
