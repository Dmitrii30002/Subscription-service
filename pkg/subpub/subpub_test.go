package subpub

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestSubscribePublish(t *testing.T) {
	bus := NewSubPub()
	subject := "test"
	var wg sync.WaitGroup
	var received string

	wg.Add(1)
	_, err := bus.Subscribe(subject, func(msg interface{}) {
		defer wg.Done()
		received = msg.(string)
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	testMsg := "hello"
	err = bus.Publish(subject, testMsg)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	wg.Wait()
	if received != testMsg {
		t.Errorf("Expected %q, got %q", testMsg, received)
	}
}

func TestMultipleSubscribers(t *testing.T) {
	bus := NewSubPub()
	subject := "multi"
	count := 5
	var wg sync.WaitGroup
	var received int

	for i := 0; i < count; i++ {
		wg.Add(1)
		_, err := bus.Subscribe(subject, func(msg interface{}) {
			defer wg.Done()
			received++
		})
		if err != nil {
			t.Fatalf("Subscribe failed: %v", err)
		}
	}

	err := bus.Publish(subject, "data")
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	wg.Wait()
	if received != count {
		t.Errorf("Expected %d subscribers, got %d", count, received)
	}
}

func TestUnsubscribe(t *testing.T) {
	bus := NewSubPub()
	subject := "unsub"
	var received bool

	sub, err := bus.Subscribe(subject, func(msg interface{}) {
		received = true
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	sub.Unsubscribe()
	err = bus.Publish(subject, "data")
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	if received {
		t.Error("Handler was called after unsubscribe")
	}
}

func TestPublishToNonexistentSubject(t *testing.T) {
	bus := NewSubPub()
	err := bus.Publish("nonexistent", "data")
	if err == nil {
		t.Error("Expected error when publishing to nonexistent subject")
	}
}

func TestConcurrentAccess(t *testing.T) {
	bus := NewSubPub()
	subject := "concurrent"
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := bus.Subscribe(subject, func(msg interface{}) {})
			if err != nil {
				t.Errorf("Subscribe failed: %v", err)
			}
		}()
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := bus.Publish(subject, "data")
			if err != nil {
				t.Errorf("Publish failed: %v", err)
			}
		}()
	}

	wg.Wait()
}

func TestClose(t *testing.T) {
	bus := NewSubPub()
	subject := "close"
	var handlerFinished bool

	_, err := bus.Subscribe(subject, func(msg interface{}) {
		time.Sleep(100 * time.Millisecond)
		handlerFinished = true
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	err = bus.Publish(subject, "data")
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err = bus.Close(ctx)
	if err == nil {
		t.Error("Expected context cancellation error")
	}

	time.Sleep(time.Second * 1)
	if !handlerFinished {
		t.Error("Handler should complete even after Close")
	}
}

func TestOperationsAfterClose(t *testing.T) {
	bus := NewSubPub()
	ctx := context.Background()
	bus.Close(ctx)

	_, err := bus.Subscribe("test", func(msg interface{}) {})
	if err == nil {
		t.Error("Expected error when subscribing after close")
	}

	err = bus.Publish("test", "data")
	if err == nil {
		t.Error("Expected error when publishing after close")
	}
}

func TestSlowSubscriber(t *testing.T) {
	bus := NewSubPub()
	subject := "slow"
	var fastReceived, slowReceived bool
	var wg sync.WaitGroup

	wg.Add(2)
	_, err := bus.Subscribe(subject, func(msg interface{}) {
		defer wg.Done()
		fastReceived = true
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	_, err = bus.Subscribe(subject, func(msg interface{}) {
		time.Sleep(100 * time.Millisecond)
		defer wg.Done()
		slowReceived = true
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	err = bus.Publish(subject, "data")
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	wg.Wait()
	if !fastReceived || !slowReceived {
		t.Errorf("Fast: %v, Slow: %v", fastReceived, slowReceived)
	}
}
