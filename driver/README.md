# driver
--
    import "github.com/autom8ter/gosub/driver"


## Usage

#### func  SetClient

```go
func SetClient(cli *Client)
```
SetClient sets the global pubsub client, useful in tests

#### func  Shutdown

```go
func Shutdown()
```

#### func  Subscribe

```go
func Subscribe(s Subscriber)
```
Subscribe starts a run loop with a Subscriber that listens to topics and waits
for a syscall.SIGINT or syscall.SIGTERM

#### type Client

```go
type Client struct {
	ServiceName string
	Provider    Provider
	Middleware  []Middleware
}
```

Client holds a reference to a Provider

#### func (Client) On

```go
func (c Client) On(opts HandlerOptions)
```
On takes HandlerOptions and subscribes to a topic, waiting for a protobuf
message calling the function when a message is received

#### func (*Client) Publish

```go
func (c *Client) Publish(ctx context.Context, topic string, msg interface{}, isJSON bool) error
```
Publish published on the client

#### type Handler

```go
type Handler interface{}
```

Handler is a specific callback used for Subscribe in the format of.. func(ctx
context.Context, obj proto.Message, msg *Msg) error for example, you can
unmarshal a custom type.. func(ctx context.Context, accounts accounts.Account,
msg *Msg) error you can also unmarshal a JSON object by supplying any type of
interface{} func(ctx context.Context, accounts models.SomeJSONAccount, msg *Msg)
error

#### type HandlerOptions

```go
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
```

HandlerOptions defines the options for a subscriber handler

#### type Middleware

```go
type Middleware interface {
	SubscribeInterceptor(opts HandlerOptions, next MsgHandler) MsgHandler
	PublisherMsgInterceptor(serviceName string, next PublishHandler) PublishHandler
}
```

Middleware is an interface to provide subscriber and publisher interceptors

#### type Msg

```go
type Msg struct {
	Message *api.Msg
	Ack     func()
	Nack    func()
}
```

Msg is a representation of a pub sub message

#### type MsgHandler

```go
type MsgHandler func(ctx context.Context, m Msg) error
```

MsgHandler is the internal or raw message handler

#### type NoopProvider

```go
type NoopProvider struct{}
```

NoopProvider is a simple provider that does nothing, for testing, defaults

#### func (NoopProvider) Publish

```go
func (np NoopProvider) Publish(ctx context.Context, topic string, m *Msg) error
```
Publish does nothing

#### func (NoopProvider) Shutdown

```go
func (np NoopProvider) Shutdown()
```
Shutdown shutsdown immediately

#### func (NoopProvider) Subscribe

```go
func (np NoopProvider) Subscribe(opts HandlerOptions, h MsgHandler)
```
Subscribe does nothing

#### type Provider

```go
type Provider interface {
	Publish(ctx context.Context, topic string, m *Msg) error
	Subscribe(opts HandlerOptions, handler MsgHandler)
	Shutdown()
}
```

Provider is generic interface for a pub sub provider

#### type PublishHandler

```go
type PublishHandler func(ctx context.Context, topic string, m *Msg) error
```

PublishHandler wraps a call to publish, for interception

#### type PublishResult

```go
type PublishResult struct {
	Ready chan struct{}
	Err   error
}
```

A PublishResult holds the result from a call to Publish.

#### func  Publish

```go
func Publish(ctx context.Context, topic string, msg proto.Message) *PublishResult
```
Publish is a convenience message which publishes to the current (global)
publisher as protobuf

#### func  PublishJSON

```go
func PublishJSON(ctx context.Context, topic string, obj interface{}) *PublishResult
```
PublishJSON is a convenience message which publishes to the current (global)
publisher as JSON

#### type Subscriber

```go
type Subscriber interface {
	// Setup is a required method that allows the subscriber service to add handlers
	// and perform any setup if required, this is usually called by pubsub upon start
	Setup(*Client)
}
```

Subscriber is a service that listens to events and registers handlers for those
events
