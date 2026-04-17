package lsp

import (
	"sync"
	"time"
)

type pendingAction struct {
	timer *time.Timer
	fn    func()
}

type debouncer struct {
	mu      sync.Mutex
	pending map[string]*pendingAction
	delay   time.Duration
}

func newDebouncer(delay time.Duration) *debouncer {
	return &debouncer{
		pending: make(map[string]*pendingAction),
		delay:   delay,
	}
}

func (d *debouncer) Schedule(key string, fn func()) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if p, ok := d.pending[key]; ok {
		p.timer.Stop()
	}

	pa := &pendingAction{fn: fn}
	pa.timer = time.AfterFunc(d.delay, func() {
		d.mu.Lock()
		delete(d.pending, key)
		d.mu.Unlock()
		fn()
	})
	d.pending[key] = pa
}

func (d *debouncer) Flush(key string) {
	d.mu.Lock()
	p, ok := d.pending[key]
	if ok {
		p.timer.Stop()
		delete(d.pending, key)
	}
	d.mu.Unlock()

	if ok {
		p.fn()
	}
}
