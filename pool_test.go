package pool

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"math"
	"math/rand"
	"sync"
	"testing"
	"time"
)

type TestObject struct {
	num int
}

type SimpleFactory struct {
	makeCounter          int
	activationCounter    int
	validateCounter      int
	activeCount          int
	evenValid            bool
	oddValid             bool
	exceptionOnPassivate bool
	exceptionOnActivate  bool
	exceptionOnDestroy   bool
	enableValidation     bool
	destroyLatency       int64
	makeLatency          int64
	validateLatency      int64
	maxTotal             int
	lock                 sync.Mutex
}

func NewSimpleFactory() *SimpleFactory {
	return &SimpleFactory{maxTotal: math.MaxInt32, evenValid: true, oddValid: true, enableValidation: true}
}

func (this *SimpleFactory) doWait(latency time.Duration) {
	time.Sleep(latency)
}

func (this *SimpleFactory) MakeObject() (*PooledObject, error) {
	fmt.Println("factory MakeObject")
	var waitLatency int64
	this.lock.Lock()
	this.activeCount = this.activeCount + 1
	if this.activeCount > this.maxTotal {
		return nil, fmt.Errorf("Too many active instances: %v", this.activeCount)
	}
	waitLatency = this.makeLatency
	this.lock.Unlock()
	if waitLatency > 0 {
		this.doWait(time.Duration(waitLatency))
	}
	var counter int
	this.lock.Lock()
	counter = this.makeCounter
	this.makeCounter = this.makeCounter + 1
	this.lock.Unlock()
	return NewPooledObject(&TestObject{num: counter}), nil
}

func (this *SimpleFactory) DestroyObject(object *PooledObject) error {
	fmt.Println("factory DestroyObject")
	var waitLatency int64
	var hurl bool
	this.lock.Lock()
	waitLatency = this.destroyLatency
	hurl = this.exceptionOnDestroy
	this.lock.Unlock()
	if waitLatency > 0 {
		this.doWait(time.Duration(waitLatency))
	}
	this.lock.Lock()
	this.activeCount = this.activeCount - 1
	this.lock.Unlock()
	if hurl {
		return errors.New("destroy error")
	}
	return nil
}

func (this *SimpleFactory) ValidateObject(object *PooledObject) bool {
	fmt.Println("factory ValidateObject")
	var validate bool
	var evenTest bool
	var oddTest bool
	var waitLatency int64
	var counter int
	this.lock.Lock()
	validate = this.enableValidation
	evenTest = this.evenValid
	oddTest = this.oddValid
	counter = this.validateCounter
	this.validateCounter = this.validateCounter + 1
	waitLatency = this.validateLatency
	this.lock.Unlock()
	if waitLatency > 0 {
		this.doWait(time.Duration(waitLatency))
	}
	if validate {
		if counter%2 == 0 {
			return evenTest
		} else {
			return oddTest
		}
	}
	return true
}

func (this *SimpleFactory) ActivateObject(object *PooledObject) error {
	fmt.Println("factory ActivateObject")
	defer fmt.Println("factory ActivateObject end")
	var hurl bool
	var evenTest bool
	var oddTest bool
	var counter int
	this.lock.Lock()
	hurl = this.exceptionOnActivate
	evenTest = this.evenValid
	oddTest = this.oddValid
	counter = this.activationCounter
	this.activationCounter = this.activationCounter + 1
	this.lock.Unlock()
	if hurl {
		var test bool
		if counter%2 == 0 {
			test = evenTest
		} else {
			test = oddTest
		}
		if !test {
			return errors.New("activate error")
		}
	}
	return nil
}

func (this *SimpleFactory) PassivateObject(object *PooledObject) error {
	fmt.Println("factory PassivateObject")
	var hurl bool
	this.lock.Lock()
	hurl = this.exceptionOnPassivate
	this.lock.Unlock()
	if hurl {
		return errors.New("passivate error")
	}
	return nil
}

type PoolTestSuite struct {
	suite.Suite
	pool    *ObjectPool
	factory *SimpleFactory
}

func (this *PoolTestSuite) assertEquals(expect interface{}, actual interface{}) {
	assert.Equal(this.T(), expect, actual)
}

func (this *PoolTestSuite) assertNotNil(object interface{}) {
	assert.NotNil(this.T(), object)
}

func (this *PoolTestSuite) assertNil(object interface{}) {
	assert.Nil(this.T(), object)
}

func (this *PoolTestSuite) tryFail(err error) {
	if err != nil {
		this.Fail(err.Error())
	}
}

