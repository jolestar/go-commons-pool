package pool

import (
	"fmt"
	"github.com/jolestar/go-commons-pool/collections"
	"sync"
	"time"
)

// PooledObjectState is PooledObjectState enum const
type PooledObjectState int

const (
	// StateIdle in the queue, not in use. default value.
	StateIdle PooledObjectState = iota
	// StateAllocated in use.
	StateAllocated
	// StateEviction in the queue, currently being tested for possible eviction.
	StateEviction
	// StateEvictionReturnToHead not in the queue, currently being tested for possible eviction. An
	// attempt to borrow the object was made while being tested which removed it
	// from the queue. It should be returned to the head of the queue once
	// eviction testing completes.
	StateEvictionReturnToHead
	// StateInvalid failed maintenance (e.g. eviction test or validation) and will be / has
	// been destroyed
	StateInvalid
	// StateAbandoned Deemed abandoned, to be invalidated.
	StateAbandoned
	// StateReturning Returning to the pool
	StateReturning
)

// TrackedUse allows pooled objects to make information available about when
// and how they were used available to the object pool. The object pool may, but
// is not required, to use this information to make more informed decisions when
// determining the state of a pooled object - for instance whether or not the
// object has been abandoned.
type TrackedUse interface {
	// GetLastUsed Get the last time o object was used in ms.
	GetLastUsed() int64
}

// PooledObject is the wrapper of origin object that is used to track the additional information, such as
// state, for the pooled objects.
type PooledObject struct {
	Object         interface{}
	CreateTime     int64
	LastBorrowTime int64
	LastReturnTime int64
	//init equals CreateTime
	LastUseTime   int64
	state         PooledObjectState
	BorrowedCount int32
	lock          sync.Mutex
}

// NewPooledObject return new init PooledObject
func NewPooledObject(object interface{}) *PooledObject {
	time := currentTimeMillis()
	return &PooledObject{Object: object, state: StateIdle, CreateTime: time, LastUseTime: time, LastBorrowTime: time, LastReturnTime: time}
}

func currentTimeMillis() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

// GetActiveTimeMillis return the time in milliseconds that this object last spent in the the active state
func (o *PooledObject) GetActiveTimeMillis() int64 {
	// Take copies to avoid concurrent issues
	rTime := o.LastReturnTime
	bTime := o.LastBorrowTime

	if rTime > bTime {
		return rTime - bTime
	}
	return currentTimeMillis() - bTime
}

// GetIdleTimeMillis the time in milliseconds that this object last spend in the the idle state
func (o *PooledObject) GetIdleTimeMillis() int64 {
	elapsed := currentTimeMillis() - o.LastReturnTime
	// elapsed may be negative if:
	// - another goroutine updates lastReturnTime during the calculation window
	// - currentTimeMillis() is not monotonic (e.g. system time is set back)
	if elapsed >= 0 {
		return elapsed
	}
	return 0
}

// GetLastUsedTime return an estimate of the last time this object was used.
func (o *PooledObject) GetLastUsedTime() int64 {
	trackedUse, ok := o.Object.(TrackedUse)
	if ok {
		if trackedUse.GetLastUsed() > o.LastUseTime {
			return trackedUse.GetLastUsed()
		}
		return o.LastUseTime
	}
	return o.LastUseTime
}

func (o *PooledObject) doAllocate() bool {
	if o.state == StateIdle {
		o.state = StateAllocated
		o.LastBorrowTime = currentTimeMillis()
		o.LastUseTime = o.LastBorrowTime
		o.BorrowedCount++
		//if (logAbandoned) {
		//borrowedBy = new AbandonedObjectCreatedException();
		//}
		return true
	} else if o.state == StateEviction {
		// TODO Allocate anyway and ignore eviction test
		o.state = StateEvictionReturnToHead
		return false
	}
	// TODO if validating and testOnBorrow == true then pre-allocate for
	// performance
	return false
}

// Allocate this object
func (o *PooledObject) Allocate() bool {
	o.lock.Lock()
	result := o.doAllocate()
	o.lock.Unlock()
	return result
}

func (o *PooledObject) doDeallocate() bool {
	if o.state == StateAllocated ||
		o.state == StateReturning {
		o.state = StateIdle
		o.LastReturnTime = currentTimeMillis()
		//borrowedBy = nil;
		return true
	}
	return false
}

// Deallocate this object
func (o *PooledObject) Deallocate() bool {
	o.lock.Lock()
	result := o.doDeallocate()
	o.lock.Unlock()
	return result
}

// Invalidate this object
func (o *PooledObject) Invalidate() {
	o.lock.Lock()
	o.invalidate()
	o.lock.Unlock()
}

func (o *PooledObject) invalidate() {
	o.state = StateInvalid
}

// GetState return current state of this object
func (o *PooledObject) GetState() PooledObjectState {
	o.lock.Lock()
	defer o.lock.Unlock()
	return o.state
}

// MarkAbandoned mark this object to Abandoned state
func (o *PooledObject) MarkAbandoned() {
	o.lock.Lock()
	o.markAbandoned()
	o.lock.Unlock()
}

func (o *PooledObject) markAbandoned() {
	o.state = StateAbandoned
}

// MarkReturning mark this object to Returning state
func (o *PooledObject) MarkReturning() {
	o.lock.Lock()
	o.markReturning()
	o.lock.Unlock()
}

func (o *PooledObject) markReturning() {
	o.state = StateReturning
}

// StartEvictionTest attempt to place the pooled object in the EVICTION state
func (o *PooledObject) StartEvictionTest() bool {
	o.lock.Lock()
	defer o.lock.Unlock()
	if o.state == StateIdle {
		o.state = StateEviction
		return true
	}

	return false
}

// EndEvictionTest  called to inform the object that the eviction test has ended.
func (o *PooledObject) EndEvictionTest(idleQueue *collections.LinkedBlockingDeque) bool {
	o.lock.Lock()
	defer o.lock.Unlock()
	if o.state == StateEviction {
		o.state = StateIdle
		return true
	} else if o.state == StateEvictionReturnToHead {
		o.state = StateIdle
		if !idleQueue.OfferFirst(o) {
			// TODO - Should never happen
			panic(fmt.Errorf("Should never happen"))
		}
	}

	return false
}
