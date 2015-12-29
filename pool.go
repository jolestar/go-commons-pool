package pool

import (
	"errors"
	"github.com/jolestar/go-commons-pool/collections"
	"math"
	"sync"
	"sync/atomic"
	"time"
	"fmt"
)
const(
	debug = true
)
type ObjectPool struct {
	AbandonedConfig         *AbandonedConfig
	PoolConfig              *ObjectPoolConfig
	closed                  bool
	closeLock               sync.Mutex
	evictionLock            sync.Mutex
	idleObjects             *collections.LinkedBlockDeque
	allObjects              *collections.SyncIdentityMap
	factory                 PooledObjectFactory
	createCount             int32
	destroyedByEvictorCount int32
	evictor                 *time.Ticker
	evictionIterator        collections.Iterator
	evictionPolicy          EvictionPolicy
}

func NewObjectPool(factory PooledObjectFactory, config *ObjectPoolConfig) *ObjectPool {
	return &ObjectPool{factory:factory, PoolConfig:config,
		idleObjects: collections.NewDeque(math.MaxInt32),
		allObjects:collections.NewSyncMap()}
}

func NewObjectPoolWithDefaultConfig(factory PooledObjectFactory) *ObjectPool {
	return NewObjectPool(factory, NewDefaultPoolConfig())
}

func (this *ObjectPool) AddObject() {
	if this.IsClosed() {
		//TODO panic?
		return
	}
	if this.factory == nil {
		panic(errors.New(
			"Cannot add objects without a factory."))
	}
	this.addIdleObject(this.create())
}

func (this *ObjectPool) addIdleObject(p *PooledObject) {
	if p != nil {
		this.factory.PassivateObject(p)
		if this.PoolConfig.Lifo {
			this.idleObjects.AddFirst(p)
		} else {
			this.idleObjects.AddLast(p)
		}
	}
}

func (this *ObjectPool) BorrowObject() (interface{}, error) {
	return this.borrowObject(this.PoolConfig.MaxWaitMillis)
}

func (this *ObjectPool) GetNumIdle() int {
	return this.idleObjects.Size()
}

func (this *ObjectPool) GetNumActive() int {
	return this.allObjects.Size() - this.idleObjects.Size()
}

