package pool

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jolestar/go-commons-pool/concurrent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type AbandonedTestObject struct {
	lock       sync.Mutex
	active     bool
	destroyed  bool
	_hash      int
	_abandoned bool
}

func (o *AbandonedTestObject) setActive(active bool) {
	o.lock.Lock()
	o.active = active
	o.lock.Unlock()
}

func NewAbandonedTestObject() *AbandonedTestObject {
	object := AbandonedTestObject{}
	return &object
}

func (o *AbandonedTestObject) GetLastUsed() time.Time {
	if o._abandoned {
		// Abandoned object sweep will occur no matter what the value of removeAbandonedTimeout,
		// because suit indicates that suit object was last used decades ago
		return time.Time{}.Add(1)
	}
	// Abandoned object sweep won't clean up suit object
	return time.Time{}
}

func (o *AbandonedTestObject) destroy() {
	o.lock.Lock()
	o.destroyed = true
	o.lock.Unlock()
}

func (o *AbandonedTestObject) hashCode() int {
	return o._hash
}

func TestAbandonedTestObject(t *testing.T) {
	t.Parallel()

	obj := NewAbandonedTestObject()
	var trackedUse TrackedUse
	trackedUse = obj
	assert.Zero(t, trackedUse.GetLastUsed())
}

type SimpleAbandonedFactory struct {
	destroyLatency  time.Duration
	validateLatency time.Duration
	counter         concurrent.AtomicInteger
}

func NewSimpleAbandonedFactory() *SimpleAbandonedFactory {
	return &SimpleAbandonedFactory{}
}

func (f *SimpleAbandonedFactory) MakeObject(context.Context) (*PooledObject, error) {
	if debugTest {
		fmt.Println("factory MakeObject")
	}
	object := NewAbandonedTestObject()
	object._hash = int(f.counter.IncrementAndGet())
	return NewPooledObject(object), nil
}

func (f *SimpleAbandonedFactory) DestroyObject(ctx context.Context, object *PooledObject) error {
	if debugTest {
		fmt.Println("factory DestroyObject")
	}
	object.Object.(*AbandonedTestObject).setActive(false)
	// while destroying instances, yield control to other threads
	// helps simulate threading errors
	//Thread.yield();
	if f.destroyLatency != 0 {
		time.Sleep(f.destroyLatency)
	}
	object.Object.(*AbandonedTestObject).destroy()
	return nil
}

func (f *SimpleAbandonedFactory) ValidateObject(ctx context.Context, object *PooledObject) bool {
	if debugTest {
		fmt.Println("factory ValidateObject")
	}
	time.Sleep(f.validateLatency)
	return true
}

func (f *SimpleAbandonedFactory) ActivateObject(ctx context.Context, object *PooledObject) error {
	if debugTest {
		fmt.Println("factory ActivateObject")
		defer fmt.Println("factory ActivateObject end")
	}
	object.Object.(*AbandonedTestObject).setActive(true)
	return nil
}

func (f *SimpleAbandonedFactory) PassivateObject(ctx context.Context, object *PooledObject) error {
	if debugTest {
		fmt.Println("factory PassivateObject")
	}
	object.Object.(*AbandonedTestObject).setActive(false)
	return nil
}

type PoolAbandonedTestSuite struct {
	suite.Suite
	pool    *ObjectPool
	factory *SimpleAbandonedFactory
}

func (suit *PoolAbandonedTestSuite) NoErrorWithResult(object interface{}, err error) interface{} {
	suit.NotNil(object)
	suit.Nil(err)
	return object
}

func (suit *PoolAbandonedTestSuite) ErrorWithResult(object interface{}, err error) error {
	suit.Nil(object)
	suit.NotNil(err)
	return err
}

func TestPoolAbandonedTestSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(PoolAbandonedTestSuite))
}

func (suit *PoolAbandonedTestSuite) SetupTest() {
	abandonedConfig := NewDefaultAbandonedConfig()

	// -- Uncomment the following line to enable logging --
	// abandonedConfig.setLogAbandoned(true);

	abandonedConfig.RemoveAbandonedOnBorrow = true
	abandonedConfig.RemoveAbandonedTimeout = 1 * time.Second
	factory := NewSimpleAbandonedFactory()
	config := NewDefaultPoolConfig()
	suit.factory = factory
	suit.pool = NewObjectPoolWithAbandonedConfig(context.Background(), factory, config, abandonedConfig)
}

