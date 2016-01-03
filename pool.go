package pool

import (
	"errors"
	"fmt"
	"github.com/jolestar/go-commons-pool/collections"
	"github.com/jolestar/go-commons-pool/concurrent"
	"math"
	"sync"
	"time"
)

const (
	debug_pool = false
)

type baseErr struct {
	msg string
}

func (this *baseErr) Error() string {
	return this.msg
}

type IllegalStatusErr struct {
	baseErr
}

func NewIllegalStatusErr(msg string) *IllegalStatusErr {
	return &IllegalStatusErr{baseErr{msg}}
}

type NoSuchElementErr struct {
	baseErr
}

func NewNoSuchElementErr(msg string) *NoSuchElementErr {
	return &NoSuchElementErr{baseErr{msg}}
}

type ObjectPool struct {
	AbandonedConfig         *AbandonedConfig
	Config                  *ObjectPoolConfig
	closed                  bool
	closeLock               sync.Mutex
	evictionLock            sync.Mutex
	idleObjects             *collections.LinkedBlockDeque
	allObjects              *collections.SyncIdentityMap
	factory                 PooledObjectFactory
	createCount             concurrent.AtomicInteger
	destroyedByEvictorCount concurrent.AtomicInteger
	destroyedCount          concurrent.AtomicInteger
	evictor                 *time.Ticker
	evictionIterator        collections.Iterator
}

func NewObjectPool(factory PooledObjectFactory, config *ObjectPoolConfig) *ObjectPool {
	pool := ObjectPool{factory: factory, Config: config,
		idleObjects:             collections.NewDeque(math.MaxInt32),
		allObjects:              collections.NewSyncMap(),
		createCount:             concurrent.AtomicInteger(0),
		destroyedByEvictorCount: concurrent.AtomicInteger(0),
		destroyedCount:          concurrent.AtomicInteger(0)}
	pool.StartEvictor()
	return &pool
}

func NewObjectPoolWithDefaultConfig(factory PooledObjectFactory) *ObjectPool {
	return NewObjectPool(factory, NewDefaultPoolConfig())
}

// Create an object using the PooledObjectFactory factory, passivate it, and then place it in
// the idle object pool. AddObject is useful for "pre-loading"
// a pool with idle objects. (Optional operation).
func (this *ObjectPool) AddObject() error {
	if this.IsClosed() {
		return NewIllegalStatusErr("Pool not open")
	}
	if this.factory == nil {
		return NewIllegalStatusErr("Cannot add objects without a factory.")
	}
	this.addIdleObject(this.create())
	return nil
}

func (this *ObjectPool) addIdleObject(p *PooledObject) {
	if p != nil {
		this.factory.PassivateObject(p)
		if this.Config.Lifo {
			this.idleObjects.AddFirst(p)
		} else {
			this.idleObjects.AddLast(p)
		}
	}
}

//Obtains an instance from this pool.
//
// Instances returned from this method will have been either newly created
// with PooledObjectFactory.MakeObject or will be a previously
// idle object and have been activated with
// PooledObjectFactory.ActivateObject and then validated with
// PooledObjectFactory.ValidateObject.
//
// By contract, clients must return the borrowed instance
// using ReturnObject, InvalidateObject
func (this *ObjectPool) BorrowObject() (interface{}, error) {
	return this.borrowObject(this.Config.MaxWaitMillis)
}

//Return the number of instances currently idle in this pool. This may be
//considered an approximation of the number of objects that can be
//BorrowObject borrowed without creating any new instances.
func (this *ObjectPool) GetNumIdle() int {
	return this.idleObjects.Size()
}


//Return the number of instances currently borrowed from this pool.
func (this *ObjectPool) GetNumActive() int {
	return this.allObjects.Size() - this.idleObjects.Size()
}

func (this *ObjectPool) GetDestroyedCount() int {
	return int(this.destroyedCount.Get())
}