func (this *ObjectPool) removeAbandoned(config *AbandonedConfig) {
	// Generate a list of abandoned objects to remove
	now := currentTimeMillis()
	timeout := now - int64((config.RemoveAbandonedTimeout * 1000))
	var remove []PooledObject
	objects := this.allObjects.Values()
	for _,o := range objects {
		pooledObject := o.(PooledObject)
		pooledObject.lock.Lock()
		if pooledObject.state == ALLOCATED &&
			pooledObject.GetLastUsedTime() <= timeout {
			pooledObject.MarkAbandoned()
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

func (this *ObjectPool) incrementCreateCount() int32 {
	return atomic.AddInt32(&this.createCount, int32(1))
}

func (this *ObjectPool) decrementCreateCount() int32 {
	return atomic.AddInt32(&this.createCount, int32(-1))
}

func (this *ObjectPool) incrementDestroyedByEvictorCount() int32 {
	return atomic.AddInt32(&this.destroyedByEvictorCount, int32(1))
}

func (this *ObjectPool) decrementDestroyedByEvictorCount() int32 {
	return atomic.AddInt32(&this.destroyedByEvictorCount, int32(-1))
}

func (this *ObjectPool) create() *PooledObject {
	if(debug){
		fmt.Printf("pool create\n")
	}
	localMaxTotal := this.PoolConfig.MaxTotal
	newCreateCount := this.incrementCreateCount()
	if localMaxTotal > -1 && int(newCreateCount) > localMaxTotal ||
		newCreateCount >= math.MaxInt32 {
		this.decrementCreateCount()
		return nil
	}

	p, e := this.factory.MakeObject()
	if e != nil {
		this.decrementCreateCount()
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
	if(debug){
		fmt.Printf("pool destroy %v \n", toDestroy)
	}
	toDestroy.Invalidate()
	this.idleObjects.Remove(toDestroy)
	this.allObjects.Remove(toDestroy.Object)
	this.factory.DestroyObject(toDestroy)
	//destroyedCount.incrementAndGet();
	this.decrementCreateCount()
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
		return nil, errors.New("Pool not open")
	}
	ac := this.AbandonedConfig
	if ac != nil && ac.RemoveAbandonedOnBorrow &&
		(this.GetNumIdle() < 2) &&
		(this.GetNumActive() > this.PoolConfig.MaxTotal-3) {
		this.removeAbandoned(ac)
	}

	var p *PooledObject

	// Get local copy of current config so it is consistent for entire
	// method execution
	blockWhenExhausted := this.PoolConfig.BlockWhenExhausted

	var create bool
	waitTime := currentTimeMillis()
	var ok bool
	for p == nil {
		create = false
		if blockWhenExhausted {
			p,ok = this.idleObjects.PollFirst().(*PooledObject)
			if !ok {
				p = this.create()
				if p != nil {
					create = true
					ok = true
				}
			}
			if(debug){
				fmt.Printf("pool create: %v, borrowMaxWaitMillis: %v, ok:%v,  p:%v \n", create, borrowMaxWaitMillis, ok, p)
			}
			if p == nil {
				if borrowMaxWaitMillis < 0 {
					p,ok = this.idleObjects.TakeFirst().(*PooledObject)
				} else {
					p,ok = this.idleObjects.PollFirstWithTimeout(time.Duration(borrowMaxWaitMillis) * time.Millisecond).(*PooledObject)
				}
			}
			if(debug){
				fmt.Printf("pool ok:%v,  p:%v \n", ok, p)
			}
			if !ok {
				return nil, errors.New("Timeout waiting for idle object")
			}
			if !p.Allocate() {
				p = nil
			}
			if(debug){
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
				return nil, errors.New("Timeout waiting for idle object")
			}
			if !p.Allocate() {
				p = nil
			}
		}

		if p != nil {
			e := this.factory.ActivateObject(p)
			if e != nil {
				if(debug){
					fmt.Printf("pool ActiveObject fail:%v \n", e)
				}
				this.destroy(p)
				p = nil
				if create {
					return nil, errors.New("Timeout waiting for idle object")
				}
			}
		}
		if(debug){
			fmt.Printf("pool ActiveObject end %v \n", p)
		}
		if p != nil && (this.PoolConfig.TestOnBorrow || create && this.PoolConfig.TestOnCreate) {
			validate := this.factory.ValidateObject(p)
			if !validate {
				this.destroy(p)
				//destroyedByBorrowValidationCount.incrementAndGet()
				p = nil
				if create {
					return nil, errors.New("Unable to validate object")
				}
			}
		}
	}

	this.updateStatsBorrow(p, currentTimeMillis()-waitTime)
	if(debug){
		fmt.Printf("pool borrowObject p:%v ,p.Object:%v \n", p, p.Object)
	}
	return p.Object,nil
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
		if this.PoolConfig.Lifo {
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

func (this *ObjectPool) ReturnObject(object interface{}) error {
	if(debug){
		fmt.Printf("pool ReturnObject %v \n", object)
	}
	if(object == nil){
		return errors.New("object is nil.")
	}
	p,ok := this.allObjects.Get(object).(*PooledObject)

	if !ok {
		if !this.isAbandonedConfig() {
			return errors.New(
				"Returned object not currently part of this pool")
		}
		return nil // Object was abandoned and removed
	}
	p.lock.Lock()

	state := p.state
	if state != ALLOCATED {
		p.lock.Unlock()
		return errors.New(
			"Object has already been returned to this pool or is invalid")
	}
	//use unlock method markReturning() not MarkReturning
	// because go lock is not recursive
	p.markReturning() // Keep from being marked abandoned
	p.lock.Unlock()
	activeTime := p.GetActiveTimeMillis()

	if this.PoolConfig.TestOnReturn {
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
		return errors.New("Object has already been returned to this pool or is invalid")
	}

	maxIdleSave := this.PoolConfig.MaxIdle
	if this.IsClosed() || maxIdleSave > -1 && maxIdleSave <= this.idleObjects.Size() {
		this.destroy(p)
	} else {
		if this.PoolConfig.Lifo {
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

func (this *ObjectPool) Clear() {
	p, ok := this.idleObjects.Poll().(*PooledObject)

	for ok {
		this.destroy(p)
		p,ok = this.idleObjects.Poll().(*PooledObject)
	}
}

func (this *ObjectPool) InvalidateObject(object interface{}) error {
	p,ok := this.allObjects.Get(object).(*PooledObject)
	if !ok {
		if this.isAbandonedConfig() {
			return nil
		} else {
			return errors.New(
				"Invalidated object not currently part of this pool")
		}
	}
	if p.GetState() != INVALID {
		this.destroy(p)
	}
	this.ensureIdle(1, false)
	return nil
}

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

	//jmxUnregister();

	// Release any threads that were waiting for an object
	this.idleObjects.InterruptTakeWaiters()
	this.closeLock.Unlock()
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
	return this.evictionPolicy
}

func (this *ObjectPool) getNumTests() int {
	numTestsPerEvictionRun := this.PoolConfig.NumTestsPerEvictionRun
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
	if this.PoolConfig.Lifo {
		return this.idleObjects.DescendingIterator()
	} else {
		return this.idleObjects.Iterator()
	}
}

func (this *ObjectPool) getMinIdle() int{
	maxIdleSave := this.PoolConfig.MaxIdle
	if this.PoolConfig.MinIdle > maxIdleSave {
		return maxIdleSave
	}
	return this.PoolConfig.MinIdle
}

func (this *ObjectPool) evict() {
	if this.idleObjects.Size() > 0 {
		var underTest *PooledObject
		evictionPolicy := this.getEvictionPolicy()
		this.evictionLock.Lock()
		evictionConfig := EvictionConfig{
			IdleEvictTime:     this.PoolConfig.MinEvictableIdleTimeMillis,
			IdleSoftEvictTime: this.PoolConfig.SoftMinEvictableIdleTimeMillis,
			MinIdle:           this.PoolConfig.MinIdle}

		testWhileIdle := this.PoolConfig.TestWhileIdle

		for i, m := 0, this.getNumTests(); i < m; i++ {
			if this.evictionIterator == nil || !this.evictionIterator.HasNext() {
				this.evictionIterator = this.EvictionIterator()
			}
			if !this.evictionIterator.HasNext() {
				// Pool exhausted, nothing to do here
				return
			}

			underTest = this.evictionIterator.Next().(*PooledObject)
			//} catch (NoSuchElementException nsee) {
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

			evict, err := evictionPolicy.Evict(&evictionConfig, underTest, this.idleObjects.Size())
			if err != nil {
				// Slightly convoluted as SwallowedExceptionListener
				// uses Exception rather than Throwable
				//PoolUtils.checkRethrow(t);
				//swallowException(new Exception(t));
				// Don't evict on error conditions
				evict = false
			}

			if evict {
				this.destroy(underTest)
				this.incrementDestroyedByEvictorCount()
			} else {
				var active bool = false
				if testWhileIdle {
					err := this.factory.ActivateObject(underTest)
					if err == nil {
						active = true
					} else {
						this.destroy(underTest)
						this.incrementDestroyedByEvictorCount()
					}
					if active {
						if !this.factory.ValidateObject(underTest) {
							this.destroy(underTest)
							this.incrementDestroyedByEvictorCount()
						} else {
							err := this.factory.PassivateObject(underTest)
							if err != nil {
								this.destroy(underTest)
								this.incrementDestroyedByEvictorCount()
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