func (suit *PoolAbandonedTestSuite) TearDownTest() {
	ctx := context.Background()
	suit.pool.Clear(ctx)
	suit.pool.Close(ctx)
	suit.pool = nil
	suit.factory = nil
}

func concurrentBorrower(ctx context.Context, pool *ObjectPool, objects chan *AbandonedTestObject, wait *sync.WaitGroup) {
	go func() {
		o, _ := pool.BorrowObject(ctx)
		if o != nil {
			objects <- o.(*AbandonedTestObject)
		}
		wait.Done()
	}()
}

func concurrentReturner(pool *ObjectPool, object *AbandonedTestObject, wait *sync.WaitGroup) {
	go func() {
		ctx := context.Background()
		wait.Wait()
		time.Sleep(20 * time.Millisecond)
		pool.ReturnObject(ctx, object)
	}()
}

/**
 * Tests fix for Bug 28579, a bug in AbandonedObjectPool that causes numActive to go negative
 * in GenericObjectPool
 */
func (suit *PoolAbandonedTestSuite) TestConcurrentInvalidation() {
	ctx := context.Background()
	poolSize := 30
	suit.pool.Config.MaxTotal = poolSize
	suit.pool.Config.MaxIdle = poolSize
	suit.pool.Config.BlockWhenExhausted = false

	// Exhaust the connection pool
	vec := make([]*AbandonedTestObject, poolSize)
	for i := 0; i < poolSize; i++ {
		vec[i] = suit.NoErrorWithResult(suit.pool.BorrowObject(ctx)).(*AbandonedTestObject)
	}

	// Abandon all borrowed objects
	for i := 0; i < len(vec); i++ {
		vec[i]._abandoned = true
	}

	// Try launching a bunch of borrows concurrently.  Abandoned sweep will be triggered for each.
	concurrentBorrows := 5

	objects := make(chan *AbandonedTestObject, poolSize)
	wait := sync.WaitGroup{}
	wait.Add(concurrentBorrows)
	for i := 0; i < concurrentBorrows; i++ {
		concurrentBorrower(ctx, suit.pool, objects, &wait)
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
			suit.NoError(suit.pool.ReturnObject(ctx, pto))
		}
	}

	// Now, the number of active instances should be 0
	suit.True(suit.pool.GetNumActive() == 0, "numActive should have been 0, was %v", suit.pool.GetNumActive())
}

/**
 * Verify that an object that gets flagged as abandoned and is subsequently returned
 * is destroyed instead of being returned to the pool (and possibly later destroyed
 * inappropriately).
 */
func (suit *PoolAbandonedTestSuite) TestAbandonedReturn() {
	ctx := context.Background()
	suit.pool.AbandonedConfig.RemoveAbandonedOnBorrow = true
	suit.pool.AbandonedConfig.RemoveAbandonedOnMaintenance = false
	suit.pool.AbandonedConfig.RemoveAbandonedTimeout = 1 * time.Second
	suit.factory.destroyLatency = 200 * time.Millisecond

	n := 10
	suit.pool.Config.MaxTotal = n
	suit.pool.Config.BlockWhenExhausted = false
	var obj *AbandonedTestObject
	for i := 0; i < n-2; i++ {
		obj = suit.NoErrorWithResult(suit.pool.BorrowObject(ctx)).(*AbandonedTestObject)
	}
	if obj == nil {
		suit.Fail("Unable to borrow object from pool")
	}
	wait := new(sync.WaitGroup)
	wait.Add(1)
	deadMansHash := obj.hashCode()
	if debugTest {
		fmt.Println("deadMansHash:", deadMansHash)
	}
	concurrentReturner(suit.pool, obj, wait)
	time.Sleep(2 * time.Second) // abandon checked out instances
	// Now start a race - returner waits until borrowObject has kicked
	// off removeAbandoned and then returns an instance that borrowObject
	// will deem abandoned.  Make sure it is not returned to the borrower.
	wait.Done() // short delay, then return instance
	obj2 := suit.NoErrorWithResult(suit.pool.BorrowObject(ctx)).(*AbandonedTestObject)
	suit.True(obj2.hashCode() != deadMansHash)
	suit.Equal(0, suit.pool.GetNumIdle())
	suit.Equal(1, suit.pool.GetNumActive())
}

/**
 * Verify that an object that gets flagged as abandoned and is subsequently
 * invalidated is only destroyed (and pool counter decremented) once.
 */
