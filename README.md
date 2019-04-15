# gosub
--
    import "github.com/autom8ter/gosub"


## Usage

#### type GoSub

```go
type GoSub struct {
}
```

GoSub provides google cloud pubsub

#### func  NewGoSub

```go
func NewGoSub(projectID string) (*GoSub, error)
```
NewGoSub creates a new GoSub instace for a project

#### func (*GoSub) DeleteTopic

```go
func (g *GoSub) DeleteTopic(name string) error
```

#### func (*GoSub) GetTopic

```go
func (g *GoSub) GetTopic(name string) (*pubsub.Topic, error)
```

#### func (*GoSub) Publish

```go
func (g *GoSub) Publish(ctx context.Context, topic string, m *driver.Msg) error
```
Publish implements Publish

#### func (*GoSub) Shutdown

```go
func (g *GoSub) Shutdown()
```
Shutdown shuts down all subscribers gracefully

#### func (*GoSub) Subscribe

```go
func (g *GoSub) Subscribe(opts driver.HandlerOptions, h driver.MsgHandler)
```
Subscribe implements Subscribe

#### func (*GoSub) TopicConfig

```go
func (g *GoSub) TopicConfig(name string, ctx context.Context) (pubsub.TopicConfig, error)
```

#### func (*GoSub) TopicExists

```go
func (g *GoSub) TopicExists(name string) (bool, error)
```

#### func (*GoSub) TopicIAM

```go
func (g *GoSub) TopicIAM(name string) (*iam.Handle, error)
```

#### func (*GoSub) TopicSubscriptions

```go
func (g *GoSub) TopicSubscriptions(name string, ctx context.Context) (*pubsub.SubscriptionIterator, error)
```

#### func (*GoSub) UpdateTopicConfig

```go
func (g *GoSub) UpdateTopicConfig(name string, ctx context.Context, labels map[string]string) (pubsub.TopicConfig, error)
```

#### type MiddlewareFunctions

```go
type MiddlewareFunctions struct {
	SubscriberWare SubscriberWare
	PublisherWare  PublisherWare
}
```


#### func  NewMiddlewareFunctions

```go
func NewMiddlewareFunctions(subscriberWare SubscriberWare, publisherWare PublisherWare) *MiddlewareFunctions
```

#### func (MiddlewareFunctions) PublisherMsgInterceptor

```go
func (m MiddlewareFunctions) PublisherMsgInterceptor(serviceName string, next driver.PublishHandler) driver.PublishHandler
```

#### func (MiddlewareFunctions) SubscribeInterceptor

```go
func (m MiddlewareFunctions) SubscribeInterceptor(opts driver.HandlerOptions, next driver.MsgHandler) driver.MsgHandler
```

#### type PublisherWare

```go
type PublisherWare func(serviceName string, next driver.PublishHandler) driver.PublishHandler
```


#### type SubscriberWare

```go
type SubscriberWare func(opts driver.HandlerOptions, next driver.MsgHandler) driver.MsgHandler
```
