package pool

import (
	"errors"
	"fmt"
	"github.com/jolestar/go-commons-pool/collections"
	"github.com/jolestar/go-commons-pool/concurrent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"math"
	"math/rand"
	"reflect"
	"sync"
	"testing"
	"time"
)

type TestObject struct {
	Num int
}

const (
	debug_simple_facotry = false
)

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
	this.lock.Lock()
	this.evenValid = valid
	this.oddValid = valid
	this.lock.Unlock()
}

func (this *SimpleFactory) setValidateLatency(validateLatency int64) {
	this.lock.Lock()
	this.validateLatency = validateLatency
	this.lock.Unlock()
}

func (this *SimpleFactory) doWait(latencyMillisecond int64) {
	time.Sleep(time.Duration(latencyMillisecond) * time.Millisecond)
}

func (this *SimpleFactory) MakeObject() (*PooledObject, error) {
	if debug_simple_facotry {
		fmt.Println("factory MakeObject")
	}
	var waitLatency int64
	this.lock.Lock()
	this.activeCount = this.activeCount + 1
	if this.activeCount > this.maxTotal {
		return nil, fmt.Errorf("Too many active instances: %v", this.activeCount)
	}
	waitLatency = this.makeLatency
	this.lock.Unlock()
	if waitLatency > 0 {
		this.doWait(waitLatency)
	}
	var counter int
	this.lock.Lock()
	counter = this.makeCounter
	this.makeCounter = this.makeCounter + 1
	this.lock.Unlock()
	return NewPooledObject(&TestObject{Num: counter}), nil
}

