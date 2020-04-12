package watcher

import (
	"context"
	"errors"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/casbin/casbin/persist"
	"gocloud.dev/pubsub"

	// Import the pubsub driver packages we want to be able to open.
	_ "gocloud.dev/pubsub/awssnssqs"
	_ "gocloud.dev/pubsub/azuresb"
	_ "gocloud.dev/pubsub/gcppubsub"
	_ "gocloud.dev/pubsub/kafkapubsub"
	_ "gocloud.dev/pubsub/mempubsub"
	_ "gocloud.dev/pubsub/natspubsub"
	_ "gocloud.dev/pubsub/rabbitpubsub"
)

// check interface compatibility
var _ persist.Watcher = &Watcher{}

// Errors
var (
	ErrNotConnected = errors.New("pubsub not connected, cannot dispatch update message")
)

// Watcher implements Casbin updates watcher to synchronize policy changes
// between the nodes
type Watcher struct {
	url          string
	callbackFunc func(string)
	connMu       *sync.RWMutex
	ctx          context.Context
	topic        *pubsub.Topic
	sub          *pubsub.Subscription
}

// New creates a new watcher  https://gocloud.dev/concepts/urls/
// gcppubsub://myproject/mytopic
func New(ctx context.Context, url string) (*Watcher, error) {
	w := &Watcher{
		url:    url,
		connMu: &sync.RWMutex{},
	}

	runtime.SetFinalizer(w, finalizer)

	err := w.initializeConnections(ctx)

	return w, err
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

func (w *Watcher) initializeConnections(ctx context.Context) error {
	w.connMu.Lock()
	defer w.connMu.Unlock()
	w.ctx = ctx
	topic, err := pubsub.OpenTopic(ctx, w.url)
	if err != nil {
		return err
	}
	w.topic = topic

	return w.subscribeToUpdates(ctx)
}

func (w *Watcher) subscribeToUpdates(ctx context.Context) error {
	sub, err := pubsub.OpenSubscription(ctx, w.url)
	if err != nil {
		return fmt.Errorf("failed to open updates subscription, error: %w", err)
	}
	w.sub = sub
	go func() {
		for {
			msg, err := sub.Receive(ctx)
			if err != nil {
				if ctx.Err() == context.Canceled {
					// nothing to do
					return
				}
				log.Printf("Error while receiving an update message: %s\n", err)
				return
			}
			w.executeCallback(msg)

			msg.Ack()
		}
	}()
	return nil
}

func (w *Watcher) executeCallback(msg *pubsub.Message) {
	w.connMu.RLock()
	defer w.connMu.RUnlock()
	if w.callbackFunc != nil {
		go w.callbackFunc(string(msg.Body))
	}
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

// Close stops and releases the watcher, the callback function will not be called any more.
func (w *Watcher) Close() {
	finalizer(w)
}

func finalizer(w *Watcher) {
	w.connMu.Lock()
	defer w.connMu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if w.topic != nil {
		err := w.topic.Shutdown(ctx)
		if err != nil {
			log.Printf("Topic shutdown failed, error: %s\n", err)
		}
		w.topic = nil
	}

	if w.sub != nil {
		err := w.sub.Shutdown(ctx)
		if err != nil {
			log.Printf("Subscription shutdown failed, error: %s\n", err)
		}
		w.sub = nil
	}

	w.callbackFunc = nil
}
