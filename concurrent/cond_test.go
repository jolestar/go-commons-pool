package concurrent

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

type LockTestObject struct {
	lock *sync.Mutex
	cond *TimeoutCond
}

func NewLockTestObject() *LockTestObject {
	lock := new(sync.Mutex)
	return &LockTestObject{lock: lock, cond: NewTimeoutCond(lock)}
}

func (o *LockTestObject) lockAndWaitWithTimeout(timeout time.Duration) (time.Duration, bool) {
	o.lock.Lock()
	defer o.lock.Unlock()
	return o.cond.WaitWithTimeout(timeout)
}

func (o *LockTestObject) lockAndWait() bool {
	o.lock.Lock()
	defer o.lock.Unlock()
	fmt.Println("lockAndWait")
	return o.cond.Wait()
}

func (o *LockTestObject) lockAndNotify() {
	o.lock.Lock()
	defer o.lock.Unlock()
	fmt.Println("lockAndNotify")
	o.cond.Signal()
}

func TestTimeoutCondWait(t *testing.T) {
	fmt.Println("TestTimeoutCondWait")
	obj := NewLockTestObject()
	wait := sync.WaitGroup{}
	wait.Add(2)
	go func() {
		obj.lockAndWait()
		wait.Done()
	}()
	time.Sleep(time.Duration(50) * time.Millisecond)
	go func() {
		obj.lockAndNotify()
		wait.Done()
	}()
	wait.Wait()
}

func TestTimeoutCondWaitTimeout(t *testing.T) {
	fmt.Println("TestTimeoutCondWaitTimeout")
	obj := NewLockTestObject()
	wait := sync.WaitGroup{}
	wait.Add(1)
	go func() {
		obj.lockAndWaitWithTimeout(time.Duration(2) * time.Second)
		wait.Done()
	}()
	wait.Wait()
}

func TestTimeoutCondWaitTimeoutNotify(t *testing.T) {
	fmt.Println("TestTimeoutCondWaitTimeoutNotify")
	obj := NewLockTestObject()
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
		obj.lockAndNotify()
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
	fmt.Println("TestTimeoutCondWaitTimeoutRemain")
	obj := NewLockTestObject()
	wait := sync.WaitGroup{}
	wait.Add(2)
	ch := make(chan time.Duration, 1)
	timeout := time.Duration(2000) * time.Millisecond
	go func() {
		remainTimeout, _ := obj.lockAndWaitWithTimeout(timeout)
		ch <- remainTimeout
		wait.Done()
	}()
	sleep(200)
	go func() {
		obj.lockAndNotify()
		wait.Done()
	}()
	wait.Wait()
	remainTimeout := <-ch
	close(ch)
	assert.True(t, remainTimeout < timeout, "expect remainTimeout %v < %v", remainTimeout, timeout)
	assert.True(t, remainTimeout >= time.Duration(200)*time.Millisecond, "expect remainTimeout %v >= 200 millisecond", remainTimeout)
}

func TestTimeoutCondHasWaiters(t *testing.T) {
	fmt.Println("TestTimeoutCondHasWaiters")
	obj := NewLockTestObject()
	wait := sync.WaitGroup{}
	wait.Add(2)
	go func() {
		obj.lockAndWait()
		wait.Done()
	}()
	time.Sleep(time.Duration(50) * time.Millisecond)
	obj.lock.Lock()
	assert.True(t, obj.cond.HasWaiters())
	obj.lock.Unlock()
	go func() {
		obj.lockAndNotify()
		wait.Done()
	}()
	wait.Wait()
	assert.False(t, obj.cond.HasWaiters())
}
func TestInterrupted(t *testing.T) {
	fmt.Println("TestInterrupted")
	obj := NewLockTestObject()
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
	fmt.Println("TestInterruptedWithTimeout")
	obj := NewLockTestObject()
	wait := sync.WaitGroup{}
	count := 5
	wait.Add(5)
	ch := make(chan bool, 5)
	timeout := time.Duration(1000) * time.Millisecond
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
	obj := NewLockTestObject()
	obj.cond.Signal()
}
