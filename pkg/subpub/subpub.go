package subpub

import (
	"context"
	"errors"
	"sync"
)

type MessageHandler func(msg interface{})

var (
	ErrBusClosed       = errors.New("event bus is closed")
	ErrSubjectNotFound = errors.New("subject was not found")
	ErrNilHandler      = errors.New("handler cannot be nil")
)

type Subscription interface {
	Unsubscribe()
}

type SubPub interface {
	Subscribe(subject string, cb MessageHandler) (Subscription, error)

	Publish(subject string, msg interface{}) error

	Close(ctx context.Context) error
}

type EventBus struct {
	mu     sync.Mutex
	subs   map[string][]*subscription
	wg     sync.WaitGroup
	closed bool
}

func (b *EventBus) Subscribe(subject string, cb MessageHandler) (Subscription, error) {
	if b.closed {
		return nil, ErrBusClosed
	}
	b.mu.Lock()
	defer b.mu.Unlock()

	if cb == nil {
		return nil, ErrNilHandler
	}
	sub := &subscription{
		subject: subject,
		handler: cb,
		bus:     b,
	}

	if _, ok := b.subs[subject]; !ok {
		b.subs[subject] = make([]*subscription, 0)
	}
	b.subs[subject] = append(b.subs[subject], sub)
	return sub, nil
}

func (b *EventBus) Publish(subject string, msg interface{}) error {
	if b.closed {
		return ErrBusClosed
	}
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, ok := b.subs[subject]; !ok {
		return ErrSubjectNotFound
	}
	copySubs := make([]*subscription, len(b.subs[subject]))
	copy(copySubs, b.subs[subject])
	var err error
	for _, s := range copySubs {
		b.wg.Add(1)
		go func(s *subscription) {
			defer b.wg.Done()
			s.handler(msg)
		}(s)
	}
	return err
}

func (b *EventBus) Close(ctx context.Context) error {
	b.closed = true
	done := make(chan bool)
	go func() {
		b.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type subscription struct {
	subject string
	handler MessageHandler
	bus     *EventBus
}

func (sub *subscription) Unsubscribe() {
	sub.bus.mu.Lock()
	defer sub.bus.mu.Unlock()

	if subs, ok := sub.bus.subs[sub.subject]; ok {
		for i, s := range subs {
			if s == sub {
				sub.bus.subs[sub.subject] = append(subs[:i], subs[i+1:]...)
				break
			}
		}
	}
}

func NewSubPub() SubPub {
	bus := &EventBus{
		subs:   make(map[string][]*subscription),
		closed: false,
	}
	return bus
}
