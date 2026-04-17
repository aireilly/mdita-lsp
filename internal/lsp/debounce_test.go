package lsp

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestDebouncerCoalesces(t *testing.T) {
	var count atomic.Int32
	d := newDebouncer(50 * time.Millisecond)

	for i := 0; i < 10; i++ {
		d.Schedule("key", func() { count.Add(1) })
	}

	time.Sleep(150 * time.Millisecond)

	if got := count.Load(); got != 1 {
		t.Errorf("expected 1 invocation, got %d", got)
	}
}

func TestDebouncerFlush(t *testing.T) {
	var count atomic.Int32
	d := newDebouncer(5 * time.Second)

	d.Schedule("key", func() { count.Add(1) })
	d.Flush("key")

	if got := count.Load(); got != 1 {
		t.Errorf("expected 1 invocation after flush, got %d", got)
	}
}

func TestDebouncerFlushNoOp(t *testing.T) {
	d := newDebouncer(50 * time.Millisecond)
	d.Flush("nonexistent")
}

func TestDebouncerIndependentKeys(t *testing.T) {
	var countA, countB atomic.Int32
	d := newDebouncer(50 * time.Millisecond)

	d.Schedule("a", func() { countA.Add(1) })
	d.Schedule("b", func() { countB.Add(1) })

	time.Sleep(150 * time.Millisecond)

	if got := countA.Load(); got != 1 {
		t.Errorf("key a: expected 1, got %d", got)
	}
	if got := countB.Load(); got != 1 {
		t.Errorf("key b: expected 1, got %d", got)
	}
}
