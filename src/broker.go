package main

import (
	"sync"
)

type brokerMap struct {
	m map[string]map[string]chan int8
	*sync.RWMutex
}

type EventBroker struct {
	Subscribe   chan session
	Unsubscribe chan session
	recipients  brokerMap
}

func NewEventBroker() EventBroker {
	return EventBroker{
		make(chan session),
		make(chan session),
		brokerMap{make(map[string]map[string]chan int8), &sync.RWMutex{}},
	}
}

func (brk *EventBroker) Init() {
	go func() {
		for {
			select {
			case s := <-brk.Subscribe:
				brk.recipients.Lock()
				if m, ok := brk.recipients.m[s.user]; ok {
					m[s.cookie.Value] = s.clipEvtCh
				} else {
					brk.recipients.m[s.user] = map[string]chan int8{s.cookie.Value: s.clipEvtCh}
				}
				brk.recipients.Unlock()
			case s := <-brk.Unsubscribe:
				brk.recipients.Lock()
				if m, ok := brk.recipients.m[s.user]; ok {
					delete(m, s.cookie.Value)
				}
				brk.recipients.Unlock()
			}
		}
	}()
}

func (brk *EventBroker) Publish(receiver string, value int8) {
	brk.recipients.RLock()
	defer brk.recipients.RUnlock()
	for _, ch := range brk.recipients.m[receiver] {
		ch <- value
	}
}
