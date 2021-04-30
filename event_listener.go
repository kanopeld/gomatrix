package gomatrix

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
)

type EventListener interface {
	On(eType EventType, callback EventListenerCallback)

	looper
}

type looper interface {
	scanEvents(ctx context.Context) error
	stop()
}

// EventListenerCallback can be used with DefaultSyncer.OnEventType to be informed of incoming events.
type EventListenerCallback func(*Event)

func NewDefaultListener(eventsChan <-chan *Event) *defaultListener {
	dl := &defaultListener{
		listeners: make(map[EventType][]EventListenerCallback),
		stopCh:    make(chan struct{}),
		events:    eventsChan,
	}
	return dl
}

type defaultListener struct {
	listeners      map[EventType][]EventListenerCallback
	listenersRWMut sync.RWMutex
	stopCh         chan struct{}
	events         <-chan *Event
	err            error
}

func (l *defaultListener) On(eType EventType, callback EventListenerCallback) {
	l.listenersRWMut.Lock()
	callsList, ok := l.listeners[eType]
	if !ok {
		callsList = make([]EventListenerCallback, 1)
	}
	callsList = append(callsList, callback)
	l.listeners[eType] = callsList
	l.listenersRWMut.Unlock()
}

func (l *defaultListener) call(e *Event) (err error) {
	if e == nil {
		return
	}

	l.listenersRWMut.RLock()
	callsList, ok := l.listeners[e.Type]
	l.listenersRWMut.Unlock()
	if !ok {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Call listener panicked! panic=%s\n%s", r, debug.Stack())
		}
	}()

	for _, fn := range callsList {
		fn(e)
	}
	return
}

func (l *defaultListener) scanEvents(ctx context.Context) error {
	for {
		select {
		case e, ok := <-l.events:
			if !ok {
				return l.err
			}
			if l.err = l.call(e); l.err != nil {
				return l.err
			}
		case <-l.stopCh:
			return l.err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (l *defaultListener) stop() {
	l.stopCh <- struct{}{}
}
