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
	/**
	Get the last time this object was used in ms.
	*/
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

func (this *PooledObject) GetActiveTimeMillis() int64 {
	// Take copies to avoid threading issues
	rTime := this.LastReturnTime
	bTime := this.LastBorrowTime

	if rTime > bTime {
		return rTime - bTime
	}
	return currentTimeMillis() - bTime
}

func (this *PooledObject) GetIdleTimeMillis() int64 {
	elapsed := currentTimeMillis() - this.LastReturnTime
	// elapsed may be negative if:
	// - another thread updates lastReturnTime during the calculation window
	// - currentTimeMillis() is not monotonic (e.g. system time is set back)
	if elapsed >= 0 {
		return elapsed
	} else {
		return 0
	}
}

func (this *PooledObject) GetLastUsedTime() int64 {
	trackedUse, ok := this.Object.(TrackedUse)
	if ok {
		if trackedUse.GetLastUsed() > this.LastUseTime {
			return trackedUse.GetLastUsed()
		} else {
			return this.LastUseTime
		}
	}
	return this.LastUseTime
}

func (this *PooledObject) doAllocate() bool {
	if this.state == IDLE {
		this.state = ALLOCATED
		this.LastBorrowTime = currentTimeMillis()
		this.LastUseTime = this.LastBorrowTime
		this.BorrowedCount++
		//if (logAbandoned) {
		//borrowedBy = new AbandonedObjectCreatedException();
		//}
		return true
	} else if this.state == EVICTION {
		// TODO Allocate anyway and ignore eviction test
		this.state = EVICTION_RETURN_TO_HEAD
		return false
	}
	// TODO if validating and testOnBorrow == true then pre-allocate for
	// performance
	return false
}

func (this *PooledObject) Allocate() bool {
	this.lock.Lock()
	result := this.doAllocate()
	this.lock.Unlock()
	return result
}

func (this *PooledObject) doDeallocate() bool {
	if this.state == ALLOCATED ||
		this.state == RETURNING {
		this.state = IDLE
		this.LastReturnTime = currentTimeMillis()
		//borrowedBy = nil;
		return true
	}
	return false
}

func (this *PooledObject) Deallocate() bool {
	this.lock.Lock()
	result := this.doDeallocate()
	this.lock.Unlock()
	return result
}

func (this *PooledObject) Invalidate() {
	this.lock.Lock()
	this.invalidate()
	this.lock.Unlock()
}

func (this *PooledObject) invalidate() {
	this.state = INVALID
}

func (this *PooledObject) Use() {
	this.LastUseTime = currentTimeMillis()
	//usedBy = new Exception("The last code to use this object was:");
}

func (this *PooledObject) GetState() PooledObjectState {
	this.lock.Lock()
	result := this.state
	this.lock.Unlock()
	return result
}

func (this *PooledObject) MarkAbandoned() {
	this.lock.Lock()
	this.markAbandoned()
	this.lock.Unlock()
}

func (this *PooledObject) markAbandoned() {
	this.state = ABANDONED
}

func (this *PooledObject) MarkReturning() {
	this.lock.Lock()
	this.markReturning()
	this.lock.Unlock()
}

func (this *PooledObject) markReturning() {
	this.state = RETURNING
}

func (this *PooledObject) StartEvictionTest() bool {
	this.lock.Lock()
	defer this.lock.Unlock()
	if this.state == IDLE {
		this.state = EVICTION
		return true
	}

	return false
}

func (this *PooledObject) EndEvictionTest(idleQueue *collections.LinkedBlockingDeque) bool {
	this.lock.Lock()
	defer this.lock.Unlock()
	if this.state == EVICTION {
		this.state = IDLE
		return true
	} else if this.state == EVICTION_RETURN_TO_HEAD {
		this.state = IDLE
		if !idleQueue.OfferFirst(this) {
			// TODO - Should never happen
			panic(fmt.Errorf("Should never happen"))
		}
	}

	return false
}
