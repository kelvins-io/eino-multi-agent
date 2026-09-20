package harness

import "sync"

type Bus struct {
	mu   sync.RWMutex
	subs map[string]map[chan Event]struct{}
}

func NewBus() *Bus {
	return &Bus{subs: map[string]map[chan Event]struct{}{}}
}

type Event struct {
	ID      uint64 `json:"id"`
	TaskID  string `json:"task_id"`
	Type    string `json:"type"`
	Agent   string `json:"agent,omitempty"`
	Message string `json:"message"`
	Payload string `json:"payload,omitempty"`
}

func (b *Bus) Publish(ev Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.subs[ev.TaskID] {
		select {
		case ch <- ev:
		default:
		}
	}
}

func (b *Bus) Subscribe(taskID string) (<-chan Event, func()) {
	ch := make(chan Event, 64)
	b.mu.Lock()
	if b.subs[taskID] == nil {
		b.subs[taskID] = map[chan Event]struct{}{}
	}
	b.subs[taskID][ch] = struct{}{}
	b.mu.Unlock()
	unsub := func() {
		b.mu.Lock()
		if set, ok := b.subs[taskID]; ok {
			delete(set, ch)
			if len(set) == 0 {
				delete(b.subs, taskID)
			}
		}
		b.mu.Unlock()
		close(ch)
	}
	return ch, unsub
}
