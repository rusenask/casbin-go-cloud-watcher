package watcher

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/casbin/casbin"
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
	defer listener.Close()

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

func TestWithEnforcerNATS(t *testing.T) {
	// Setup nats server
	s := gnatsd.RunDefaultServer()
	defer s.Shutdown()

	natsEndpoint := fmt.Sprintf("nats://localhost:%d", nats.DefaultPort)
	natsSubject := "nats://casbin-policy-updated-subject"

	cannel := make(chan string, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// gocloud.dev connects to NATS server based on env variable
	os.Setenv("NATS_SERVER_URL", natsEndpoint)

	w, err := New(ctx, natsSubject)
	if err != nil {
		t.Fatalf("Failed to create updater, error: %s", err)
	}
	defer w.Close()

	// Initialize the enforcer.
	e := casbin.NewEnforcer("./test_data/model.conf", "./test_data/policy.csv")

	// Set the watcher for the enforcer.
	e.SetWatcher(w)

	// By default, the watcher's callback is automatically set to the
	// enforcer's LoadPolicy() in the SetWatcher() call.
	// We can change it by explicitly setting a callback.
	w.SetUpdateCallback(func(msg string) {
		cannel <- "enforcer"
	})

	// Update the policy to test the effect.
	e.SavePolicy()

	// Validate that listener received message
	select {
	case res := <-cannel:
		if res != "enforcer" {
			t.Fatalf("Got unexpected message :%v", res)
		}
	case <-time.After(time.Second * 10):
		t.Fatal("The enforcer didn't send message in time")
	}
	close(cannel)
}

func TestWithEnforcerMemory(t *testing.T) {

	endpointURL := "mem://topicA"

	cannel := make(chan string, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w, err := New(ctx, endpointURL)
	if err != nil {
		t.Fatalf("Failed to create updater, error: %s", err)
	}
	defer w.Close()

	// Initialize the enforcer.
	e := casbin.NewEnforcer("./test_data/model.conf", "./test_data/policy.csv")

	// Set the watcher for the enforcer.
	e.SetWatcher(w)

	// By default, the watcher's callback is automatically set to the
	// enforcer's LoadPolicy() in the SetWatcher() call.
	// We can change it by explicitly setting a callback.
	w.SetUpdateCallback(func(msg string) {
		cannel <- "enforcer"
	})

	// Update the policy to test the effect.
	e.SavePolicy()

	// Validate that listener received message
	select {
	case res := <-cannel:
		if res != "enforcer" {
			t.Fatalf("Got unexpected message :%v", res)
		}
	case <-time.After(time.Second * 5):
		t.Fatal("The enforcer didn't send message in time")
	}
	close(cannel)
}

func TestWithEnforcerMemoryB(t *testing.T) {

	endpointURL := "mem://topicA"

	cannel := make(chan string, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w, err := New(ctx, endpointURL)
	if err != nil {
		t.Fatalf("Failed to create updater, error: %s", err)
	}
	defer w.Close()

	// Initialize the enforcer.
	e := casbin.NewEnforcer("./test_data/model.conf", "./test_data/policy.csv")

	// Set the watcher for the enforcer.
	e.SetWatcher(w)

	// By default, the watcher's callback is automatically set to the
	// enforcer's LoadPolicy() in the SetWatcher() call.
	// We can change it by explicitly setting a callback.
	w.SetUpdateCallback(func(msg string) {
		cannel <- "enforcer"
	})

	// Update the policy to test the effect.
	e.SavePolicy()

	// Validate that listener received message
	select {
	case res := <-cannel:
		if res != "enforcer" {
			t.Fatalf("Got unexpected message :%v", res)
		}
	case <-time.After(time.Second * 5):
		t.Fatal("The enforcer didn't send message in time")
	}
	close(cannel)
}
