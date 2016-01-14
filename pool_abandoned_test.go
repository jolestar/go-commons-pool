package pool

import (
	"fmt"
	"github.com/jolestar/go-commons-pool/concurrent"
	"github.com/stretchr/testify/suite"
	"sync"
	"testing"
	"time"
)

type AbandonedTestObject struct {
	active     bool
	destroyed  bool
	_hash      int
	_abandoned bool
}

func NewAbandonedTestObject() *AbandonedTestObject {
	object := AbandonedTestObject{}
	return &object
}

func (this *AbandonedTestObject) GetLastUsed() int64 {
	if this._abandoned {
		// Abandoned object sweep will occur no matter what the value of removeAbandonedTimeout,
		// because this indicates that this object was last used decades ago
		return 1
	}
	// Abandoned object sweep won't clean up this object
	return 0
}

func (this *AbandonedTestObject) destroy() {
	this.destroyed = true
}

func (this *AbandonedTestObject) hashCode() int {
	return this._hash
}

type SimpleAbandonedFactory struct {
	destroyLatency  int64
	validateLatency int64
	counter         concurrent.AtomicInteger
}

func NewSimpleAbandonedFactory() *SimpleAbandonedFactory {
	return &SimpleAbandonedFactory{}
}

func (this *SimpleAbandonedFactory) doWait(latencyMillisecond int64) {
	time.Sleep(time.Duration(latencyMillisecond) * time.Millisecond)
}

func (this *SimpleAbandonedFactory) MakeObject() (*PooledObject, error) {
	if debug_test {
		fmt.Println("factory MakeObject")
	}
	object := NewAbandonedTestObject()
	object._hash = int(this.counter.IncrementAndGet())
	return NewPooledObject(object), nil
}

func (this *SimpleAbandonedFactory) DestroyObject(object *PooledObject) error {
	if debug_test {
		fmt.Println("factory DestroyObject")
	}
	object.Object.(*AbandonedTestObject).active = false
	// while destroying instances, yield control to other threads
	// helps simulate threading errors
	//Thread.yield();
	if this.destroyLatency != 0 {
		this.doWait(this.destroyLatency)
	}
	object.Object.(*AbandonedTestObject).destroy()
	return nil
}

func (this *SimpleAbandonedFactory) ValidateObject(object *PooledObject) bool {
	if debug_test {
		fmt.Println("factory ValidateObject")
	}
	this.doWait(this.validateLatency)
	return true
}

func (this *SimpleAbandonedFactory) ActivateObject(object *PooledObject) error {
	if debug_test {
		fmt.Println("factory ActivateObject")
		defer fmt.Println("factory ActivateObject end")
	}
	object.Object.(*AbandonedTestObject).active = true
	return nil
}

func (this *SimpleAbandonedFactory) PassivateObject(object *PooledObject) error {
	if debug_test {
		fmt.Println("factory PassivateObject")
	}
	object.Object.(*AbandonedTestObject).active = false
	return nil
}

type PoolAbandonedTestSuite struct {
	suite.Suite
	pool    *ObjectPool
	factory *SimpleAbandonedFactory
}

func (this *PoolAbandonedTestSuite) NoErrorWithResult(object interface{}, err error) interface{} {
	this.NotNil(object)
	this.Nil(err)
	return object
}

func (this *PoolAbandonedTestSuite) ErrorWithResult(object interface{}, err error) error {
	this.Nil(object)
	this.NotNil(err)
	return err
}

func TestPoolAbandonedTestSuite(t *testing.T) {
	suite.Run(t, new(PoolAbandonedTestSuite))
}

func (this *PoolAbandonedTestSuite) SetupTest() {
	fmt.Println("PoolAbandonedTestSuite SetupTest")
	abandonedConfig := NewDefaultAbandonedConfig()

	// -- Uncomment the following line to enable logging --
	// abandonedConfig.setLogAbandoned(true);

	abandonedConfig.RemoveAbandonedOnBorrow = true
	abandonedConfig.RemoveAbandonedTimeout = 1
	factory := NewSimpleAbandonedFactory()
	config := NewDefaultPoolConfig()
	this.factory = factory
	this.pool = NewObjectPoolWithAbandonedConfig(factory, config, abandonedConfig)
}