func TestPoolTestSuite(t *testing.T) {
	suite.Run(t, new(PoolTestSuite))
}

func (this *PoolTestSuite) SetupTest() {

}

func (this *PoolTestSuite) TearDownTest() {
	this.pool.Close()
}

func (this *PoolTestSuite) makeEmptyPool(maxTotal int) {
	this.factory = NewSimpleFactory()
	this.pool = NewObjectPoolWithDefaultConfig(this.factory)
	this.pool.PoolConfig.MaxTotal = maxTotal
}

func getNthObject(num int) *TestObject {
	return &TestObject{num: num}
}

func (this *PoolTestSuite) TestBaseBorrow() {
	this.makeEmptyPool(3)

	o0, err := this.pool.BorrowObject()

	assert.Nil(this.T(), err)
	assert.NotNil(this.T(), o0)

	assert.Equal(this.T(), getNthObject(0), o0)
	o1, _ := this.pool.BorrowObject()
	assert.Equal(this.T(), getNthObject(1), o1)
	o2, _ := this.pool.BorrowObject()
	assert.Equal(this.T(), getNthObject(2), o2)
}

func (this *PoolTestSuite) TestBaseAddObject() {
	this.makeEmptyPool(3)
	this.assertEquals(0, this.pool.GetNumIdle())
	this.assertEquals(0, this.pool.GetNumActive())
	fmt.Println("test AddObject")
	this.pool.AddObject()

	this.assertEquals(1, this.pool.GetNumIdle())
	this.assertEquals(0, this.pool.GetNumActive())

	fmt.Println("test BorrowObject")
	obj, err := this.pool.BorrowObject()
	if err != nil {
		this.Fail(err.Error())
	}

	this.assertEquals(getNthObject(0), obj)
	this.assertEquals(0, this.pool.GetNumIdle())
	this.assertEquals(1, this.pool.GetNumActive())
	err = this.pool.ReturnObject(obj)
	if err != nil {
		this.Fail(err.Error())
	}
	this.assertEquals(1, this.pool.GetNumIdle())
	this.assertEquals(0, this.pool.GetNumActive())
}

func (this *PoolTestSuite) isLifo() bool {
	return true
}

func (this *PoolTestSuite) isFifo() bool {
	return false
}

func (this *PoolTestSuite) TestBaseBorrowReturn() {
	this.makeEmptyPool(3)

	obj0, err0 := this.pool.BorrowObject()
	this.tryFail(err0)
	this.assertEquals(getNthObject(0), obj0)
	obj1, err1 := this.pool.BorrowObject()
	this.tryFail(err1)
	this.assertEquals(getNthObject(1), obj1)
	obj2, err2 := this.pool.BorrowObject()
	this.tryFail(err2)
	this.assertEquals(getNthObject(2), obj2)
	this.pool.ReturnObject(obj2)
	obj2, err2 = this.pool.BorrowObject()
	this.assertEquals(getNthObject(2), obj2)
	this.pool.ReturnObject(obj1)
	obj1, err1 = this.pool.BorrowObject()
	this.tryFail(err1)
	this.assertEquals(getNthObject(1), obj1)
	this.pool.ReturnObject(obj0)
	this.pool.ReturnObject(obj2)
	obj2, err2 = this.pool.BorrowObject()
	this.tryFail(err2)
	if this.isLifo() {
		this.assertEquals(getNthObject(2), obj2)
	}
	if this.isFifo() {
		this.assertEquals(getNthObject(0), obj2)
	}

	obj0, err0 = this.pool.BorrowObject()
	this.tryFail(err0)
	if this.isLifo() {
		this.assertEquals(getNthObject(0), obj0)
	}
	if this.isFifo() {
		this.assertEquals(getNthObject(2), obj0)
	}
}

func (this *PoolTestSuite) TestBaseNumActiveNumIdle() {
	this.makeEmptyPool(3)

	this.assertEquals(0, this.pool.GetNumActive())
	this.assertEquals(0, this.pool.GetNumIdle())
	obj0, err0 := this.pool.BorrowObject()
	this.tryFail(err0)
	this.assertEquals(1, this.pool.GetNumActive())
	this.assertEquals(0, this.pool.GetNumIdle())
	obj1, err1 := this.pool.BorrowObject()
	this.tryFail(err1)
	this.assertEquals(2, this.pool.GetNumActive())
	this.assertEquals(0, this.pool.GetNumIdle())
	this.pool.ReturnObject(obj1)
	this.assertEquals(1, this.pool.GetNumActive())
	this.assertEquals(1, this.pool.GetNumIdle())
	err := this.pool.ReturnObject(obj0)
	this.tryFail(err)
	this.assertEquals(0, this.pool.GetNumActive())
	this.assertEquals(2, this.pool.GetNumIdle())
}

