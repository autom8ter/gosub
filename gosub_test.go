package gosub_test

import (
	"context"
	"github.com/autom8ter/api/go/api"
	"github.com/autom8ter/gosub"
	"io/ioutil"
	"os"
	"testing"
	"time"

	uuid "github.com/gofrs/uuid"
	"github.com/golang/protobuf/proto"

	"github.com/stretchr/testify/assert"
)

func TestGooglePublishSubscribe(t *testing.T) {
	sub := "autom8ter_" + api.Util.UUID()

	ps, err := gosub.NewGoSub(os.Getenv("PROJECT_ID"))
	assert.Nil(t, err)
	assert.NotNil(t, ps)

	done := make(chan bool)

	topic := "autom8ter_topic"
	_, err = ps.getTopic(topic)
	assert.Nil(t, err)

	opts := pubsub.HandlerOptions{
		Topic:       topic,
		Name:        sub,
		ServiceName: "test",
	}

	a := test.Account{Name: "Alex"}
	ps.subscribe(opts, func(ctx context.Context, m pubsub.Msg) error {
		var ac test.Account
		proto.Unmarshal(m.Data, &ac)

		assert.Equal(t, ac.Name, a.Name)

		done <- true
		return nil
	}, done)

	// Wait for subscription
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		assert.Fail(t, "Subscription failed after timeout")
	}

	assert.Nil(t, ps.deleteTopic(topic))
}