func (this *PoolAbandonedTestSuite) TearDownTest() {
	this.pool.Clear()
	this.pool.Close()
	this.pool = nil
	this.factory = nil
}

func concurrentBorrower(pool *ObjectPool, objects chan *AbandonedTestObject, wait *sync.WaitGroup) {
	go func() {
		o, _ := pool.BorrowObject()
		if o != nil {
			objects <- o.(*AbandonedTestObject)
		}
		wait.Done()
	}()
}

func concurrentReturner(pool *ObjectPool, object *AbandonedTestObject, wait *sync.WaitGroup) {
	go func() {
		wait.Wait()
		sleep(20)
		pool.ReturnObject(object)
	}()
}

/**
 * Tests fix for Bug 28579, a bug in AbandonedObjectPool that causes numActive to go negative
 * in GenericObjectPool
 */
func (this *PoolAbandonedTestSuite) TestConcurrentInvalidation() {
	fmt.Println("PoolAbandonedTestSuite TestConcurrentInvalidation")
	POOL_SIZE := 30
	this.pool.Config.MaxTotal = POOL_SIZE
	this.pool.Config.MaxIdle = POOL_SIZE
	this.pool.Config.BlockWhenExhausted = false

	// Exhaust the connection pool
	vec := make([]*AbandonedTestObject, POOL_SIZE)
	for i := 0; i < POOL_SIZE; i++ {
		vec[i] = this.NoErrorWithResult(this.pool.BorrowObject()).(*AbandonedTestObject)
	}

	// Abandon all borrowed objects
	for i := 0; i < len(vec); i++ {
		vec[i]._abandoned = true
	}

	// Try launching a bunch of borrows concurrently.  Abandoned sweep will be triggered for each.
	CONCURRENT_BORROWS := 5

	objects := make(chan *AbandonedTestObject, POOL_SIZE)
	wait := sync.WaitGroup{}
	wait.Add(CONCURRENT_BORROWS)
	for i := 0; i < CONCURRENT_BORROWS; i++ {
		concurrentBorrower(this.pool, objects, &wait)
	}

	// Wait for all the goroutine to finish
	wait.Wait()

	for i := 0; i < len(objects); i++ {
		vec = append(vec, <-objects)
	}
	close(objects)
	// Return all objects that have not been destroyed
	for i := 0; i < len(vec); i++ {
		pto := vec[i]
		if pto.active {
			this.NoError(this.pool.ReturnObject(pto))
		}
	}

	// Now, the number of active instances should be 0
	this.True(this.pool.GetNumActive() == 0, "numActive should have been 0, was %v", this.pool.GetNumActive())
}

/**
 * Verify that an object that gets flagged as abandoned and is subsequently returned
 * is destroyed instead of being returned to the pool (and possibly later destroyed
 * inappropriately).
 */
func (this *PoolAbandonedTestSuite) TestAbandonedReturn() {
	this.pool.AbandonedConfig.RemoveAbandonedOnBorrow = true
	this.pool.AbandonedConfig.RemoveAbandonedOnMaintenance = false
	this.pool.AbandonedConfig.RemoveAbandonedTimeout = 1
	this.factory.destroyLatency = 200

	n := 10
	this.pool.Config.MaxTotal = n
	this.pool.Config.BlockWhenExhausted = false
	var obj *AbandonedTestObject
	for i := 0; i < n-2; i++ {
		obj = this.NoErrorWithResult(this.pool.BorrowObject()).(*AbandonedTestObject)
	}
	if obj == nil {
		this.Fail("Unable to borrow object from pool")
	}
	wait := new(sync.WaitGroup)
	wait.Add(1)
	deadMansHash := obj.hashCode()
	fmt.Println("deadMansHash:", deadMansHash)
	concurrentReturner(this.pool, obj, wait)
	sleep(2000) // abandon checked out instances
	// Now start a race - returner waits until borrowObject has kicked
	// off removeAbandoned and then returns an instance that borrowObject
	// will deem abandoned.  Make sure it is not returned to the borrower.
	wait.Done() // short delay, then return instance
	obj2 := this.NoErrorWithResult(this.pool.BorrowObject()).(*AbandonedTestObject)
	this.True(obj2.hashCode() != deadMansHash)
	this.Equal(0, this.pool.GetNumIdle())
	this.Equal(1, this.pool.GetNumActive())
}