func (this *PoolTestSuite) TestBaseClear() {
	this.makeEmptyPool(3)

	this.assertEquals(0, this.pool.GetNumActive())
	this.assertEquals(0, this.pool.GetNumIdle())
	obj0, err0 := this.pool.BorrowObject()
	this.tryFail(err0)
	obj1, err1 := this.pool.BorrowObject()
	this.tryFail(err1)
	this.assertEquals(2, this.pool.GetNumActive())
	this.assertEquals(0, this.pool.GetNumIdle())
	this.pool.ReturnObject(obj1)
	this.pool.ReturnObject(obj0)
	this.assertEquals(0, this.pool.GetNumActive())
	this.assertEquals(2, this.pool.GetNumIdle())
	this.pool.Clear()
	this.assertEquals(0, this.pool.GetNumActive())
	this.assertEquals(0, this.pool.GetNumIdle())
	obj2, err2 := this.pool.BorrowObject()
	this.tryFail(err2)
	this.assertEquals(getNthObject(2), obj2)
}

func (this *PoolTestSuite) TestBaseInvalidateObject() {
	this.makeEmptyPool(3)

	this.assertEquals(0, this.pool.GetNumActive())
	this.assertEquals(0, this.pool.GetNumIdle())
	obj0, err0 := this.pool.BorrowObject()
	this.tryFail(err0)
	obj1, err1 := this.pool.BorrowObject()
	this.tryFail(err1)
	this.assertEquals(2, this.pool.GetNumActive())
	this.assertEquals(0, this.pool.GetNumIdle())
	err := this.pool.InvalidateObject(obj0)
	this.tryFail(err)
	this.assertEquals(1, this.pool.GetNumActive())
	this.assertEquals(0, this.pool.GetNumIdle())
	err = this.pool.InvalidateObject(obj1)
	this.tryFail(err)
	this.assertEquals(0, this.pool.GetNumActive())
	this.assertEquals(0, this.pool.GetNumIdle())
}

func (this *PoolTestSuite) TestBaseClosePool() {
	this.makeEmptyPool(3)

	obj, err := this.pool.BorrowObject()
	this.tryFail(err)
	this.pool.ReturnObject(obj)

	this.pool.Close()
	obj, err = this.pool.BorrowObject()
	assert.NotNil(this.T(), err)
	assert.Nil(this.T(), obj)
}

func (this *PoolTestSuite) TestWhenExhaustedFail() {
	this.makeEmptyPool(1)

	this.pool.PoolConfig.BlockWhenExhausted = false
	obj1, err1 := this.pool.BorrowObject()
	this.tryFail(err1)
	this.assertNotNil(obj1)

	obj2, err2 := this.pool.BorrowObject()
	this.assertNil(obj2)
	//TODO check error type
	this.assertNotNil(err2)

	this.pool.ReturnObject(obj1)
	this.assertEquals(1, this.pool.GetNumIdle())
}

func (this *PoolTestSuite) TestWhenExhaustedBlock() {
	this.makeEmptyPool(1)

	this.pool.PoolConfig.BlockWhenExhausted = true
	this.pool.PoolConfig.MaxWaitMillis = int64(10)
	obj1, err1 := this.pool.BorrowObject()
	this.tryFail(err1)
	this.assertNotNil(obj1)
	obj2, err2 := this.pool.BorrowObject()
	this.assertNil(obj2)
	//TODO check error type
	this.assertNotNil(err2)

	this.pool.ReturnObject(obj1)
}

func borrowAndWait(pool *ObjectPool, pause time.Duration) chan int {
	ch := make(chan int, 1)
	go func() {
		preborrow := currentTimeMillis()
		obj, _ := pool.BorrowObject()
		//objectId = obj;
		postborrow := currentTimeMillis()
		ch <- int(postborrow - preborrow)
		time.Sleep(pause)
		if obj != nil {
			pool.ReturnObject(obj)
		}
		//postreturn = System.currentTimeMillis();
	}()
	return ch
}