func (this *ObjectPool) removeAbandoned(config *AbandonedConfig) {
	// Generate a list of abandoned objects to remove
	now := currentTimeMillis()
	timeout := now - int64((config.RemoveAbandonedTimeout * 1000))
	var remove []PooledObject
	objects := this.allObjects.Values()
	for _, o := range objects {
		pooledObject := o.(PooledObject)
		pooledObject.lock.Lock()
		if pooledObject.state == ALLOCATED &&
			pooledObject.GetLastUsedTime() <= timeout {
			pooledObject.markAbandoned()
			remove = append(remove, pooledObject)
		}
		pooledObject.lock.Unlock()
	}

	// Now remove the abandoned objects
	for _, pooledObject := range remove {
		//if (config.getLogAbandoned()) {
		//pooledObject.printStackTrace(ac.getLogWriter());
		//}
		this.InvalidateObject(pooledObject.Object)
	}
}

func (this *ObjectPool) create() *PooledObject {
	if debug_pool {
		fmt.Printf("pool create\n")
	}
	localMaxTotal := this.Config.MaxTotal
	newCreateCount := this.createCount.IncrementAndGet()
	if localMaxTotal > -1 && int(newCreateCount) > localMaxTotal ||
		newCreateCount >= math.MaxInt32 {
		this.createCount.DecrementAndGet()
		return nil
	}

	p, e := this.factory.MakeObject()
	if e != nil {
		this.createCount.DecrementAndGet()
		//return error ?
		return nil
	}

	//	ac := this.abandonedConfig;
	//	if (ac != null && ac.getLogAbandoned()) {
	//		p.setLogAbandoned(true);
	//	}
	this.allObjects.Put(p.Object, p)
	return p
}

func (this *ObjectPool) destroy(toDestroy *PooledObject) {
	this.doDestroy(toDestroy, false)
}

func (this *ObjectPool) doDestroy(toDestroy *PooledObject, inLock bool) {
	if debug_pool {
		fmt.Printf("pool destroy %v \n", toDestroy.Object)
	}
	//golang has not recursive lock, so ...
	if inLock {
		toDestroy.invalidate()
	} else {
		toDestroy.Invalidate()
	}
	this.idleObjects.Remove(toDestroy)
	this.allObjects.Remove(toDestroy.Object)
	this.factory.DestroyObject(toDestroy)
	this.destroyedCount.IncrementAndGet()
	this.createCount.DecrementAndGet()
}

func (this *ObjectPool) updateStatsBorrow(object *PooledObject, timeMillis int64) {
	//TODO
}

func (this *ObjectPool) updateStatsReturn(activeTime int64) {
	//TODO
	//returnedCount.incrementAndGet();
	//activeTimes.add(activeTime);
}

