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

func (this *LockTestObject) lockAndWaitWithTimeout(timeout time.Duration) (time.Duration, bool) {
	this.lock.Lock()
	defer this.lock.Unlock()
	return this.cond.WaitWithTimeout(timeout)
}

func (this *LockTestObject) lockAndWait() {
	this.lock.Lock()
	defer this.lock.Unlock()
	fmt.Println("lockAndWait")
	this.cond.Wait()
}

func (this *LockTestObject) lockAndNotify() {
	this.lock.Lock()
	defer this.lock.Unlock()
	fmt.Println("lockAndNotify")
	this.cond.Signal()
}

func TestTimeoutCondWait(t *testing.T) {
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
	obj := NewLockTestObject()
	wait := sync.WaitGroup{}
	wait.Add(2)
	ch := make(chan time.Duration, 1)
	timeout := 2000
	go func() {
		remainTimeout, _ := obj.lockAndWaitWithTimeout(time.Duration(timeout) * time.Millisecond)
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
	assert.True(t, remainTimeout < time.Duration(timeout)*time.Millisecond)
	assert.True(t, remainTimeout >= time.Duration(200)*time.Millisecond)
}

func TestTimeoutCondHasWaiters(t *testing.T) {
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
