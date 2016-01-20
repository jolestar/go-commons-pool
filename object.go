package pool

import (
	"fmt"
	"github.com/jolestar/go-commons-pool/collections"
	"sync"
	"time"
)

type PooledObjectState int

const (
	IDLE PooledObjectState = iota
	ALLOCATED
	EVICTION
	EVICTION_RETURN_TO_HEAD
	VALIDATION
	VALIDATION_PREALLOCATED
	VALIDATION_RETURN_TO_HEAD
	INVALID
	ABANDONED
	RETURNING
)

type TrackedUse interface {
	//Get the last time o object was used in ms.
	GetLastUsed() int64
}

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

func NewPooledObject(object interface{}) *PooledObject {
	time := currentTimeMillis()
	return &PooledObject{Object: object, state: IDLE, CreateTime: time, LastUseTime: time, LastBorrowTime: time, LastReturnTime: time}
}

func currentTimeMillis() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func (o *PooledObject) GetActiveTimeMillis() int64 {
	// Take copies to avoid concurrent issues
	rTime := o.LastReturnTime
	bTime := o.LastBorrowTime

	if rTime > bTime {
		return rTime - bTime
	}
	return currentTimeMillis() - bTime
}

func (o *PooledObject) GetIdleTimeMillis() int64 {
	elapsed := currentTimeMillis() - o.LastReturnTime
	// elapsed may be negative if:
	// - another goroutine updates lastReturnTime during the calculation window
	// - currentTimeMillis() is not monotonic (e.g. system time is set back)
	if elapsed >= 0 {
		return elapsed
	} else {
		return 0
	}
}

func (o *PooledObject) GetLastUsedTime() int64 {
	trackedUse, ok := o.Object.(TrackedUse)
	if ok {
		if trackedUse.GetLastUsed() > o.LastUseTime {
			return trackedUse.GetLastUsed()
		} else {
			return o.LastUseTime
		}
	}
	return o.LastUseTime
}

func (o *PooledObject) doAllocate() bool {
	if o.state == IDLE {
		o.state = ALLOCATED
		o.LastBorrowTime = currentTimeMillis()
		o.LastUseTime = o.LastBorrowTime
		o.BorrowedCount++
		//if (logAbandoned) {
		//borrowedBy = new AbandonedObjectCreatedException();
		//}
		return true
	} else if o.state == EVICTION {
		// TODO Allocate anyway and ignore eviction test
		o.state = EVICTION_RETURN_TO_HEAD
		return false
	}
	// TODO if validating and testOnBorrow == true then pre-allocate for
	// performance
	return false
}

func (o *PooledObject) Allocate() bool {
	o.lock.Lock()
	result := o.doAllocate()
	o.lock.Unlock()
	return result
}

func (o *PooledObject) doDeallocate() bool {
	if o.state == ALLOCATED ||
		o.state == RETURNING {
		o.state = IDLE
		o.LastReturnTime = currentTimeMillis()
		//borrowedBy = nil;
		return true
	}
	return false
}

func (o *PooledObject) Deallocate() bool {
	o.lock.Lock()
	result := o.doDeallocate()
	o.lock.Unlock()
	return result
}

func (o *PooledObject) Invalidate() {
	o.lock.Lock()
	o.invalidate()
	o.lock.Unlock()
}

func (o *PooledObject) invalidate() {
	o.state = INVALID
}

func (o *PooledObject) GetState() PooledObjectState {
	o.lock.Lock()
	defer o.lock.Unlock()
	return o.state
}

func (o *PooledObject) MarkAbandoned() {
	o.lock.Lock()
	o.markAbandoned()
	o.lock.Unlock()
}

func (o *PooledObject) markAbandoned() {
	o.state = ABANDONED
}

func (o *PooledObject) MarkReturning() {
	o.lock.Lock()
	o.markReturning()
	o.lock.Unlock()
}

func (o *PooledObject) markReturning() {
	o.state = RETURNING
}

func (o *PooledObject) StartEvictionTest() bool {
	o.lock.Lock()
	defer o.lock.Unlock()
	if o.state == IDLE {
		o.state = EVICTION
		return true
	}

	return false
}

func (o *PooledObject) EndEvictionTest(idleQueue *collections.LinkedBlockingDeque) bool {
	o.lock.Lock()
	defer o.lock.Unlock()
	if o.state == EVICTION {
		o.state = IDLE
		return true
	} else if o.state == EVICTION_RETURN_TO_HEAD {
		o.state = IDLE
		if !idleQueue.OfferFirst(o) {
			// TODO - Should never happen
			panic(fmt.Errorf("Should never happen"))
		}
	}

	return false
}
