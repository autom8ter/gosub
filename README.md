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