func (this *PoolTestSuite) TestWhenExhaustedBlockInterrupt() {
	this.makeEmptyPool(1)

	this.pool.PoolConfig.BlockWhenExhausted = true
	this.pool.PoolConfig.MaxWaitMillis = int64(-1)

	obj1, _ := this.pool.BorrowObject()

	// Make sure on object was obtained
	this.assertNotNil(obj1)

	// Create a separate thread to try and borrow another object
	//WaitingTestThread wtt = new WaitingTestThread(pool, 200000);
	ch := borrowAndWait(this.pool, time.Duration(200000)*time.Millisecond)

	// Give wtt time to start
	time.Sleep(time.Duration(200) * time.Millisecond)

	this.pool.idleObjects.InterruptTakeWaiters()

	// Give interrupt time to take effect
	time.Sleep(time.Duration(200) * time.Millisecond)

	borrowTime := <-ch
	fmt.Println("TestWhenExhaustedBlockInterrupt borrowTime:", borrowTime)
	assert.True(this.T(), borrowTime > 200)

	// Check thread was interrupted
	//assertTrue(wtt._thrown instanceof InterruptedException);

	// Return object to the pool
	this.pool.ReturnObject(obj1)

	// Bug POOL-162 - check there is now an object in the pool
	this.pool.PoolConfig.MaxWaitMillis = int64(10)
	obj2, err2 := this.pool.BorrowObject()
	this.tryFail(err2)
	this.assertNotNil(obj2)
	this.pool.ReturnObject(obj2)

}

func (this *PoolTestSuite) TestEvictWhileEmpty() {
	this.makeEmptyPool(DEFAULT_MAX_TOTAL)

	this.pool.evict()
	this.pool.evict()
}

type TestRunnable struct {
	/** pool to borrow from */
	pool *ObjectPool

	/** number of borrow attempts */
	iter int

	/** delay before each borrow attempt */
	startDelay int

	/** time to hold each borrowed object before returning it */
	holdTime int

	/** whether or not start and hold time are randomly generated */
	randomDelay bool

	/** object expected to be borrowed (fail otherwise) */
	expectedObject interface{}

	complete bool
	failed   bool
	error    error
}

func NewTestThreadSimple(pool *ObjectPool, iter int, delay int, randomDelay bool) *TestRunnable {
	return NewTestThread(pool,iter, delay,delay,randomDelay, nil)
}

func NewTestThread(pool *ObjectPool, iter int, startDelay int,
holdTime int, randomDelay bool, obj interface{}) *TestRunnable {
	return &TestRunnable{pool: pool, iter: iter, startDelay: startDelay, holdTime: holdTime, randomDelay: randomDelay, expectedObject: obj}
}

func (this *TestRunnable) Run() {
	for i := 0; i < this.iter; i++ {
		var startDelay int
		if this.randomDelay {
			startDelay = int(rand.Int31n(int32(this.startDelay)))
		} else {
			startDelay = this.startDelay
		}
		var holdTime int
		if this.randomDelay {
			holdTime = int(rand.Int31n(int32(this.holdTime)))
		} else {
			holdTime = this.holdTime
		}
		time.Sleep(time.Duration(startDelay) * time.Millisecond)
		obj, err := this.pool.BorrowObject()
		if err != nil {
			this.error = err
			this.failed = true
			this.complete = true
			break
		}

		if this.expectedObject != nil && !(this.expectedObject == obj) {
			this.error = fmt.Errorf("Expected: %v found: %v", this.expectedObject, obj)
			this.failed = true
			this.complete = true
			break
		}
		time.Sleep(time.Duration(holdTime) * time.Millisecond)
		err = this.pool.ReturnObject(obj)
		if err != nil {
			this.error = err
			this.failed = true
			this.complete = true
			break
		}
	}
	this.complete = true
}

func (this *PoolTestSuite) TestEvictAddObjects() {
	this.makeEmptyPool(DEFAULT_MAX_TOTAL)

	this.factory.makeLatency = 300
	this.factory.maxTotal = 2
	this.pool.PoolConfig.MaxTotal = 2
	this.pool.PoolConfig.MinIdle = 1
	this.pool.BorrowObject() // numActive = 1, numIdle = 0
	// Create a test thread that will run once and try a borrow after
	// 150ms fixed delay
	 borrower := NewTestThreadSimple(this.pool, 1, 150, false)
	borrowerThread := NewThreadWithRunnable(borrower)
	//// Set evictor to run in 100 ms - will create idle instance
	this.pool.PoolConfig.TimeBetweenEvictionRunsMillis = int64(100)
	borrowerThread.Start();  // Off to the races
	borrowerThread.Join();
	fmt.Printf("TestEvictAddObjects %v error:%v",borrower,borrower.error )
	assert.True(this.T(),!borrower.failed)
}
