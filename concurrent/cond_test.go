package concurrent

import (
	"math"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

type LockTestObject struct {
	t    *testing.T
	lock *sync.Mutex
	cond *TimeoutCond
}

func NewLockTestObject(t *testing.T) *LockTestObject {
	lock := new(sync.Mutex)
	return &LockTestObject{t: t, lock: lock, cond: NewTimeoutCond(lock)}
}

func (o *LockTestObject) lockAndWaitWithTimeout(timeout time.Duration) (time.Duration, bool) {
	o.lock.Lock()
	defer o.lock.Unlock()
	return o.cond.WaitWithTimeout(timeout)
}

func (o *LockTestObject) lockAndWait() bool {
	o.lock.Lock()
	defer o.lock.Unlock()
	o.t.Log("lockAndWait")
	return o.cond.Wait()
}

func (o *LockTestObject) lockAndSignal() {
	o.lock.Lock()
	defer o.lock.Unlock()
	o.t.Log("lockAndNotify")
	o.cond.Signal()
}

func (o *LockTestObject) hasWaiters() bool {
	return o.cond.HasWaiters()
}

func TestTimeoutCondWait(t *testing.T) {
	t.Parallel()

	t.Log("TestTimeoutCondWait")
	obj := NewLockTestObject(t)
	wait := sync.WaitGroup{}
	wait.Add(2)
	go func() {
		obj.lockAndWait()
		wait.Done()
	}()
	time.Sleep(50 * time.Millisecond)
	go func() {
		obj.lockAndSignal()
		wait.Done()
	}()
	wait.Wait()
}

func TestTimeoutCondWaitTimeout(t *testing.T) {
	t.Parallel()

	t.Log("TestTimeoutCondWaitTimeout")
	obj := NewLockTestObject(t)
	wait := sync.WaitGroup{}
	wait.Add(1)
	go func() {
		obj.lockAndWaitWithTimeout(2 * time.Second)
		wait.Done()
	}()
	wait.Wait()
}

func TestTimeoutCondWaitTimeoutNotify(t *testing.T) {
	t.Parallel()

	t.Log("TestTimeoutCondWaitTimeoutNotify")
	obj := NewLockTestObject(t)
	wait := sync.WaitGroup{}
	wait.Add(2)
	ch := make(chan int, 1)
	timeout := 2000
	go func() {
		begin := currentTimeMillis()
		obj.lockAndWaitWithTimeout(time.Duration(timeout) * time.Millisecond)
		end := currentTimeMillis()
		ch <- int((end - begin))
		wait.Done()
	}()
	sleep(200)
	go func() {
		obj.lockAndSignal()
		wait.Done()
	}()
	wait.Wait()
	time := <-ch
	close(ch)
	assert.True(t, time < timeout)
	assert.True(t, time >= 200)
}

func sleep(millisecond int) {
	time.Sleep(time.Duration(millisecond) * time.Millisecond)
}

func currentTimeMillis() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func TestTimeoutCondWaitTimeoutRemain(t *testing.T) {
	t.Parallel()

	t.Log("TestTimeoutCondWaitTimeoutRemain")
	obj := NewLockTestObject(t)
	wait := sync.WaitGroup{}
	wait.Add(2)
	ch := make(chan time.Duration, 1)
	timeout := 2000 * time.Millisecond
	go func() {
		remainTimeout, _ := obj.lockAndWaitWithTimeout(timeout)
		ch <- remainTimeout
		wait.Done()
	}()
	sleep(200)
	go func() {
		obj.lockAndSignal()
		wait.Done()
	}()
	wait.Wait()
	remainTimeout := <-ch
	close(ch)
	assert.True(t, remainTimeout < timeout, "expect remainTimeout %v < %v", remainTimeout, timeout)
	assert.True(t, remainTimeout >= 200*time.Millisecond, "expect remainTimeout %v >= 200 millisecond", remainTimeout)
}

func TestTimeoutCondHasWaiters(t *testing.T) {
	t.Parallel()

	t.Log("TestTimeoutCondHasWaiters")
	obj := NewLockTestObject(t)
	waitersCount := 2
	ch := make(chan struct{}, waitersCount)
	for i := 0; i < 2; i++ {
		go func() {
			obj.lockAndWait()
			ch <- struct{}{}
		}()
	}
	time.Sleep(50 * time.Millisecond)
	assert.True(t, obj.hasWaiters(), "Should have waiters")

	obj.lockAndSignal()
	<-ch
	assert.True(t, obj.hasWaiters(), "Should still have waiters")

	obj.lockAndSignal()
	<-ch
	assert.False(t, obj.hasWaiters(), "Should no longer have waiters")
}

func TestTooManyWaiters(t *testing.T) {
	t.Parallel()

	obj := NewLockTestObject(t)
	obj.cond.hasWaiters = math.MaxUint64

	require.Panics(t, func() { obj.lockAndWait() })
}

func TestRemoveWaiterUsedIncorrectly(t *testing.T) {
	t.Parallel()

	cond := NewTimeoutCond(&sync.Mutex{})
	require.Panics(t, cond.removeWaiter)
}

func TestInterrupted(t *testing.T) {
	t.Parallel()

	t.Log("TestInterrupted")
	obj := NewLockTestObject(t)
	wait := sync.WaitGroup{}
	count := 5
	wait.Add(5)
	ch := make(chan bool, 5)
	for i := 0; i < count; i++ {
		go func() {
			ch <- obj.lockAndWait()
			wait.Done()
		}()
	}
	sleep(100)
	go func() { obj.cond.Interrupt() }()
	wait.Wait()
	for i := 0; i < count; i++ {
		b := <-ch
		assert.True(t, b, "expect %v interrupted bug get false", i)
	}
}

func TestInterruptedWithTimeout(t *testing.T) {
	t.Parallel()

	t.Log("TestInterruptedWithTimeout")
	obj := NewLockTestObject(t)
	wait := sync.WaitGroup{}
	count := 5
	wait.Add(5)
	ch := make(chan bool, 5)
	timeout := 1000 * time.Millisecond
	for i := 0; i < count; i++ {
		go func() {
			_, interrupted := obj.lockAndWaitWithTimeout(timeout)
			ch <- interrupted
			wait.Done()
		}()
	}
	sleep(100)
	go func() { obj.cond.Interrupt() }()
	wait.Wait()
	for i := 0; i < count; i++ {
		b := <-ch
		assert.True(t, b, "expect %v interrupted bug get false", i)
	}
}

func TestSignalNoWait(t *testing.T) {
	t.Parallel()

	obj := NewLockTestObject(t)
	obj.cond.Signal()
}
