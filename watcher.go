package watcher

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/casbin/casbin/persist"
	"gocloud.dev/pubsub"

	// Import the pubsub driver packages we want to be able to open.
	_ "gocloud.dev/pubsub/awssnssqs"
	_ "gocloud.dev/pubsub/azuresb"
	_ "gocloud.dev/pubsub/gcppubsub"
	_ "gocloud.dev/pubsub/kafkapubsub"
	_ "gocloud.dev/pubsub/natspubsub"
	_ "gocloud.dev/pubsub/rabbitpubsub"
)

var _ persist.Watcher = &Watcher{}

// Errors
var (
	ErrNotConnected = errors.New("pubsub not connected, cannot dispatch update message")
)

type Watcher struct {
	opts         *Options
	callbackFunc func(string)
	connMu       sync.RWMutex
	ctx          context.Context
	topic        *pubsub.Topic
	sub          *pubsub.Subscription
}

type Options struct {
	URL string // gcppubsub://myproject/mytopic
}

// New creates a new watcher  https://gocloud.dev/concepts/urls/
func New(opts *Options) *Watcher {
	return &Watcher{
		opts: opts,
	}
}

// SetUpdateCallback sets the callback function that the watcher will call
// when the policy in DB has been changed by other instances.
// A classic callback is Enforcer.LoadPolicy().
func (w *Watcher) SetUpdateCallback(callbackFunc func(string)) error {
	w.connMu.Lock()
	w.callbackFunc = callbackFunc
	w.connMu.Unlock()
	return nil
}

func (w *Watcher) Connect(ctx context.Context) error {
	w.connMu.Lock()
	defer w.connMu.Unlock()
	w.ctx = ctx
	topic, err := pubsub.OpenTopic(ctx, w.opts.URL)
	if err != nil {
		return err
	}
	w.topic = topic

	return nil
}

func (w *Watcher) subscribeToUpdates(ctx context.Context) error {
	sub, err := pubsub.OpenSubscription(ctx, w.opts.URL)
	if err != nil {
		return fmt.Errorf("failed to open updates subscription, error: %w", err)
	}
	w.sub = sub
	for {
		msg, err := sub.Receive(ctx)
		if err != nil {
			return err
		}
		w.executeCallback(msg)
	}
}

func (w *Watcher) executeCallback(msg *pubsub.Message) {
	w.connMu.RLock()
	defer w.connMu.RUnlock()
	if w.callbackFunc != nil {
		go w.callbackFunc(string(msg.Body))
	}
}

func (w *Watcher) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	w.topic.Shutdown(ctx)
}

// Update calls the update callback of other instances to synchronize their policy.
// It is usually called after changing the policy in DB, like Enforcer.SavePolicy(),
// Enforcer.AddPolicy(), Enforcer.RemovePolicy(), etc.
func (w *Watcher) Update() error {
	w.connMu.RLock()
	defer w.connMu.RUnlock()
	if w.topic == nil {
		return ErrNotConnected
	}
	m := &pubsub.Message{Body: []byte("")}
	return w.topic.Send(w.ctx, m)
}
