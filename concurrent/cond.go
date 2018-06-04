package concurrent

import (
	"context"
	"sync"
	"time"
)

// TimeoutCond is a sync.Cond  improve for support wait timeout.
type TimeoutCond struct {
	L          sync.Locker
	signal     chan int
	hasWaiters bool
}

// NewTimeoutCond return a new TimeoutCond
func NewTimeoutCond(l sync.Locker) *TimeoutCond {
	cond := TimeoutCond{L: l, signal: make(chan int, 0)}
	return &cond
}

// WaitWithContext waits for a signal, or for the context do be done. Returns true if signaled.
func (cond *TimeoutCond) WaitWithContext(ctx context.Context) bool {
	cond.setHasWaiters(true)
	ch := cond.signal
	//wait should unlock mutex,  if not will cause deadlock
	cond.L.Unlock()
	defer cond.setHasWaiters(false)
	defer cond.L.Lock()

	select {
	case _, ok := <-ch:
		return !ok
	case <-ctx.Done():
		return false
	}
}

// WaitWithTimeout wait for signal return remain wait time, and is interrupted
func (cond *TimeoutCond) WaitWithTimeout(timeout time.Duration) (time.Duration, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	begin := time.Now()
	interrupted := cond.WaitWithContext(ctx)
	elapsed := time.Since(begin)
	remainingTimeout := timeout - elapsed

	return remainingTimeout, interrupted
}

func (cond *TimeoutCond) setHasWaiters(value bool) {
	cond.hasWaiters = value
}

// HasWaiters queries whether any goroutine are waiting on this condition
func (cond *TimeoutCond) HasWaiters() bool {
	return cond.hasWaiters
}

// Wait for signal return waiting is interrupted
func (cond *TimeoutCond) Wait() bool {
	cond.setHasWaiters(true)
	//copy signal in lock, avoid data race with Interrupt
	ch := cond.signal
	cond.L.Unlock()
	defer cond.setHasWaiters(false)
	defer cond.L.Lock()
	_, ok := <-ch
	return !ok
}

// Signal wakes one goroutine waiting on c, if there is any.
func (cond *TimeoutCond) Signal() {
	select {
	case cond.signal <- 1:
	default:
	}
}

// Interrupt goroutine wait on this TimeoutCond
func (cond *TimeoutCond) Interrupt() {
	cond.L.Lock()
	defer cond.L.Unlock()
	close(cond.signal)
	cond.signal = make(chan int, 0)
}