/**
 * Verify that an object that gets flagged as abandoned and is subsequently
 * invalidated is only destroyed (and pool counter decremented) once.
 */
func (this *PoolAbandonedTestSuite) TestAbandonedInvalidate() {
	this.pool.AbandonedConfig.RemoveAbandonedOnBorrow = false
	this.pool.AbandonedConfig.RemoveAbandonedOnMaintenance = true
	this.pool.AbandonedConfig.RemoveAbandonedTimeout = 1
	// destroys take 200 ms
	this.factory.destroyLatency = 200
	n := 10
	this.pool.Config.MaxTotal = n
	this.pool.Config.BlockWhenExhausted = false
	this.pool.Config.TimeBetweenEvictionRunsMillis = 500
	this.pool.StartEvictor()

	var obj *AbandonedTestObject
	for i := 0; i < 5; i++ {
		obj = this.NoErrorWithResult(this.pool.BorrowObject()).(*AbandonedTestObject)
	}

	sleep(1000)                     // abandon checked out instances and let evictor start
	this.pool.InvalidateObject(obj) // Should not trigger another destroy / decrement
	sleep(2000)                     // give evictor time to finish destroys
	this.Equal(0, this.pool.GetNumActive())
	this.Equal(5, this.pool.GetDestroyedCount())
}

/**
 * Verify that an object that the evictor identifies as abandoned while it
 * is in process of being returned to the pool is not destroyed.
 */
func (this *PoolAbandonedTestSuite) TestRemoveAbandonedWhileReturning() {
	this.pool.AbandonedConfig.RemoveAbandonedOnBorrow = false
	this.pool.AbandonedConfig.RemoveAbandonedOnMaintenance = true
	this.pool.AbandonedConfig.RemoveAbandonedTimeout = 1

	this.factory.validateLatency = 1000
	n := 10

	this.pool.Config.MaxTotal = n
	this.pool.Config.BlockWhenExhausted = false
	this.pool.Config.TimeBetweenEvictionRunsMillis = 500
	this.pool.Config.TestOnReturn = true
	this.pool.StartEvictor()

	// Borrow an object, wait long enough for it to be abandoned
	// then arrange for evictor to run while it is being returned
	// validation takes a second, evictor runs every 500 ms
	obj := this.NoErrorWithResult(this.pool.BorrowObject())
	sleep(50)                   // abandon obj
	this.pool.ReturnObject(obj) // evictor will run during validation
	obj2 := this.NoErrorWithResult(this.pool.BorrowObject())
	this.Equal(obj, obj2)                             // should get original back
	this.True(!obj2.(*AbandonedTestObject).destroyed) // and not destroyed
}

/**
 * Test case for https://issues.apache.org/jira/browse/DBCP-260.
 * Borrow and abandon all the available objects then attempt to borrow one
 * further object which should block until the abandoned objects are
 * removed. We don't want the test to block indefinitely when it fails so
 * use maxWait be check we don't actually have to wait that long.
 *
 */
func (this *PoolAbandonedTestSuite) TestWhenExhaustedBlock() {
	this.pool.AbandonedConfig.RemoveAbandonedOnBorrow = false
	this.pool.AbandonedConfig.RemoveAbandonedOnMaintenance = true
	this.pool.AbandonedConfig.RemoveAbandonedTimeout = 1
	this.pool.Config.MaxTotal = 1
	this.pool.Config.TimeBetweenEvictionRunsMillis = 500
	this.pool.StartEvictor()

	this.NoErrorWithResult(this.pool.BorrowObject())

	start := currentTimeMillis()
	o2 := this.NoErrorWithResult(this.pool.borrowObject(5000))
	end := currentTimeMillis()

	this.pool.ReturnObject(o2)

	this.True(end-start < 5000)
}

func (this *PoolAbandonedTestSuite) TestAbandonedConfigReturnObjectError() {
	obj := NewAbandonedTestObject()
	err := this.pool.ReturnObject(obj)
	this.Nil(err)
}