func (this *SimpleFactory) DestroyObject(object *PooledObject) error {
	if debug_simple_facotry {
		fmt.Println("factory DestroyObject")
	}
	var waitLatency int64
	var hurl bool
	this.lock.Lock()
	waitLatency = this.destroyLatency
	hurl = this.exceptionOnDestroy
	this.lock.Unlock()
	if waitLatency > 0 {
		this.doWait(waitLatency)
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
	if debug_simple_facotry {
		fmt.Println("factory ValidateObject")
	}
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
		this.doWait(waitLatency)
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
	if debug_simple_facotry {
		fmt.Println("factory ActivateObject")
		defer fmt.Println("factory ActivateObject end")
	}
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
	if debug_simple_facotry {
		fmt.Println("factory PassivateObject")
	}
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

func (this *PoolTestSuite) NoErrorWithResult(object interface{}, err error) interface{} {
	this.NotNil(object)
	this.Nil(err)
	return object
}

func (this *PoolTestSuite) ErrorWithResult(object interface{}, err error) error {
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
	return &TestObject{Num: num}
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

func (this *PoolTestSuite) TestBorrowReturnAsync() {
	this.pool.Config.MaxTotal = 1
	this.pool.Config.MaxWaitMillis = 1000

	obj0 := this.NoErrorWithResult(this.pool.BorrowObject())
	this.assertEquals(getNthObject(0), obj0)

	//start new goroutine to borrow will block
	ch := make(chan interface{}, 1)
	go func() {
		obj, _ := this.pool.BorrowObject()
		ch <- obj
	}()
	sleep(100)

	//return obj0
	go func() {
		this.pool.ReturnObject(obj0)
	}()
	sleep(100)
	obj1 := <-ch
	this.NotNil(obj1)
	this.Equal(obj0, obj1)
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
	obj0 := this.NoErrorWithResult(this.pool.BorrowObject())
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
	obj2 := this.NoErrorWithResult(this.pool.BorrowObject())
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
	_, ok := err2.(*NoSuchElementErr)
	this.True(ok, "expect NoSuchElementErr but get", reflect.TypeOf(err2))

	this.pool.ReturnObject(obj1)
	this.assertEquals(1, this.pool.GetNumIdle())
}

func (this *PoolTestSuite) TestWhenExhaustedBlock() {
	this.pool.Config.MaxTotal = 1

	this.pool.Config.BlockWhenExhausted = true
	this.pool.Config.MaxWaitMillis = int64(10)
	obj1 := this.NoErrorWithResult(this.pool.BorrowObject())

	err2 := this.ErrorWithResult(this.pool.BorrowObject())
	_, ok := err2.(*NoSuchElementErr)
	this.True(ok, "expect NoSuchElementErr but get", reflect.TypeOf(err2))

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
	close(ch)
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

type TestThreadArg struct {
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
}

type TestThreadResult struct {
	complete   bool
	failed     bool
	error      error
	preborrow  int64
	postborrow int64
	postreturn int64
	ended      int64
	objectId   interface{}
}

func NewTesThreadArgSimple(pool *ObjectPool, iter int, delay int, randomDelay bool) *TestThreadArg {
	return NewTestThreadArg(pool, iter, delay, delay, randomDelay, nil)
}

func NewTestThreadArg(pool *ObjectPool, iter int, startDelay int,
	holdTime int, randomDelay bool, obj interface{}) *TestThreadArg {
	return &TestThreadArg{pool: pool, iter: iter, startDelay: startDelay, holdTime: holdTime, randomDelay: randomDelay, expectedObject: obj}
}

func threadRun(arg *TestThreadArg) chan TestThreadResult {
	resultChan := make(chan TestThreadResult, 1)
	result := TestThreadResult{}
	go func() {
		for i := 0; i < arg.iter; i++ {
			var startDelay int
			if arg.randomDelay {
				startDelay = int(rand.Int31n(int32(arg.startDelay)))
			} else {
				startDelay = arg.startDelay
			}
			var holdTime int
			if arg.randomDelay {
				holdTime = int(rand.Int31n(int32(arg.holdTime)))
			} else {
				holdTime = arg.holdTime
			}
			sleep(startDelay)
			startBorrow := currentTimeMillis()
			obj, err := arg.pool.BorrowObject()
			endBorrow := currentTimeMillis()
			if err != nil {
				fmt.Println("borrow error, time:", endBorrow-startBorrow)
				result.error = err
				result.failed = true
				result.complete = true
				break
			}
			if arg.expectedObject != nil && !(arg.expectedObject == obj) {
				result.error = fmt.Errorf("Expected: %v found: %v", arg.expectedObject, obj)
				result.failed = true
				result.complete = true
				break
			}
			sleep(holdTime)
			//startReturn := currentTimeMillis()
			err = arg.pool.ReturnObject(obj)
			//endReturn := currentTimeMillis()
			//fmt.Println("returnTime:", endReturn - startReturn)
			if err != nil {
				result.error = err
				result.failed = true
				result.complete = true
				break
			}
		}
		result.complete = true
		resultChan <- result
	}()
	return resultChan
}

func (this *PoolTestSuite) TestEvictAddObjects() {

	this.factory.makeLatency = 300
	this.factory.maxTotal = 2
	this.pool.Config.MaxTotal = 2
	this.pool.Config.MinIdle = 1
	this.pool.BorrowObject() // numActive = 1, numIdle = 0
	// Create a test thread that will run once and try a borrow after
	// 150ms fixed delay
	borrower := NewTesThreadArgSimple(this.pool, 1, 150, false)
	//// Set evictor to run in 100 ms - will create idle instance
	this.pool.Config.TimeBetweenEvictionRunsMillis = int64(100)
	ch := threadRun(borrower)
	result := <-ch
	close(ch)
	fmt.Printf("TestEvictAddObjects %v error:%v", borrower, result.error)
	this.True(!result.failed)
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
		time.Sleep(time.Duration(100) * time.Millisecond)
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
		time.Sleep(time.Duration(100) * time.Millisecond)
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
	_, ok := err.(*NoSuchElementErr)
	this.True(ok, "expect NoSuchElementErr but get", reflect.TypeOf(err))

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
	_, ok := err3.(*NoSuchElementErr)
	this.True(ok, "expect NoSuchElementErr but get", reflect.TypeOf(err3))

	this.NoError(this.pool.ReturnObject(obj2))
	this.NoError(this.pool.ReturnObject(obj))

	this.NoErrorWithResult(this.pool.BorrowObject())
	this.NoErrorWithResult(this.pool.BorrowObject())
}

func (this *PoolTestSuite) TestMaxTotalZero() {
	this.pool.Config.MaxTotal = 0
	this.pool.Config.BlockWhenExhausted = false
	err := this.ErrorWithResult(this.pool.BorrowObject())
	this.Error(err)
	//fail("Expected NoSuchElementException");
}

func (this *PoolTestSuite) TestMaxTotalUnderLoad() {
	// Config
	numThreads := 199 // And main thread makes a round 200.
	numIter := 20
	delay := 25
	maxTotal := 10

	this.factory.maxTotal = maxTotal
	this.pool.Config.MaxTotal = maxTotal
	this.pool.Config.BlockWhenExhausted = true
	this.pool.Config.TimeBetweenEvictionRunsMillis = int64(-1)

	// Start threads to borrow objects
	threadArgs := make([]*TestThreadArg, numThreads)
	resultChans := make([]chan TestThreadResult, numThreads)
	for i := 0; i < numThreads; i++ {
		// Factor of 2 on iterations so main thread does work whilst other
		// threads are running. Factor of 2 on delay so average delay for
		// other threads == actual delay for main thread
		threadArgs[i] = NewTesThreadArgSimple(this.pool, numIter*2, delay*2, true)
		resultChans[i] = threadRun(threadArgs[i])
	}
	// Give the threads a chance to start doing some work
	time.Sleep(time.Duration(5000) * time.Millisecond)

	for i := 0; i < numIter; i++ {
		var obj interface{}
		time.Sleep(time.Duration(delay) * time.Millisecond)

		obj, err := this.pool.BorrowObject()
		this.NoError(err)
		// Under load, observed _numActive > _maxTotal
		if this.pool.GetNumActive() > this.pool.Config.MaxTotal {
			this.Fail("Too many active objects")
		}
		time.Sleep(time.Duration(delay) * time.Millisecond)
		if obj != nil {
			this.NoError(this.pool.ReturnObject(obj))
		}
	}

	for i := 0; i < numThreads; i++ {
		result := <-resultChans[i]
		close(resultChans[i])
		if result.failed {
			this.Fail(fmt.Sprintf("Thread %v failed: %v", i, result.error.Error()))
		}
	}
}

func (this *PoolTestSuite) TestStartAndStopEvictor() {
	// set up pool without evictor
	this.pool.Config.MaxIdle = 6
	this.pool.Config.MaxTotal = 6
	this.pool.Config.NumTestsPerEvictionRun = 6
	this.pool.Config.MinEvictableIdleTimeMillis = int64(100)

	for j := 0; j < 2; j++ {
		// populate the pool
		{
			active := make([]*TestObject, 6)
			for i := 0; i < 6; i++ {
				active[i] = this.NoErrorWithResult(this.pool.BorrowObject()).(*TestObject)
			}
			for i := 0; i < 6; i++ {
				this.NoError(this.pool.ReturnObject(active[i]))
			}
		}

		// note that it stays populated
		this.Equal(6, this.pool.GetNumIdle(), "Should have 6 idle")

		// start the evictor
		this.pool.Config.TimeBetweenEvictionRunsMillis = int64(50)

		//re config evictor
		this.pool.StartEvictor()

		// wait a second (well, .2 seconds)
		time.Sleep(time.Duration(200) * time.Millisecond)

		// assert that the evictor has cleared out the pool
		this.Equal(0, this.pool.GetNumIdle(), "Should have 0 idle")

		// stop the evictor
		this.pool.startEvictor(int64(0))
	}
}

func (this *PoolTestSuite) TestEvictionWithNegativeNumTests() {
	// when numTestsPerEvictionRun is negative, it represents a fraction of the idle objects to test
	this.pool.Config.MaxIdle = 6
	this.pool.Config.MaxTotal = 6
	this.pool.Config.NumTestsPerEvictionRun = -2
	this.pool.Config.MinEvictableIdleTimeMillis = int64(50)

	this.pool.Config.TimeBetweenEvictionRunsMillis = int64(100)
	this.pool.StartEvictor()

	active := make([]*TestObject, 6)
	for i := 0; i < 6; i++ {
		active[i] = this.NoErrorWithResult(this.pool.BorrowObject()).(*TestObject)
	}
	for i := 0; i < 6; i++ {
		this.NoError(this.pool.ReturnObject(active[i]))
	}

	time.Sleep(time.Duration(100) * time.Millisecond)
	this.True(this.pool.GetNumIdle() <= 6, "Should at most 6 idle, found %v", this.pool.GetNumIdle())
	time.Sleep(time.Duration(100) * time.Millisecond)
	this.True(this.pool.GetNumIdle() <= 3, "Should at most 3 idle, found %v", this.pool.GetNumIdle())
	time.Sleep(time.Duration(100) * time.Millisecond)
	this.True(this.pool.GetNumIdle() <= 2, "Should be at most 2 idle, found %v", this.pool.GetNumIdle())
	time.Sleep(time.Duration(100) * time.Millisecond)
	this.Equal(0, this.pool.GetNumIdle(), "Should be zero idle, found %v", this.pool.GetNumIdle())
}

func (this *PoolTestSuite) TestEviction() {
	this.pool.Config.MaxIdle = 500
	this.pool.Config.MaxTotal = 500
	this.pool.Config.NumTestsPerEvictionRun = 100
	this.pool.Config.MinEvictableIdleTimeMillis = int64(250)
	this.pool.Config.TimeBetweenEvictionRunsMillis = int64(500)
	this.pool.StartEvictor()

	this.pool.Config.TestWhileIdle = true
	active := make([]*TestObject, 500)

	for i := 0; i < 500; i++ {
		active[i] = this.NoErrorWithResult(this.pool.BorrowObject()).(*TestObject)
	}
	for i := 0; i < 500; i++ {
		this.NoError(this.pool.ReturnObject(active[i]))
	}

	time.Sleep(time.Duration(1000) * time.Millisecond)
	this.True(this.pool.GetNumIdle() < 500, "Should be less than 500 idle, found %v", this.pool.GetNumIdle())
	time.Sleep(time.Duration(600) * time.Millisecond)
	this.True(this.pool.GetNumIdle() < 400, "Should be less than 400 idle, found %v", this.pool.GetNumIdle())
	time.Sleep(time.Duration(600) * time.Millisecond)
	this.True(this.pool.GetNumIdle() < 300, "Should be less than 300 idle, found %v", this.pool.GetNumIdle())
	time.Sleep(time.Duration(600) * time.Millisecond)
	this.True(this.pool.GetNumIdle() < 200, "Should be less than 200 idle, found %v", this.pool.GetNumIdle())
	time.Sleep(time.Duration(600) * time.Millisecond)
	this.True(this.pool.GetNumIdle() < 100, "Should be less than 100 idle, found %v", this.pool.GetNumIdle())
	time.Sleep(time.Duration(600) * time.Millisecond)
	this.Equal(0, this.pool.GetNumIdle(), "Should be zero idle, found %v", this.pool.GetNumIdle())

	for i := 0; i < 500; i++ {
		active[i] = this.NoErrorWithResult(this.pool.BorrowObject()).(*TestObject)
	}
	for i := 0; i < 500; i++ {
		this.NoError(this.pool.ReturnObject(active[i]))
	}

	time.Sleep(time.Duration(1000) * time.Millisecond)
	this.True(this.pool.GetNumIdle() < 500, "Should be less than 500 idle, found %v", this.pool.GetNumIdle())
	time.Sleep(time.Duration(600) * time.Millisecond)
	this.True(this.pool.GetNumIdle() < 400, "Should be less than 400 idle, found %v", this.pool.GetNumIdle())
	time.Sleep(time.Duration(600) * time.Millisecond)
	this.True(this.pool.GetNumIdle() < 300, "Should be less than 300 idle, found %v", this.pool.GetNumIdle())
	time.Sleep(time.Duration(600) * time.Millisecond)
	this.True(this.pool.GetNumIdle() < 200, "Should be less than 200 idle, found %v", this.pool.GetNumIdle())
	time.Sleep(time.Duration(600) * time.Millisecond)
	this.True(this.pool.GetNumIdle() < 100, "Should be less than 100 idle, found %v", this.pool.GetNumIdle())
	time.Sleep(time.Duration(600) * time.Millisecond)
	this.Equal(0, this.pool.GetNumIdle(), "Should be zero idle, found %v", this.pool.GetNumIdle())
}

type TestEvictionPolicy struct {
	callCount concurrent.AtomicInteger
}

func (this *TestEvictionPolicy) Evict(config *EvictionConfig, underTest *PooledObject, idleCount int) bool {
	if this.callCount.IncrementAndGet() > 1500 {
		return true
	}
	return false
}

var TestEvictionPolicyName = "github.com/jolestar/go-commons-pool/TestEvictionPolicy"

func (this *PoolTestSuite) TestEvictionPolicy() {
	this.pool.Config.MaxIdle = 500
	this.pool.Config.MaxTotal = 500
	this.pool.Config.NumTestsPerEvictionRun = 500
	this.pool.Config.MinEvictableIdleTimeMillis = int64(250)
	this.pool.Config.TimeBetweenEvictionRunsMillis = int64(500)
	this.pool.StartEvictor()
	this.pool.Config.TestWhileIdle = true
	evictionPolicy := new(TestEvictionPolicy)

	RegistryEvictionPolicy(TestEvictionPolicyName, evictionPolicy)

	_, ok := this.pool.getEvictionPolicy().(*DefaultEvictionPolicy)
	this.True(ok, "EvictionPolicy is not default policy")

	this.pool.Config.EvictionPolicyName = TestEvictionPolicyName
	this.Equal(evictionPolicy, this.pool.getEvictionPolicy())

	active := make([]*TestObject, 500)
	for i := 0; i < 500; i++ {
		active[i] = this.NoErrorWithResult(this.pool.BorrowObject()).(*TestObject)
	}
	for i := 0; i < 500; i++ {
		this.NoError(this.pool.ReturnObject(active[i]))
	}

	// Eviction policy ignores first 1500 attempts to evict and then always
	// evicts. After 1s, there should have been two runs of 500 tests so no
	// evictions
	time.Sleep(time.Duration(1000) * time.Millisecond)
	this.Equal(500, this.pool.GetNumIdle(), "Should be 500 idle")
	// A further 1s wasn't enough so allow 2s for the evictor to clear out
	// all of the idle objects.
	time.Sleep(time.Duration(2000) * time.Millisecond)
	this.Equal(0, this.pool.GetNumIdle(), "Should be 0 idle")
}

func (this *PoolTestSuite) TestEvictionSoftMinIdle() {

	this.pool.Config.MaxIdle = 5
	this.pool.Config.MaxTotal = 5
	this.pool.Config.NumTestsPerEvictionRun = 5
	this.pool.Config.MinEvictableIdleTimeMillis = int64(3000)
	this.pool.Config.SoftMinEvictableIdleTimeMillis = int64(1000)
	this.pool.Config.MinIdle = 2

	active := make([]*TestObject, 5)
	for i := 0; i < 5; i++ {
		active[i] = this.NoErrorWithResult(this.pool.BorrowObject()).(*TestObject)
	}

	for i := 0; i < 5; i++ {
		this.pool.ReturnObject(active[i])
	}

	// Soft evict all but minIdle(2)
	time.Sleep(time.Duration(1500) * time.Millisecond)
	this.pool.evict()
	this.Equal(2, this.pool.GetNumIdle(), "Idle count different than expected.")

	// Hard evict the rest.
	time.Sleep(time.Duration(1600) * time.Millisecond)
	this.pool.evict()
	this.Equal(0, this.pool.GetNumIdle(), "Idle count different than expected.")
}

func (this *PoolTestSuite) TestEvictionInvalid() {
	this.pool = NewObjectPoolWithDefaultConfig(NewPooledObjectFactory(
		func() (interface{}, error) {
			return &TestObject{}, nil
		}, nil, func(object *PooledObject) bool {
			//fmt.Printf("TestEvictionInvalid valid object %v \n", object)
			time.Sleep(time.Duration(1000) * time.Millisecond)
			return false
		}, nil, nil))

	this.pool.Config.MaxIdle = 1
	this.pool.Config.MaxTotal = 1
	this.pool.Config.TestOnBorrow = false
	this.pool.Config.TestOnReturn = false
	this.pool.Config.TestWhileIdle = true
	this.pool.Config.MinEvictableIdleTimeMillis = int64(100000)
	this.pool.Config.NumTestsPerEvictionRun = 1

	p := this.NoErrorWithResult(this.pool.BorrowObject())
	this.NoError(this.pool.ReturnObject(p))

	// Run eviction in a separate thread
	go func() {
		fmt.Println("TestEvictionInvalid evict thread.")
		this.pool.evict()
	}()

	// Sleep to make sure evictor has started
	time.Sleep(time.Duration(300) * time.Millisecond)

	err := this.ErrorWithResult(this.pool.borrowObject(1))
	_, ok := err.(*NoSuchElementErr)
	this.True(ok, "expect NoSuchElementErr, but get %v", reflect.TypeOf(ok))

	// Make sure evictor has finished
	time.Sleep(time.Duration(1000) * time.Millisecond)
	// Should have an empty pool
	this.Equal(0, this.pool.GetNumIdle(), "Idle count different than expected.")
	this.Equal(0, this.pool.GetNumActive(), "Total count different than expected.")
}

func (this *PoolTestSuite) TestConcurrentInvalidate() {
	// Get allObjects and idleObjects loaded with some instances
	nObjects := 1000
	this.pool.Config.MaxTotal = nObjects
	this.pool.Config.MaxIdle = nObjects
	active := make([]*TestObject, nObjects)
	for i := 0; i < nObjects; i++ {
		active[i] = this.NoErrorWithResult(this.pool.BorrowObject()).(*TestObject)
	}
	for i := 0; i < nObjects; i++ {
		if i%2 == 0 {
			this.NoError(this.pool.ReturnObject(active[i]))
		}
	}
	nThreads := 20
	nIterations := 60
	// Randomly generated list of distinct invalidation targets
	targets := make(map[int]bool)
	for j := 0; j < nIterations; j++ {
		// Get a random invalidation target
		targ := rand.Intn(nObjects)
		for targets[targ] {
			targ = rand.Intn(nObjects)
		}
		targets[targ] = true
		// Launch nThreads threads all trying to invalidate the target
		results := make(chan bool, nThreads)
		for i := 0; i < nThreads; i++ {
			go func(pool *ObjectPool, obj *TestObject) {
				err := pool.InvalidateObject(obj)
				_, ok := err.(*IllegalStatusErr)
				if err != nil && !ok {
					results <- false
					fmt.Printf("TestConcurrentInvalidate InvalidateObject error:%v, obj: %v \n", err, obj)
				} else {
					results <- true
				}
			}(this.pool, active[targ])
		}
		for i := 0; i < nThreads; i++ {
			done := <-results
			this.True(done)
		}
	}
	this.Equal(nIterations, this.pool.GetDestroyedCount())
}

func sleep(millisecond int) {
	time.Sleep(time.Duration(millisecond) * time.Millisecond)
}

func (this *PoolTestSuite) TestMinIdle() {
	this.pool.Config.MaxIdle = 500
	this.pool.Config.MinIdle = 5
	this.pool.Config.MaxTotal = 10
	this.pool.Config.NumTestsPerEvictionRun = 0
	this.pool.Config.MinEvictableIdleTimeMillis = int64(50)
	this.pool.Config.TimeBetweenEvictionRunsMillis = int64(100)
	this.pool.Config.TestWhileIdle = true
	this.pool.StartEvictor()

	sleep(150)
	this.Equal(5, this.pool.GetNumIdle(), "Should be 5 idle, found %v", this.pool.GetNumIdle())

	active := make([]*TestObject, 5)
	active[0] = this.NoErrorWithResult(this.pool.BorrowObject()).(*TestObject)
	sleep(150)
	this.Equal(5, this.pool.GetNumIdle(), "Should be 5 idle, found %v", this.pool.GetNumIdle())

	for i := 1; i < 5; i++ {
		active[i] = this.NoErrorWithResult(this.pool.BorrowObject()).(*TestObject)
	}

	sleep(150)
	this.Equal(5, this.pool.GetNumIdle(), "Should be 5 idle, found %v", this.pool.GetNumIdle())

	for i := 0; i < 5; i++ {
		this.NoError(this.pool.ReturnObject(active[i]))
	}
	sleep(150)
	this.Equal(10, this.pool.GetNumIdle(), "Should be 10 idle, found %v", this.pool.GetNumIdle())
}

func (this *PoolTestSuite) TestMinIdleMaxTotal() {
	this.pool.Config.MaxIdle = 500
	this.pool.Config.MinIdle = 5
	this.pool.Config.MaxTotal = 10
	this.pool.Config.NumTestsPerEvictionRun = 0
	this.pool.Config.MinEvictableIdleTimeMillis = int64(50)
	this.pool.Config.TimeBetweenEvictionRunsMillis = int64(100)
	this.pool.Config.TestWhileIdle = true
	this.pool.StartEvictor()

	sleep(150)
	this.Equal(5, this.pool.GetNumIdle(), "Should be 5 idle, found %v", this.pool.GetNumIdle())

	active := make([]*TestObject, 10)
	sleep(150)
	this.Equal(5, this.pool.GetNumIdle(), "Should be 5 idle, found %v", this.pool.GetNumIdle())

	for i := 0; i < 5; i++ {
		active[i] = this.NoErrorWithResult(this.pool.BorrowObject()).(*TestObject)
	}
	sleep(150)
	this.Equal(5, this.pool.GetNumIdle(), "Should be 5 idle, found %v", this.pool.GetNumIdle())

	for i := 0; i < 5; i++ {
		this.NoError(this.pool.ReturnObject(active[i]))
	}
	sleep(150)
	this.Equal(10, this.pool.GetNumIdle(), "Should be 10 idle, found %v", this.pool.GetNumIdle())

	for i := 0; i < 10; i++ {
		active[i] = this.NoErrorWithResult(this.pool.BorrowObject()).(*TestObject)
	}
	sleep(150)
	this.Equal(0, this.pool.GetNumIdle(), "Should be 0 idle, found %v", this.pool.GetNumIdle())

	for i := 0; i < 10; i++ {
		this.NoError(this.pool.ReturnObject(active[i]))
	}
	sleep(150)
	this.Equal(10, this.pool.GetNumIdle(), "Should be 10 idle, found %v", this.pool.GetNumIdle())
}

func runTestThreads(t *testing.T, numThreads int, iterations int, delay int, testPool *ObjectPool) {

	arg := NewTestThreadArg(testPool, iterations, delay, delay, true, nil)
	resultChans := make([]chan TestThreadResult, numThreads)
	for i := 0; i < numThreads; i++ {
		resultChans[i] = threadRun(arg)
	}
	results := make([]TestThreadResult, numThreads)
	failedThreads := make([]int, 0)
	for i := 0; i < numThreads; i++ {
		result := <-resultChans[i]
		results[i] = result
		close(resultChans[i])
		if result.failed {
			failedThreads = append(failedThreads, i)
		}
	}
	if len(failedThreads) > 0 {
		for _, t := range failedThreads {
			fmt.Printf("Thread %v failed %v \n", t, results[t].error)
		}
		assert.Fail(t, fmt.Sprintf("Thread %v failed", failedThreads))
	}
}

func (this *PoolTestSuite) TestThreaded1() {
	this.pool.Config.MaxTotal = 15
	this.pool.Config.MaxIdle = 15
	this.pool.Config.MaxWaitMillis = int64(1000)
	runTestThreads(this.T(), 20, 100, 50, this.pool)
}

func (this *PoolTestSuite) TestMaxTotalInvariant() {
	maxTotal := 15
	this.factory.evenValid = false    // Every other validation fails
	this.factory.destroyLatency = 100 // Destroy takes 100 ms
	this.factory.maxTotal = maxTotal  // (makes - destroys) bound
	this.factory.enableValidation = true
	this.pool.Config.MaxTotal = maxTotal
	this.pool.Config.MaxIdle = -1
	this.pool.Config.TestOnReturn = true
	this.pool.Config.MaxWaitMillis = int64(1000)
	runTestThreads(this.T(), 5, 10, 50, this.pool)
}

func concurrentBorrowAndEvictThread(borrow bool, pool *ObjectPool) chan interface{} {
	ch := make(chan interface{}, 1)
	go func(borrow bool, pool *ObjectPool) {
		if borrow {
			obj, _ := pool.BorrowObject()
			ch <- obj
		} else {
			pool.evict()
			ch <- 1
		}
	}(borrow, pool)
	return ch
}

func (this *PoolTestSuite) TestConcurrentBorrowAndEvict() {
	this.pool.Config.MaxTotal = 1
	this.NoError(this.pool.AddObject())
	//set MaxWaitMillis avoid test use too long time
	this.pool.Config.MaxWaitMillis = 1000

	for i := 0; i < 5000; i++ {
		one := concurrentBorrowAndEvictThread(true, this.pool)
		two := concurrentBorrowAndEvictThread(false, this.pool)

		obj := <-one
		close(one)
		<-two
		close(two)
		this.NotNil(obj)
		this.NoError(this.pool.ReturnObject(obj))

		//Uncomment this for a progress indication
		//		if i%10 == 0 {
		//			fmt.Println(i)
		//		}
	}
}

//Verifies that concurrent threads never "share" instances
func (this *PoolTestSuite) TestNoInstanceOverlap() {
	maxTotal := 5
	numThreads := 100
	delay := 1
	iterations := 1000
	this.pool.Config.MaxTotal = maxTotal
	this.pool.Config.MaxIdle = maxTotal
	this.pool.Config.TestOnBorrow = true
	this.pool.Config.BlockWhenExhausted = true
	this.pool.Config.MaxWaitMillis = int64(-1)
	runTestThreads(this.T(), numThreads, iterations, delay, this.pool)
	this.Equal(0, this.pool.GetDestroyedByBorrowValidationCount())
}

func (this *PoolTestSuite) TestWhenExhaustedBlockClosePool() {
	this.pool.Config.MaxTotal = 1
	this.pool.Config.BlockWhenExhausted = true
	this.pool.Config.MaxWaitMillis = int64(-1)
	obj1 := this.NoErrorWithResult(this.pool.BorrowObject())

	// Make sure an object was obtained
	this.NotNil(obj1)

	// Create a separate thread to try and borrow another object
	ch := waitTestThread(this.pool, 200)
	// Give wtt time to start
	sleep(200)

	// close the pool (Bug POOL-189)
	this.pool.Close()

	// Give interrupt time to take effect
	sleep(200)

	// Check thread was interrupted
	result := <-ch
	close(ch)
	_, ok := result.error.(*collections.InterruptedErr)
	this.True(ok, "expect InterruptedErr, but get: %v", reflect.TypeOf(result.error))
}

func waitTestThread(pool *ObjectPool, pause int) chan TestThreadResult {
	ch := make(chan TestThreadResult, 1)
	go func() {
		result := TestThreadResult{}
		result.preborrow = currentTimeMillis()
		obj, err := pool.BorrowObject()
		result.objectId = obj
		result.postborrow = currentTimeMillis()
		if err == nil {
			sleep(pause)
			pool.ReturnObject(obj)
		}
		result.complete = true
		result.error = err
		result.failed = err != nil
		result.postreturn = currentTimeMillis()
		result.ended = currentTimeMillis()
		ch <- result
	}()
	return ch
}

func (this *PoolTestSuite) TestFIFO() {
	this.pool.Config.Lifo = false
	this.NoError(this.pool.AddObject()) // "0"
	this.NoError(this.pool.AddObject()) // "1"
	this.NoError(this.pool.AddObject()) // "2"
	this.Equal(getNthObject(0), this.NoErrorWithResult(this.pool.BorrowObject()), "Oldest")
	this.Equal(getNthObject(1), this.NoErrorWithResult(this.pool.BorrowObject()), "Middle")
	this.Equal(getNthObject(2), this.NoErrorWithResult(this.pool.BorrowObject()), "Youngest")
	o := this.NoErrorWithResult(this.pool.BorrowObject())
	this.Equal(getNthObject(3), o, "new-3")
	this.NoError(this.pool.ReturnObject(o))
	this.Equal(o, this.NoErrorWithResult(this.pool.BorrowObject()), "returned-3")
	this.Equal(getNthObject(4), this.NoErrorWithResult(this.pool.BorrowObject()), "new-4")
}

func (this *PoolTestSuite) TestLIFO() {
	this.pool.Config.Lifo = true
	this.NoError(this.pool.AddObject()) // "0"
	this.NoError(this.pool.AddObject()) // "1"
	this.NoError(this.pool.AddObject()) // "2"
	this.Equal(getNthObject(2), this.NoErrorWithResult(this.pool.BorrowObject()), "Youngest")
	this.Equal(getNthObject(1), this.NoErrorWithResult(this.pool.BorrowObject()), "Middle")
	this.Equal(getNthObject(0), this.NoErrorWithResult(this.pool.BorrowObject()), "Oldest")
	o := this.NoErrorWithResult(this.pool.BorrowObject())
	this.Equal(getNthObject(3), o, "new-3")
	this.NoError(this.pool.ReturnObject(o))
	this.Equal(o, this.NoErrorWithResult(this.pool.BorrowObject()), "returned-3")
	this.Equal(getNthObject(4), this.NoErrorWithResult(this.pool.BorrowObject()), "new-4")
}

func (this *PoolTestSuite) TestAddObject() {
	this.Equal(0, this.pool.GetNumIdle(), "should be zero idle")
	this.NoError(this.pool.AddObject())
	this.Equal(1, this.pool.GetNumIdle(), "should be one idle")
	this.Equal(0, this.pool.GetNumActive(), "should be zero active")
	obj := this.NoErrorWithResult(this.pool.BorrowObject())
	this.Equal(0, this.pool.GetNumIdle(), "should be zero idle")
	this.Equal(1, this.pool.GetNumActive(), "should be one active")
	this.NoError(this.pool.ReturnObject(obj))
	this.Equal(1, this.pool.GetNumIdle(), "should be one idle")
	this.Equal(0, this.pool.GetNumActive(), "should be zero active")
}

//TODO
//func (this *PoolTestSuite)  TestBorrowObjectFairness() {}

/**
 * On first borrow, first object fails validation, second object is OK.
 * Subsequent borrows are OK. This was POOL-152.
 */
func (this *PoolTestSuite) TestBrokenFactoryShouldNotBlockPool() {
	maxTotal := 1

	this.factory.maxTotal = maxTotal
	this.pool.Config.MaxTotal = maxTotal
	this.pool.Config.BlockWhenExhausted = true
	this.pool.Config.TestOnBorrow = true

	// First borrow object will need to create a new object which will fail
	// validation.
	this.factory.setValid(false)
	obj, ex := this.pool.BorrowObject()
	// Failure expected
	_, ok := ex.(*NoSuchElementErr)
	this.True(ok, "expect NoSuchElementErr, but get: %v", reflect.TypeOf(ex))
	this.Nil(obj)

	// Configure factory to create valid objects so subsequent borrows work
	this.factory.setValid(true)

	// Subsequent borrows should be OK
	obj = this.NoErrorWithResult(this.pool.BorrowObject())
	this.NoError(this.pool.ReturnObject(obj))
}

var DISPLAY_THREAD_DETAILS = true

/*
 * Test multi-threaded pool access.
 * Multiple threads, but maxTotal only allows half the threads to succeed.
 *
 * This test was prompted by Continuum build failures in the Commons DBCP test case:
 * TestPerUserPoolDataSource.testMultipleThreads2()
 * Let's see if the this fails on Continuum too!
 */
func (this *PoolTestSuite) TestMaxWaitMultiThreaded() {
	maxWait := 500          // wait for connection
	holdTime := 2 * maxWait // how long to hold connection
	threads := 10           // number of threads to grab the object initially
	this.pool.Config.BlockWhenExhausted = true
	this.pool.Config.MaxWaitMillis = int64(maxWait)
	this.pool.Config.MaxTotal = threads
	// Create enough threads so half the threads will have to wait
	resultChans := make([]chan TestThreadResult, threads*2)
	origin := currentTimeMillis() - 1000
	for i := 0; i < len(resultChans); i++ {
		resultChans[i] = waitTestThread(this.pool, holdTime)
	}
	var failed int = 0
	results := make([]TestThreadResult, len(resultChans))
	for i := 0; i < len(resultChans); i++ {
		ch := resultChans[i]
		result := <-ch
		close(ch)
		results[i] = result
		if result.error != nil {
			failed++
		}
	}
	if DISPLAY_THREAD_DETAILS || len(resultChans)/2 != failed {
		fmt.Println(
			"MaxWait: ", maxWait,
			" HoldTime: ", holdTime,
			" MaxTotal: ", threads,
			" Threads: ", len(resultChans),
			" Failed: ", failed)
		for _, result := range results {
			fmt.Println(
				"Preborrow: ", (result.preborrow - origin),
				" Postborrow: ", (result.postborrow - origin),
				" BorrowTime: ", (result.postborrow - result.preborrow),
				" PostReturn: ", (result.postreturn - origin),
				" Ended: ", (result.ended - origin),
				" ObjId: ", result.objectId)
		}
	}
	this.Equal(len(resultChans)/2, failed, "Expected half the threads to fail")
}

/**
* Test the following scenario:
*   Thread 1 borrows an instance
*   Thread 2 starts to borrow another instance before thread 1 returns its instance
*   Thread 1 returns its instance while thread 2 is validating its newly created instance
* The test verifies that the instance created by Thread 2 is not leaked.
 */
func (this *PoolTestSuite) TestMakeConcurrentWithReturn() {
	this.pool.Config.TestOnBorrow = true
	this.factory.setValid(true)
	// Borrow and return an instance, with a short wait
	ch := waitTestThread(this.pool, 200)
	sleep(50) // wait for validation to succeed
	// Slow down validation and borrow an instance
	this.factory.setValidateLatency(400)
	instance := this.NoErrorWithResult(this.pool.BorrowObject())
	// Now make sure that we have not leaked an instance
	this.Equal(this.factory.makeCounter, this.pool.GetNumIdle()+1)
	this.NoError(this.pool.ReturnObject(instance))
	this.Equal(this.factory.makeCounter, this.pool.GetNumIdle())
	<-ch
	close(ch)
}

/**
 * Verify that threads waiting on a depleted pool get served when a checked out object is
 * invalidated.
 *
 * JIRA: POOL-240
 */
func (this *PoolTestSuite) TestInvalidateFreesCapacity() {
	this.pool.Config.MaxTotal = 2
	this.pool.Config.MaxWaitMillis = 500
	this.pool.Config.BlockWhenExhausted = true
	// Borrow an instance and hold if for 5 seconds
	ch1 := waitTestThread(this.pool, 5000)
	// Borrow another instance
	obj := this.NoErrorWithResult(this.pool.BorrowObject())
	// Launch another thread - will block, but fail in 500 ms
	ch2 := waitTestThread(this.pool, 100)
	// Invalidate the object borrowed by this thread - should allow thread2 to create
	sleep(20)
	this.NoError(this.pool.InvalidateObject(obj))
	sleep(600) // Wait for thread2 to timeout
	result2 := <-ch2
	close(ch2)
	if result2.error != nil {
		this.Fail(result2.error.Error())
	}
	<-ch1
	close(ch1)
}

/**
* Verify that threads waiting on a depleted pool get served when a returning object fails
* validation.
*
* JIRA: POOL-240
*
 */
func (this *PoolTestSuite) TestValidationFailureOnReturnFreesCapacity() {
	this.factory.setValid(false) // Validate will always fail
	this.factory.enableValidation = true
	this.pool.Config.MaxTotal = 2
	this.pool.Config.MaxWaitMillis = int64(1500)
	this.pool.Config.TestOnReturn = true
	this.pool.Config.TestOnBorrow = false
	// Borrow an instance and hold if for 5 seconds
	ch1 := waitTestThread(this.pool, 5000)
	// Borrow another instance and return it after 500 ms (validation will fail)
	ch2 := waitTestThread(this.pool, 500)
	sleep(50)
	// Try to borrow an object
	obj := this.NoErrorWithResult(this.pool.BorrowObject())
	this.NoError(this.pool.ReturnObject(obj))
	<-ch1
	close(ch1)
	<-ch2
	close(ch2)
}

//TODO
//func (this *PoolTestSuite) TestSwallowedExceptionListener() {
//}

// POOL-248
func (this *PoolTestSuite) TestMultipleReturnOfSameObject() {
	this.Equal(0, this.pool.GetNumActive())
	this.Equal(0, this.pool.GetNumIdle())

	obj := this.NoErrorWithResult(this.pool.BorrowObject())

	this.Equal(1, this.pool.GetNumActive())
	this.Equal(0, this.pool.GetNumIdle())

	this.NoError(this.pool.ReturnObject(obj))

	this.Equal(0, this.pool.GetNumActive())
	this.Equal(1, this.pool.GetNumIdle())

	err := this.pool.ReturnObject(obj)
	_, ok := err.(*IllegalStatusErr)
	this.True(ok, "expect IllegalStatusErr, but get %v", reflect.TypeOf(err))
	this.Equal(0, this.pool.GetNumActive())
	this.Equal(1, this.pool.GetNumIdle())
}

// TODO POOL-259
//func (this *PoolTestSuite) TestClientWaitStats() {
//}

// POOL-276
func (this *PoolTestSuite) TestValidationOnCreateOnly() {
	this.pool.Config.MaxTotal = 1
	this.pool.Config.TestOnCreate = true
	this.pool.Config.TestOnBorrow = false
	this.pool.Config.TestOnReturn = false
	this.pool.Config.TestWhileIdle = false

	o1 := this.NoErrorWithResult(this.pool.BorrowObject())
	this.Equal(getNthObject(0), o1)
	go func() {
		sleep(3000)
		this.pool.ReturnObject(o1)
	}()

	o2 := this.NoErrorWithResult(this.pool.BorrowObject())
	this.Equal(getNthObject(0), o2)

	this.Equal(1, this.factory.validateCounter)
}

/**
* Verifies that when a factory's makeObject produces instances that are not
* discernible by == , the pool can handle them.
*
* JIRA: POOL-283
 */
func (this *PoolTestSuite) TestEqualsIndiscernible() {
	pool := NewObjectPoolWithDefaultConfig(NewPooledObjectFactorySimple(func() (interface{}, error) {
		return make(map[string]string), nil
	}))
	m1 := this.NoErrorWithResult(pool.BorrowObject())
	m2 := this.NoErrorWithResult(pool.BorrowObject())
	this.NoError(pool.ReturnObject(m1))
	this.NoError(pool.ReturnObject(m2))
	pool.Close()
}

/**
 * Verifies that when a borrowed object is mutated in a way that does not
 * preserve equality and hashcode, the pool can recognized it on return.
 *
 * JIRA: POOL-284
 */
func (this *PoolTestSuite) TestMutable() {
	pool := NewObjectPoolWithDefaultConfig(NewPooledObjectFactorySimple(func() (interface{}, error) {
		return make(map[string]string), nil
	}))
	m1 := this.NoErrorWithResult(pool.BorrowObject()).(map[string]string)
	m2 := this.NoErrorWithResult(pool.BorrowObject()).(map[string]string)
	m1["k1"] = "v1"
	m2["k2"] = "v2"
	this.NoError(pool.ReturnObject(m1))
	this.NoError(pool.ReturnObject(m2))
	this.Equal(2, pool.GetNumIdle())
	pool.Close()
}

/**
* Verifies that returning an object twice (without borrow in between) causes ISE
* but does not re-validate or re-passivate the instance.
*
* JIRA: POOL-285
 */
//TODO
//func (this *PoolTestSuite) TestMultipleReturn() {
//}