func (this *ObjectPool) borrowObject(borrowMaxWaitMillis int64) (interface{}, error) {
	if this.closed {
		return nil, NewIllegalStatusErr("Pool not open")
	}
	ac := this.AbandonedConfig
	if ac != nil && ac.RemoveAbandonedOnBorrow &&
		(this.GetNumIdle() < 2) &&
		(this.GetNumActive() > this.Config.MaxTotal-3) {
		this.removeAbandoned(ac)
	}

	var p *PooledObject

	// Get local copy of current config so it is consistent for entire
	// method execution
	blockWhenExhausted := this.Config.BlockWhenExhausted

	var create bool
	waitTime := currentTimeMillis()
	var ok bool
	for p == nil {
		create = false
		if blockWhenExhausted {
			p, ok = this.idleObjects.PollFirst().(*PooledObject)
			if !ok {
				p = this.create()
				if p != nil {
					create = true
					ok = true
				}
			}
			if debug_pool {
				fmt.Printf("pool create: %v, borrowMaxWaitMillis: %v, ok:%v,  p:%v \n", create, borrowMaxWaitMillis, ok, p)
			}
			if p == nil {
				if borrowMaxWaitMillis < 0 {
					p, ok = this.idleObjects.TakeFirst().(*PooledObject)
				} else {
					p, ok = this.idleObjects.PollFirstWithTimeout(time.Duration(borrowMaxWaitMillis) * time.Millisecond).(*PooledObject)
				}
			}
			if debug_pool {
				fmt.Printf("pool ok:%v,  p:%v \n", ok, p)
			}
			if !ok {
				return nil, NewNoSuchElementErr("Timeout waiting for idle object")
			}
			if !p.Allocate() {
				p = nil
			}
			if debug_pool {
				fmt.Printf("pool p.Allocate p:%v \n", p)
			}
		} else {
			p, ok = this.idleObjects.PollFirst().(*PooledObject)
			if !ok {
				p = this.create()
				if p != nil {
					create = true
				}
			}
			if p == nil {
				return nil, NewNoSuchElementErr("Timeout waiting for idle object")
			}
			if !p.Allocate() {
				p = nil
			}
		}

		if p != nil {
			e := this.factory.ActivateObject(p)
			if e != nil {
				if debug_pool {
					fmt.Printf("pool ActiveObject fail:%v \n", e)
				}
				this.destroy(p)
				p = nil
				if create {
					return nil, NewNoSuchElementErr("Timeout waiting for idle object")
				}
			}
		}
		if debug_pool {
			fmt.Printf("pool ActiveObject end %v \n", p)
		}
		if p != nil && (this.Config.TestOnBorrow || create && this.Config.TestOnCreate) {
			validate := this.factory.ValidateObject(p)
			if !validate {
				this.destroy(p)
				//destroyedByBorrowValidationCount.incrementAndGet()
				p = nil
				if create {
					return nil, NewNoSuchElementErr("Unable to validate object")
				}
			}
		}
	}

	this.updateStatsBorrow(p, currentTimeMillis()-waitTime)
	if debug_pool {
		fmt.Printf("pool borrowObject p:%v ,p.Object:%v \n", p, p.Object)
	}
	return p.Object, nil
}

func (this *ObjectPool) isAbandonedConfig() bool {
	return this.AbandonedConfig != nil
}

func (this *ObjectPool) ensureIdle(idleCount int, always bool) {
	//if (idleCount < 1 || this.IsClosed() || (!always && !this.idleObjects.hasTakeWaiters())) {
	//TODO how to implement this.idleObjects.hasTakeWaiters()?
	if idleCount < 1 || this.IsClosed() || !always {
		return
	}

	for this.idleObjects.Size() < idleCount {
		p := this.create()
		if p == nil {
			// Can't create objects, no reason to think another call to
			// create will work. Give up.
			break
		}
		if this.Config.Lifo {
			this.idleObjects.AddFirst(p)
		} else {
			this.idleObjects.AddLast(p)
		}
	}
	if this.IsClosed() {
		// Pool closed while object was being added to idle objects.
		// Make sure the returned object is destroyed rather than left
		// in the idle object pool (which would effectively be a leak)
		this.Clear()
	}
}

func (this *ObjectPool) IsClosed() bool {
	return this.closed
}

