package watcher

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	gnatsd "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
)

func TestNATSWatcher(t *testing.T) {
	// Setup nats server
	s := gnatsd.RunDefaultServer()
	defer s.Shutdown()

	natsEndpoint := fmt.Sprintf("nats://localhost:%d", nats.DefaultPort)
	natsSubject := "nats://casbin-policy-updated-subject"

	updaterCh := make(chan string, 1)
	listenerCh := make(chan string, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// gocloud.dev connects to NATS server based on env variable
	os.Setenv("NATS_SERVER_URL", natsEndpoint)

	// updater represents the Casbin enforcer instance that changes the policy in DB
	// Use the endpoint of nats as parameter.
	updater, err := New(ctx, natsSubject)
	if err != nil {
		t.Fatalf("Failed to create updater, error: %s", err)
	}
	defer updater.Close()
	updater.SetUpdateCallback(func(msg string) {
		updaterCh <- "updater"
	})

	// listener represents any other Casbin enforcer instance that watches the change of policy in DB
	listener, err := New(ctx, natsSubject)
	if err != nil {
		t.Fatalf("Failed to create second listener: %s", err)
	}

	// listener should set a callback that gets called when policy changes
	err = listener.SetUpdateCallback(func(msg string) {
		listenerCh <- "listener"
	})
	if err != nil {
		t.Fatalf("Failed to set listener callback: %s", err)
	}

	// updater changes the policy, and sends the notifications.
	err = updater.Update()
	if err != nil {
		t.Fatalf("The updater failed to send Update: %s", err)
	}

	// Validate that listener received message
	var updaterReceived bool
	var listenerReceived bool
	for {
		select {
		case res := <-listenerCh:
			if res != "listener" {
				t.Fatalf("Message from unknown source: %v", res)
				break
			}
			listenerReceived = true
		case res := <-updaterCh:
			if res != "updater" {
				t.Fatalf("Message from unknown source: %v", res)
				break
			}
			updaterReceived = true
		case <-time.After(time.Second * 10):
			t.Fatal("Updater or listener didn't received message in time")
			break
		}
		if updaterReceived && listenerReceived {
			close(listenerCh)
			close(updaterCh)
			break
		}
	}
}
