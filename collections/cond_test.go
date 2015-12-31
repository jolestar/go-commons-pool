package collections

import (
	"fmt"
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

func (this *LockTestObject) lockAndWaitWithTimeout(timeout time.Duration) {
	this.lock.Lock()
	defer this.lock.Unlock()
	this.cond.WaitWithTimeout(timeout)
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
	wait.Add(2)
	go func() {
		obj.lockAndWaitWithTimeout(time.Duration(2) * time.Second)
		wait.Done()
	}()
	time.Sleep(time.Duration(50) * time.Millisecond)
	go func() {
		obj.lockAndNotify()
		wait.Done()
	}()
	wait.Wait()
}