// Return an instance to the pool. By contract, object
// must have been obtained using BorrowObject()
func (this *ObjectPool) ReturnObject(object interface{}) error {
	if debug_pool {
		fmt.Printf("pool ReturnObject %v \n", object)
	}
	if object == nil {
		return errors.New("object is nil.")
	}
	p, ok := this.allObjects.Get(object).(*PooledObject)

	if !ok {
		if !this.isAbandonedConfig() {
			return NewIllegalStatusErr(
				"Returned object not currently part of this pool")
		}
		return nil // Object was abandoned and removed
	}
	p.lock.Lock()

	state := p.state
	if state != ALLOCATED {
		p.lock.Unlock()
		return NewIllegalStatusErr(
			"Object has already been returned to this pool or is invalid")
	}
	//use unlock method markReturning() not MarkReturning
	// because go lock is not recursive
	p.markReturning() // Keep from being marked abandoned
	p.lock.Unlock()
	activeTime := p.GetActiveTimeMillis()

	if this.Config.TestOnReturn {
		if !this.factory.ValidateObject(p) {
			this.destroy(p)
			this.ensureIdle(1, false)
			this.updateStatsReturn(activeTime)
			// swallowException(e);
			return nil
		}
	}

	err := this.factory.PassivateObject(p)
	if err != nil {
		//swallowException(e1);
		this.destroy(p)
		this.ensureIdle(1, false)
		this.updateStatsReturn(activeTime)
		// swallowException(e);
		return nil
	}

	if !p.Deallocate() {
		return NewIllegalStatusErr("Object has already been returned to this pool or is invalid")
	}

	maxIdleSave := this.Config.MaxIdle
	if this.IsClosed() || maxIdleSave > -1 && maxIdleSave <= this.idleObjects.Size() {
		this.destroy(p)
	} else {
		if this.Config.Lifo {
			this.idleObjects.AddFirst(p)
		} else {
			this.idleObjects.AddLast(p)
		}
		if this.IsClosed() {
			// Pool closed while object was being added to idle objects.
			// Make sure the returned object is destroyed rather than left
			// in the idle object pool (which would effectively be a leak)
			this.Clear()
		}
	}
	this.updateStatsReturn(activeTime)
	return nil
}

//Clears any objects sitting idle in the pool, releasing any associated
//resources (optional operation). Idle objects cleared must be
//PooledObjectFactory.DestroyObject(PooledObject) .
func (this *ObjectPool) Clear() {
	p, ok := this.idleObjects.Poll().(*PooledObject)

	for ok {
		this.destroy(p)
		p, ok = this.idleObjects.Poll().(*PooledObject)
	}
}

// Invalidates an object from the pool.
//
// By contract, object must have been obtained
// using BorrowObject.
// 
// This method should be used when an object that has been borrowed is
// determined (due to an exception or other problem) to be invalid.
func (this *ObjectPool) InvalidateObject(object interface{}) error {
	p, ok := this.allObjects.Get(object).(*PooledObject)
	if !ok {
		if this.isAbandonedConfig() {
			return nil
		} else {
			return NewIllegalStatusErr(
				"Invalidated object not currently part of this pool")
		}
	}
	p.lock.Lock()
	if p.state != INVALID {
		this.doDestroy(p, true)
	}
	p.lock.Unlock()
	this.ensureIdle(1, false)
	return nil
}

//Close this pool, and free any resources associated with it.
func (this *ObjectPool) Close() {
	if this.IsClosed() {
		return
	}
	this.closeLock.Lock()
	if this.IsClosed() {
		return
	}

	// Stop the evictor before the pool is closed since evict() calls
	// assertOpen()
	this.startEvictor(-1)

	this.closed = true
	// This clear removes any idle objects
	this.Clear()

	// Release any threads that were waiting for an object
	this.idleObjects.InterruptTakeWaiters()
	this.closeLock.Unlock()
}

//if ObjectPool.Config.TimeBetweenEvictionRunsMillis change, should call this method to let it to take effect.
func (this *ObjectPool) StartEvictor() {
	this.startEvictor(this.Config.TimeBetweenEvictionRunsMillis)
}

func (this *ObjectPool) startEvictor(delay int64) {
	this.evictionLock.Lock()
	if nil != this.evictor {
		this.evictor.Stop()
		this.evictor = nil
		//evictionIterator = null;
	}
	if delay > 0 {
		this.evictor = time.NewTicker(time.Duration(delay) * time.Millisecond)
		go func() {
			for _ = range this.evictor.C {
				this.evict()
				this.ensureMinIdle()
			}
		}()
	}
	this.evictionLock.Unlock()
}

func (this *ObjectPool) getEvictionPolicy() EvictionPolicy {
	evictionPolicy := GetEvictionPolicy(this.Config.EvictionPolicyName)
	if evictionPolicy == nil {
		evictionPolicy = GetEvictionPolicy(DEFAULT_EVICTION_POLICY_NAME)
	}
	return evictionPolicy
}

