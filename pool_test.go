package pool

import (
	"errors"
	"fmt"
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

func (this *SimpleFactory) setValid(valid bool) {
	this.evenValid = valid
	this.oddValid = valid
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
	this.Equal(expect, actual)
}

func (this *PoolTestSuite) assertNotNil(object interface{}) {
	this.NotNil(object)
}

func (this *PoolTestSuite) assertNil(object interface{}) {
	this.Nil(object)
}

func (this *PoolTestSuite) NoErrorWithResult(object interface{}, err error) (interface{}){
	this.NotNil(object)
	this.Nil(err)
	return object
}

func (this *PoolTestSuite) ErrorWithResult(object interface{}, err error) (error){
	this.Nil(object)
	this.NotNil(err)
	return err
}

func TestPoolTestSuite(t *testing.T) {
	suite.Run(t, new(PoolTestSuite))
}

func (this *PoolTestSuite) SetupTest() {
	this.makeEmptyPool(DEFAULT_MAX_TOTAL)
}

func (this *PoolTestSuite) TearDownTest() {
	this.pool.Clear()
	this.pool.Close()
	this.pool = nil
	this.factory = nil
}

func (this *PoolTestSuite) makeEmptyPool(maxTotal int) {
	this.factory = NewSimpleFactory()
	this.pool = NewObjectPoolWithDefaultConfig(this.factory)
	this.pool.Config.MaxTotal = maxTotal
}

func getNthObject(num int) *TestObject {
	return &TestObject{num: num}
}

func (this *PoolTestSuite) TestBaseBorrow() {
	this.pool.Config.MaxTotal = 3
	o0, err := this.pool.BorrowObject()

	this.Nil(err)
	this.NotNil(o0)

	this.Equal(getNthObject(0), o0)
	o1, _ := this.pool.BorrowObject()
	this.Equal(getNthObject(1), o1)
	o2, _ := this.pool.BorrowObject()
	this.Equal(getNthObject(2), o2)
}

func (this *PoolTestSuite) TestBaseAddObject() {
	this.pool.Config.MaxTotal = 3
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
	this.pool.Config.MaxTotal = 3

	obj0 := this.NoErrorWithResult(this.pool.BorrowObject())
	this.assertEquals(getNthObject(0), obj0)
	obj1 := this.NoErrorWithResult(this.pool.BorrowObject())
	this.assertEquals(getNthObject(1), obj1)
	obj2 := this.NoErrorWithResult(this.pool.BorrowObject())
	this.assertEquals(getNthObject(2), obj2)

	this.NoError(this.pool.ReturnObject(obj2))

	obj2 = this.NoErrorWithResult(this.pool.BorrowObject())
	this.assertEquals(getNthObject(2), obj2)
	this.pool.ReturnObject(obj1)
	obj1 = this.NoErrorWithResult(this.pool.BorrowObject())

	this.assertEquals(getNthObject(1), obj1)
	this.pool.ReturnObject(obj0)
	this.pool.ReturnObject(obj2)
	obj2 = this.NoErrorWithResult(this.pool.BorrowObject())
	if this.isLifo() {
		this.assertEquals(getNthObject(2), obj2)
	}
	if this.isFifo() {
		this.assertEquals(getNthObject(0), obj2)
	}

	obj0 = this.NoErrorWithResult(this.pool.BorrowObject())
	if this.isLifo() {
		this.assertEquals(getNthObject(0), obj0)
	}
	if this.isFifo() {
		this.assertEquals(getNthObject(2), obj0)
	}
}

func (this *PoolTestSuite) TestBaseNumActiveNumIdle() {
	this.pool.Config.MaxTotal = 3

	this.assertEquals(0, this.pool.GetNumActive())
	this.assertEquals(0, this.pool.GetNumIdle())
	obj0 := this.NoErrorWithResult(this.pool.BorrowObject())
	this.assertEquals(1, this.pool.GetNumActive())
	this.assertEquals(0, this.pool.GetNumIdle())
	obj1 := this.NoErrorWithResult(this.pool.BorrowObject())
	this.assertEquals(2, this.pool.GetNumActive())
	this.assertEquals(0, this.pool.GetNumIdle())
	this.pool.ReturnObject(obj1)
	this.assertEquals(1, this.pool.GetNumActive())
	this.assertEquals(1, this.pool.GetNumIdle())
	this.NoError(this.pool.ReturnObject(obj0))
	this.assertEquals(0, this.pool.GetNumActive())
	this.assertEquals(2, this.pool.GetNumIdle())
}

func (this *PoolTestSuite) TestBaseClear() {
	this.pool.Config.MaxTotal = 3

	this.assertEquals(0, this.pool.GetNumActive())
	this.assertEquals(0, this.pool.GetNumIdle())
	obj0  := this.NoErrorWithResult(this.pool.BorrowObject())
	obj1 := this.NoErrorWithResult(this.pool.BorrowObject())
	this.assertEquals(2, this.pool.GetNumActive())
	this.assertEquals(0, this.pool.GetNumIdle())
	this.pool.ReturnObject(obj1)
	this.pool.ReturnObject(obj0)
	this.assertEquals(0, this.pool.GetNumActive())
	this.assertEquals(2, this.pool.GetNumIdle())
	this.pool.Clear()
	this.assertEquals(0, this.pool.GetNumActive())
	this.assertEquals(0, this.pool.GetNumIdle())
	obj2  := this.NoErrorWithResult(this.pool.BorrowObject())
	this.assertEquals(getNthObject(2), obj2)
}

func (this *PoolTestSuite) TestBaseInvalidateObject() {
	this.pool.Config.MaxTotal = 3

	this.assertEquals(0, this.pool.GetNumActive())
	this.assertEquals(0, this.pool.GetNumIdle())
	obj0 := this.NoErrorWithResult(this.pool.BorrowObject())
	obj1 := this.NoErrorWithResult(this.pool.BorrowObject())
	this.assertEquals(2, this.pool.GetNumActive())
	this.assertEquals(0, this.pool.GetNumIdle())
	err := this.pool.InvalidateObject(obj0)
	this.NoError(err)
	this.assertEquals(1, this.pool.GetNumActive())
	this.assertEquals(0, this.pool.GetNumIdle())
	err = this.pool.InvalidateObject(obj1)
	this.NoError(err)
	this.assertEquals(0, this.pool.GetNumActive())
	this.assertEquals(0, this.pool.GetNumIdle())
}

func (this *PoolTestSuite) TestBaseClosePool() {
	this.pool.Config.MaxTotal = 3

	obj, err := this.pool.BorrowObject()
	this.NoError(err)
	this.pool.ReturnObject(obj)

	this.pool.Close()
	obj, err = this.pool.BorrowObject()
	this.NotNil(err)
	this.Nil(obj)
}

func (this *PoolTestSuite) TestWhenExhaustedFail() {
	this.pool.Config.MaxTotal = 1

	this.pool.Config.BlockWhenExhausted = false
	obj1 := this.NoErrorWithResult(this.pool.BorrowObject())

	err2 := this.ErrorWithResult(this.pool.BorrowObject())
	//TODO check error type
	this.assertNotNil(err2)

	this.pool.ReturnObject(obj1)
	this.assertEquals(1, this.pool.GetNumIdle())
}

func (this *PoolTestSuite) TestWhenExhaustedBlock() {
	this.pool.Config.MaxTotal = 1

	this.pool.Config.BlockWhenExhausted = true
	this.pool.Config.MaxWaitMillis = int64(10)
	obj1 := this.NoErrorWithResult(this.pool.BorrowObject())

	err2 := this.ErrorWithResult(this.pool.BorrowObject())
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
	this.pool.Config.MaxTotal = 1

	this.pool.Config.BlockWhenExhausted = true
	this.pool.Config.MaxWaitMillis = int64(-1)

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
	this.True(borrowTime >= 200)

	// Check thread was interrupted
	//assertTrue(wtt._thrown instanceof InterruptedException);

	// Return object to the pool
	this.pool.ReturnObject(obj1)

	// Bug POOL-162 - check there is now an object in the pool
	this.pool.Config.MaxWaitMillis = int64(10)
	obj2 := this.NoErrorWithResult(this.pool.BorrowObject())
	this.pool.ReturnObject(obj2)

}

func (this *PoolTestSuite) TestEvictWhileEmpty() {

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
	return NewTestThread(pool, iter, delay, delay, randomDelay, nil)
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

	this.factory.makeLatency = 300
	this.factory.maxTotal = 2
	this.pool.Config.MaxTotal = 2
	this.pool.Config.MinIdle = 1
	this.pool.BorrowObject() // numActive = 1, numIdle = 0
	// Create a test thread that will run once and try a borrow after
	// 150ms fixed delay
	borrower := NewTestThreadSimple(this.pool, 1, 150, false)
	borrowerThread := NewThreadWithRunnable(borrower)
	//// Set evictor to run in 100 ms - will create idle instance
	this.pool.Config.TimeBetweenEvictionRunsMillis = int64(100)
	borrowerThread.Start() // Off to the races
	borrowerThread.Join()
	fmt.Printf("TestEvictAddObjects %v error:%v", borrower, borrower.error)
	this.True(!borrower.failed)
}

func (this *PoolTestSuite) TestEvictLIFO() {
	this.checkEvict(true)
}

func (this *PoolTestSuite) TestEvictFIFO() {
	this.checkEvict(false)
}

func (this *PoolTestSuite) checkEvict(lifo bool) {
	var idle int
	// yea this is hairy but it tests all the code paths in GOP.evict()
	this.pool.Config.SoftMinEvictableIdleTimeMillis = int64(10)
	this.pool.Config.MinIdle = 2
	this.pool.Config.TestWhileIdle = true
	this.pool.Config.Lifo = lifo
	Prefill(this.pool, 5)
	this.pool.evict()
	idle = this.pool.GetNumIdle()
	fmt.Printf("checkEvict lifo:%v idel:%v \n", lifo, idle)
	this.factory.evenValid = false
	this.factory.oddValid = false
	this.factory.exceptionOnActivate = true
	this.pool.evict()
	idle = this.pool.GetNumIdle()
	fmt.Printf("checkEvict lifo:%v idel:%v \n", lifo, idle)
	Prefill(this.pool, 5)
	this.factory.exceptionOnActivate = false
	this.factory.exceptionOnPassivate = true
	this.pool.evict()
	idle = this.pool.GetNumIdle()
	fmt.Printf("checkEvict lifo:%v idel:%v \n", lifo, idle)
	this.factory.exceptionOnPassivate = false
	this.factory.evenValid = true
	this.factory.oddValid = true
	time.Sleep(time.Duration(125) * time.Millisecond)
	this.pool.evict()
	idle = this.pool.GetNumIdle()
	fmt.Printf("checkEvict lifo:%v idel:%v \n", lifo, idle)
	this.assertEquals(2, this.pool.GetNumIdle())
}

func (this *PoolTestSuite) TestEvictionOrder() {
	this.checkEvictionOrder(false)
	this.TearDownTest()
	this.SetupTest()
	this.checkEvictionOrder(true)
}

func (this *PoolTestSuite) checkEvictionOrder(lifo bool) {
	this.checkEvictionOrderPart1(lifo)
	this.TearDownTest()
	this.SetupTest()
	this.checkEvictionOrderPart2(lifo)
}

func (this *PoolTestSuite) checkEvictionOrderPart1(lifo bool) {
	this.pool.Config.NumTestsPerEvictionRun = 2
	this.pool.Config.MinEvictableIdleTimeMillis = 100
	this.pool.Config.Lifo = lifo
	for i := 0; i < 5; i++ {
		this.pool.AddObject()
		time.Sleep(100)
	}
	// Order, oldest to youngest, is "0", "1", ...,"4"
	this.pool.evict() // Should evict "0" and "1"
	obj, _ := this.pool.BorrowObject()
	this.True(getNthObject(0) != obj, "oldest not evicted")
	this.True(getNthObject(1) != obj, "second oldest not evicted")
	// 2 should be next out for FIFO, 4 for LIFO
	var expect *TestObject
	if lifo {
		expect = getNthObject(4)
	} else {
		expect = getNthObject(2)
	}
	this.Equal(expect, obj, "Wrong instance returned")
}

func (this *PoolTestSuite) checkEvictionOrderPart2(lifo bool) {
	// Two eviction runs in sequence
	this.pool.Config.NumTestsPerEvictionRun = 2
	this.pool.Config.MinEvictableIdleTimeMillis = int64(100)
	this.pool.Config.Lifo = lifo
	for i := 0; i < 5; i++ {
		this.pool.AddObject()
		time.Sleep(100)
	}
	this.pool.evict() // Should evict "0" and "1"
	this.pool.evict() // Should evict "2" and "3"
	obj, _ := this.pool.BorrowObject()
	this.Equal(getNthObject(4), obj, "Wrong instance remaining in pool")
}

func (this *PoolTestSuite) TestEvictorVisiting() {
	this.checkEvictorVisiting(true)
	this.checkEvictorVisiting(false)
}

func (this *PoolTestSuite) checkEvictorVisiting(lifo bool) {
	//TODO
}

func (this *PoolTestSuite) TestExceptionOnPassivateDuringReturn() {
	obj, _ := this.pool.BorrowObject()
	this.factory.exceptionOnPassivate = true
	this.pool.ReturnObject(obj)
	this.assertEquals(0, this.pool.GetNumIdle())
}

func (this *PoolTestSuite) TestExceptionOnDestroyDuringBorrow() {
	this.factory.exceptionOnDestroy = true
	this.pool.Config.TestOnBorrow = true
	this.pool.BorrowObject()
	this.factory.setValid(false) // Make validation fail on next borrow attempt
	_, err := this.pool.BorrowObject()
	this.NotNil(err)
	this.assertEquals(1, this.pool.GetNumActive())
	this.assertEquals(0, this.pool.GetNumIdle())
}

func (this *PoolTestSuite) TestExceptionOnDestroyDuringReturn() {
	this.factory.exceptionOnDestroy = true
	this.pool.Config.TestOnReturn = true
	obj1, _ := this.pool.BorrowObject()
	this.pool.BorrowObject()
	this.factory.setValid(false) // Make validation fail
	this.pool.ReturnObject(obj1)
	this.assertEquals(1, this.pool.GetNumActive())
	this.assertEquals(0, this.pool.GetNumIdle())
}

func (this *PoolTestSuite) TestExceptionOnActivateDuringBorrow() {
	obj1, _ := this.pool.BorrowObject()
	obj2, _ := this.pool.BorrowObject()
	this.pool.ReturnObject(obj1)
	this.pool.ReturnObject(obj2)
	this.factory.exceptionOnActivate = true
	this.factory.evenValid = false
	// Activation will now throw every other time
	// First attempt throws, but loop continues and second succeeds
	obj, _ := this.pool.BorrowObject()
	this.assertEquals(1, this.pool.GetNumActive())
	this.assertEquals(0, this.pool.GetNumIdle())

	this.pool.ReturnObject(obj)
	this.factory.setValid(false)
	// Validation will now fail on activation when borrowObject returns
	// an idle instance, and then when attempting to create a new instance
	_, err := this.pool.BorrowObject()
	//TODO check error type
	this.NotNil(err)

	this.assertEquals(0, this.pool.GetNumActive())
	this.assertEquals(0, this.pool.GetNumIdle())
}

func (this *PoolTestSuite) TestNegativeMaxTotal() {
	this.pool.Config.MaxTotal = -1
	this.pool.Config.BlockWhenExhausted = false
	obj, _ := this.pool.BorrowObject()
	this.assertEquals(getNthObject(0), obj)
	this.pool.ReturnObject(obj)
}

func (this *PoolTestSuite) TestMaxIdle() {
	this.pool.Config.MaxTotal = 100
	this.pool.Config.MaxIdle = 8
	active := make([]*TestObject, 100)
	for i := 0; i < 100; i++ {
		obj, err := this.pool.BorrowObject()
		this.NoError(err)
		testObj := obj.(*TestObject)
		this.NotNil(testObj)
		active[i] = testObj
	}
	this.assertEquals(100, this.pool.GetNumActive())
	this.assertEquals(0, this.pool.GetNumIdle())
	for i := 0; i < 100; i++ {
		obj := active[i]
		fmt.Printf("TestMaxIdle ReturnObject %v \n", obj)
		err := this.pool.ReturnObject(obj)
		this.NoError(err)
		this.assertEquals(99-i, this.pool.GetNumActive())
		idle := this.pool.Config.MaxIdle
		if i < idle {
			idle = i + 1
		}
		this.assertEquals(idle, this.pool.GetNumIdle())
	}
}

func (this *PoolTestSuite) TestMaxIdleZero() {
	this.pool.Config.MaxTotal = 100
	this.pool.Config.MaxIdle = 0
	active := make([]*TestObject, 100)
	for i := 0; i < 100; i++ {
		obj, err := this.pool.BorrowObject()
		this.NoError(err)
		testObj := obj.(*TestObject)
		this.NotNil(testObj)
		active[i] = testObj
	}
	this.assertEquals(100, this.pool.GetNumActive())
	this.assertEquals(0, this.pool.GetNumIdle())
	for i := 0; i < 100; i++ {
		this.pool.ReturnObject(active[i])
		this.assertEquals(99-i, this.pool.GetNumActive())
		this.assertEquals(0, this.pool.GetNumIdle())
	}
}

func (this *PoolTestSuite) TestMaxTotal() {
	this.pool.Config.MaxTotal = 3
	this.pool.Config.BlockWhenExhausted = false

	this.NoErrorWithResult(this.pool.BorrowObject())
	this.NoErrorWithResult(this.pool.BorrowObject())
	this.NoErrorWithResult(this.pool.BorrowObject())
	_, err := this.pool.BorrowObject()
	this.Error(err)
}

func (this *PoolTestSuite) TestTimeoutNoLeak() {
	this.pool.Config.MaxTotal = 2
	this.pool.Config.MaxWaitMillis = int64(10)
	this.pool.Config.BlockWhenExhausted = true
	obj, err := this.pool.BorrowObject()
	this.NoError(err)
	obj2 := this.NoErrorWithResult(this.pool.BorrowObject())
	err3 := this.ErrorWithResult(this.pool.BorrowObject())
	//TODO check error type
	this.Error(err3)
	//fail("Expecting NoSuchElementException");
	this.NoError(this.pool.ReturnObject(obj2))
	this.NoError(this.pool.ReturnObject(obj))

	this.NoErrorWithResult(this.pool.BorrowObject())
	this.NoErrorWithResult(this.pool.BorrowObject())
}