func (suit *PoolAbandonedTestSuite) TestAbandonedInvalidate() {
	ctx := context.Background()
	suit.pool.AbandonedConfig.RemoveAbandonedOnBorrow = false
	suit.pool.AbandonedConfig.RemoveAbandonedOnMaintenance = true
	suit.pool.AbandonedConfig.RemoveAbandonedTimeout = 1 * time.Second
	// destroys take 200 ms
	suit.factory.destroyLatency = 200 * time.Millisecond
	n := 10
	suit.pool.Config.MaxTotal = n
	suit.pool.Config.BlockWhenExhausted = false
	suit.pool.Config.TimeBetweenEvictionRuns = 500 * time.Millisecond
	suit.pool.StartEvictor()

	var obj *AbandonedTestObject
	for i := 0; i < 5; i++ {
		obj = suit.NoErrorWithResult(suit.pool.BorrowObject(ctx)).(*AbandonedTestObject)
	}

	time.Sleep(1 * time.Second)          // abandon checked out instances and let evictor start
	suit.pool.InvalidateObject(ctx, obj) // Should not trigger another destroy / decrement
	time.Sleep(2 * time.Second)          // give evictor time to finish destroys
	suit.Equal(0, suit.pool.GetNumActive())
	suit.Equal(5, suit.pool.GetDestroyedCount())
}

/**
 * Verify that an object that the evictor identifies as abandoned while it
 * is in process of being returned to the pool is not destroyed.
 */
func (suit *PoolAbandonedTestSuite) TestRemoveAbandonedWhileReturning() {
	ctx := context.Background()

	suit.pool.AbandonedConfig.RemoveAbandonedOnBorrow = false
	suit.pool.AbandonedConfig.RemoveAbandonedOnMaintenance = true
	suit.pool.AbandonedConfig.RemoveAbandonedTimeout = 1 * time.Second

	suit.factory.validateLatency = 1 * time.Second
	n := 10

	suit.pool.Config.MaxTotal = n
	suit.pool.Config.BlockWhenExhausted = false
	suit.pool.Config.TimeBetweenEvictionRuns = 500 * time.Millisecond
	suit.pool.Config.TestOnReturn = true
	suit.pool.StartEvictor()

	// Borrow an object, wait long enough for it to be abandoned
	// then arrange for evictor to run while it is being returned
	// validation takes a second, evictor runs every 500 ms
	obj := suit.NoErrorWithResult(suit.pool.BorrowObject(ctx))
	time.Sleep(50 * time.Millisecond) // abandon obj
	suit.pool.ReturnObject(ctx, obj)  // evictor will run during validation
	obj2 := suit.NoErrorWithResult(suit.pool.BorrowObject(ctx))
	suit.Equal(obj, obj2)                             // should get original back
	suit.True(!obj2.(*AbandonedTestObject).destroyed) // and not destroyed
}

/**
 * Test case for https://issues.apache.org/jira/browse/DBCP-260.
 * Borrow and abandon all the available objects then attempt to borrow one
 * further object which should block until the abandoned objects are
 * removed. We don't want the test to block indefinitely when it fails so
 * use maxWait be check we don't actually have to wait that long.
 *
 */
func (suit *PoolAbandonedTestSuite) TestWhenExhaustedBlock() {
	ctx := context.Background()

	suit.pool.AbandonedConfig.RemoveAbandonedOnBorrow = false
	suit.pool.AbandonedConfig.RemoveAbandonedOnMaintenance = true
	suit.pool.AbandonedConfig.RemoveAbandonedTimeout = 1 * time.Second
	suit.pool.Config.MaxTotal = 1
	suit.pool.Config.TimeBetweenEvictionRuns = 500 * time.Millisecond
	suit.pool.StartEvictor()

	suit.NoErrorWithResult(suit.pool.BorrowObject(ctx))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	o2 := suit.NoErrorWithResult(suit.pool.borrowObject(ctx))
	elapsed := time.Since(start)

	suit.pool.ReturnObject(ctx, o2)

	suit.True(elapsed < 5*time.Second)
}

func (suit *PoolAbandonedTestSuite) TestAbandonedConfigReturnObjectError() {
	ctx := context.Background()

	obj := NewAbandonedTestObject()
	err := suit.pool.ReturnObject(ctx, obj)
	suit.Nil(err)
}