func (this *ObjectPool) getNumTests() int {
	numTestsPerEvictionRun := this.Config.NumTestsPerEvictionRun
	if numTestsPerEvictionRun >= 0 {
		if numTestsPerEvictionRun < this.idleObjects.Size() {
			return numTestsPerEvictionRun
		} else {
			return this.idleObjects.Size()
		}
	}
	return int((math.Ceil(float64(this.idleObjects.Size()) / math.Abs(float64(numTestsPerEvictionRun)))))
}

func (this *ObjectPool) EvictionIterator() collections.Iterator {
	if this.Config.Lifo {
		return this.idleObjects.DescendingIterator()
	} else {
		return this.idleObjects.Iterator()
	}
}

func (this *ObjectPool) getMinIdle() int {
	maxIdleSave := this.Config.MaxIdle
	if this.Config.MinIdle > maxIdleSave {
		return maxIdleSave
	}
	return this.Config.MinIdle
}

func (this *ObjectPool) evict() {
	if debug_pool {
		fmt.Printf("pool evict idle: %v \n", this.idleObjects.Size())
	}
	if this.idleObjects.Size() > 0 {
		var underTest *PooledObject
		evictionPolicy := this.getEvictionPolicy()
		this.evictionLock.Lock()
		evictionConfig := EvictionConfig{
			IdleEvictTime:     this.Config.MinEvictableIdleTimeMillis,
			IdleSoftEvictTime: this.Config.SoftMinEvictableIdleTimeMillis,
			MinIdle:           this.Config.MinIdle}

		testWhileIdle := this.Config.TestWhileIdle
		for i, m := 0, this.getNumTests(); i < m; i++ {
			if this.evictionIterator == nil || !this.evictionIterator.HasNext() {
				this.evictionIterator = this.EvictionIterator()
			}
			if !this.evictionIterator.HasNext() {
				// Pool exhausted, nothing to do here
				return
			}

			underTest = this.evictionIterator.Next().(*PooledObject)
			if underTest == nil {
				// Object was borrowed in another thread
				// Don't count this as an eviction test so reduce i;
				i--
				this.evictionIterator = nil
				continue
			}

			if !underTest.startEvictionTest() {
				// Object was borrowed in another thread
				// Don't count this as an eviction test so reduce i;
				i--
				continue
			}

			// User provided eviction policy could throw all sorts of
			// crazy exceptions. Protect against such an exception
			// killing the eviction thread.

			evict := evictionPolicy.Evict(&evictionConfig, underTest, this.idleObjects.Size())

			if evict {
				if debug_pool {
					fmt.Printf("pool evict %v \n", underTest.Object)
				}
				this.destroy(underTest)
				this.destroyedByEvictorCount.IncrementAndGet()
			} else {
				var active bool = false
				if testWhileIdle {
					err := this.factory.ActivateObject(underTest)
					if err == nil {
						active = true
					} else {
						this.destroy(underTest)
						this.destroyedByEvictorCount.IncrementAndGet()
					}
					if active {
						if !this.factory.ValidateObject(underTest) {
							this.destroy(underTest)
							this.destroyedByEvictorCount.IncrementAndGet()
						} else {
							err := this.factory.PassivateObject(underTest)
							if err != nil {
								this.destroy(underTest)
								this.destroyedByEvictorCount.IncrementAndGet()
							}
						}
					}
				}
				if !underTest.endEvictionTest(this.idleObjects) {
					// TODO - May need to add code here once additional
					// states are used
				}
			}
		}
		this.evictionLock.Unlock()
	}
	ac := this.AbandonedConfig
	if ac != nil && ac.RemoveAbandonedOnMaintenance {
		this.removeAbandoned(ac)
	}
}

func (this *ObjectPool) ensureMinIdle() {
	this.ensureIdle(this.getMinIdle(), true)
}

func (this *ObjectPool) preparePool() {
	if this.getMinIdle() < 1 {
		return
	}
	this.ensureMinIdle()
}

func Prefill(pool *ObjectPool, count int) {
	for i := 0; i < count; i++ {
		pool.AddObject()
	}
}
