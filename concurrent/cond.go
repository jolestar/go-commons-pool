package concurrent

import (
	"sync"
	"time"
)

type TimeoutCond struct {
	L          sync.Locker
	signal     chan int
	hasWaiters bool
}

func NewTimeoutCond(l sync.Locker) *TimeoutCond {
	cond := TimeoutCond{L: l, signal: make(chan int, 0)}
	return &cond
}

//wait for signal return remain wait time, and is interrupted
func (this *TimeoutCond) WaitWithTimeout(timeout time.Duration) (time.Duration, bool) {
	this.setHasWaiters(true)
	ch := this.signal
	//wait should unlock mutex,  if not will cause deadlock
	this.L.Unlock()
	defer this.setHasWaiters(false)
	defer this.L.Lock()

	begin := time.Now().UnixNano()
	select {
	case _, ok := <-ch:
		end := time.Now().UnixNano()
		remainTimeout := timeout - time.Duration(end-begin)
		return remainTimeout, !ok
	case <-time.After(timeout):
		return 0, false
	}
}

func (this *TimeoutCond) setHasWaiters(value bool) {
	this.hasWaiters = value
}

//Queries whether any goroutine are waiting on this condition
func (this *TimeoutCond) HasWaiters() bool {
	return this.hasWaiters
}

//wait for signal return waiting is interrupted
func (this *TimeoutCond) Wait() bool {
	this.setHasWaiters(true)
	//copy signal in lock, avoid data race with Interrupt
	ch := this.signal
	this.L.Unlock()
	defer this.setHasWaiters(false)
	defer this.L.Lock()
	_, ok := <-ch
	return !ok
}

// Signal wakes one goroutine waiting on c, if there is any.
func (this *TimeoutCond) Signal() {
	select {
	case this.signal <- 1:
	default:
	}
}

func (this *TimeoutCond) Interrupt() {
	this.L.Lock()
	defer this.L.Unlock()
	close(this.signal)
	this.signal = make(chan int, 0)
}
