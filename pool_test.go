package pool

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"math"
	"math/rand"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/fortytw2/leaktest"
	"github.com/jolestar/go-commons-pool/collections"
	"github.com/jolestar/go-commons-pool/concurrent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type TestObject struct {
	Num int
}

var (
	debugTest = false
)

func init() {
	rand.Seed(time.Now().UnixNano())
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
	exceptionOnMake      bool
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

func (f *SimpleFactory) setValid(valid bool) {
	f.lock.Lock()
	f.evenValid = valid
	f.oddValid = valid
	f.lock.Unlock()
}

func (f *SimpleFactory) setValidateLatency(validateLatency int64) {
	f.lock.Lock()
	f.validateLatency = validateLatency
	f.lock.Unlock()
}

func (f *SimpleFactory) doWait(latencyMillisecond int64) {
	time.Sleep(time.Duration(latencyMillisecond) * time.Millisecond)
}

func (f *SimpleFactory) MakeObject(context.Context) (*PooledObject, error) {
	if debugTest {
		fmt.Println("factory MakeObject")
	}
	if f.exceptionOnMake {
		return nil, errors.New("make object error")
	}
	var waitLatency int64
	f.lock.Lock()
	f.activeCount = f.activeCount + 1
	if f.activeCount > f.maxTotal {
		return nil, fmt.Errorf("Too many active instances: %v", f.activeCount)
	}
	waitLatency = f.makeLatency
	f.lock.Unlock()
	if waitLatency > 0 {
		f.doWait(waitLatency)
	}
	var counter int
	f.lock.Lock()
	counter = f.makeCounter
	f.makeCounter = f.makeCounter + 1
	f.lock.Unlock()
	return NewPooledObject(&TestObject{Num: counter}), nil
}

func (f *SimpleFactory) DestroyObject(ctx context.Context, object *PooledObject) error {
	if debugTest {
		fmt.Println("factory DestroyObject")
	}
	var waitLatency int64
	var hurl bool
	f.lock.Lock()
	waitLatency = f.destroyLatency
	hurl = f.exceptionOnDestroy
	f.lock.Unlock()
	if waitLatency > 0 {
		f.doWait(waitLatency)
	}
	f.lock.Lock()
	f.activeCount = f.activeCount - 1
	f.lock.Unlock()
	if hurl {
		return errors.New("destroy error")
	}
	return nil
}

func (f *SimpleFactory) ValidateObject(ctx context.Context, object *PooledObject) bool {
	if debugTest {
		fmt.Println("factory ValidateObject")
	}
	var validate bool
	var evenTest bool
	var oddTest bool
	var waitLatency int64
	var counter int
	f.lock.Lock()
	validate = f.enableValidation
	evenTest = f.evenValid
	oddTest = f.oddValid
	counter = f.validateCounter
	f.validateCounter = f.validateCounter + 1
	waitLatency = f.validateLatency
	f.lock.Unlock()
	if waitLatency > 0 {
		f.doWait(waitLatency)
	}
	if validate {
		if counter%2 == 0 {
			return evenTest
		}
		return oddTest
	}
	return true
}

func (f *SimpleFactory) ActivateObject(ctx context.Context, object *PooledObject) error {
	if debugTest {
		fmt.Println("factory ActivateObject")
		defer fmt.Println("factory ActivateObject end")
	}
	var hurl bool
	var evenTest bool
	var oddTest bool
	var counter int
	f.lock.Lock()
	hurl = f.exceptionOnActivate
	evenTest = f.evenValid
	oddTest = f.oddValid
	counter = f.activationCounter
	f.activationCounter = f.activationCounter + 1
	f.lock.Unlock()
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

func (f *SimpleFactory) PassivateObject(ctx context.Context, object *PooledObject) error {
	if debugTest {
		fmt.Println("factory PassivateObject")
	}
	var hurl bool
	f.lock.Lock()
	hurl = f.exceptionOnPassivate
	f.lock.Unlock()
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

func (suit *PoolTestSuite) assertEquals(expect interface{}, actual interface{}) {
	suit.Equal(expect, actual)
}

func (suit *PoolTestSuite) assertNotNil(object interface{}) {
	suit.NotNil(object)
}

func (suit *PoolTestSuite) assertNil(object interface{}) {
	suit.Nil(object)
}

func (suit *PoolTestSuite) NoErrorWithResult(object interface{}, err error) interface{} {
	suit.NotNil(object)
	suit.Nil(err)
	return object
}

func (suit *PoolTestSuite) ErrorWithResult(object interface{}, err error) error {
	suit.Nil(object)
	suit.NotNil(err)
	return err
}

func TestPoolTestSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(PoolTestSuite))
}

func (suit *PoolTestSuite) SetupTest() {
	suit.makeEmptyPool(context.Background(), DefaultMaxTotal)
}

func (suit *PoolTestSuite) TearDownTest() {
	ctx := context.Background()
	suit.pool.Clear(ctx)
	suit.pool.Close(ctx)
	suit.pool = nil
	suit.factory = nil
}

func (suit *PoolTestSuite) makeEmptyPool(ctx context.Context, maxTotal int) {
	suit.factory = NewSimpleFactory()
	suit.pool = NewObjectPoolWithDefaultConfig(ctx, suit.factory)
	suit.pool.Config.MaxTotal = maxTotal
}

func getNthObject(num int) *TestObject {
	return &TestObject{Num: num}
}

func (suit *PoolTestSuite) TestBaseBorrow() {
	ctx := context.Background()
	suit.pool.Config.MaxTotal = 3
	o0, err := suit.pool.BorrowObject(ctx)

	suit.Nil(err)
	suit.NotNil(o0)

	suit.Equal(getNthObject(0), o0)
	o1, _ := suit.pool.BorrowObject(ctx)
	suit.Equal(getNthObject(1), o1)
	o2, _ := suit.pool.BorrowObject(ctx)
	suit.Equal(getNthObject(2), o2)
}

func (suit *PoolTestSuite) TestBaseAddObject() {
	ctx := context.Background()
	suit.pool.Config.MaxTotal = 3
	suit.assertEquals(0, suit.pool.GetNumIdle())
	suit.assertEquals(0, suit.pool.GetNumActive())
	if debugTest {
		fmt.Println("test AddObject")
	}
	suit.pool.AddObject(ctx)

	suit.assertEquals(1, suit.pool.GetNumIdle())
	suit.assertEquals(0, suit.pool.GetNumActive())
	if debugTest {
		fmt.Println("test BorrowObject")
	}
	obj, err := suit.pool.BorrowObject(ctx)
	if err != nil {
		suit.Fail(err.Error())
	}

	suit.assertEquals(getNthObject(0), obj)
	suit.assertEquals(0, suit.pool.GetNumIdle())
	suit.assertEquals(1, suit.pool.GetNumActive())
	err = suit.pool.ReturnObject(ctx, obj)
	if err != nil {
		suit.Fail(err.Error())
	}
	suit.assertEquals(1, suit.pool.GetNumIdle())
	suit.assertEquals(0, suit.pool.GetNumActive())
}

func (suit *PoolTestSuite) isLifo() bool {
	return true
}

func (suit *PoolTestSuite) isFifo() bool {
	return false
}

func (suit *PoolTestSuite) TestBaseBorrowReturn() {
	ctx := context.Background()

	suit.pool.Config.MaxTotal = 3

	obj0 := suit.NoErrorWithResult(suit.pool.BorrowObject(ctx))
	suit.assertEquals(getNthObject(0), obj0)
	obj1 := suit.NoErrorWithResult(suit.pool.BorrowObject(ctx))
	suit.assertEquals(getNthObject(1), obj1)
	obj2 := suit.NoErrorWithResult(suit.pool.BorrowObject(ctx))
	suit.assertEquals(getNthObject(2), obj2)

	suit.NoError(suit.pool.ReturnObject(ctx, obj2))

	obj2 = suit.NoErrorWithResult(suit.pool.BorrowObject(ctx))
	suit.assertEquals(getNthObject(2), obj2)
	suit.pool.ReturnObject(ctx, obj1)
	obj1 = suit.NoErrorWithResult(suit.pool.BorrowObject(ctx))

	suit.assertEquals(getNthObject(1), obj1)
	suit.pool.ReturnObject(ctx, obj0)
	suit.pool.ReturnObject(ctx, obj2)
	obj2 = suit.NoErrorWithResult(suit.pool.BorrowObject(ctx))
	if suit.isLifo() {
		suit.assertEquals(getNthObject(2), obj2)
	}
	if suit.isFifo() {
		suit.assertEquals(getNthObject(0), obj2)
	}

	obj0 = suit.NoErrorWithResult(suit.pool.BorrowObject(ctx))
	if suit.isLifo() {
		suit.assertEquals(getNthObject(0), obj0)
	}
	if suit.isFifo() {
		suit.assertEquals(getNthObject(2), obj0)
	}
}

func (suit *PoolTestSuite) TestBorrowReturnAsync() {
	suit.pool.Config.MaxTotal = 1

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	obj0 := suit.NoErrorWithResult(suit.pool.BorrowObject(ctx))
	suit.assertEquals(getNthObject(0), obj0)

	//start new goroutine to borrow will block
	ch := make(chan interface{}, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		obj, _ := suit.pool.BorrowObject(ctx)
		ch <- obj
	}()
	sleep(100)

	//return obj0
	go func() {
		suit.pool.ReturnObject(context.Background(), obj0)
	}()
	sleep(100)
	obj1 := <-ch
	suit.NotNil(obj1)
	suit.Equal(obj0, obj1)
}

func (suit *PoolTestSuite) TestBaseNumActiveNumIdle() {
	ctx := context.Background()

	suit.pool.Config.MaxTotal = 3

	suit.assertEquals(0, suit.pool.GetNumActive())
	suit.assertEquals(0, suit.pool.GetNumIdle())
	obj0 := suit.NoErrorWithResult(suit.pool.BorrowObject(ctx))
	suit.assertEquals(1, suit.pool.GetNumActive())
	suit.assertEquals(0, suit.pool.GetNumIdle())
	obj1 := suit.NoErrorWithResult(suit.pool.BorrowObject(ctx))
	suit.assertEquals(2, suit.pool.GetNumActive())
	suit.assertEquals(0, suit.pool.GetNumIdle())
	suit.pool.ReturnObject(ctx, obj1)
	suit.assertEquals(1, suit.pool.GetNumActive())
	suit.assertEquals(1, suit.pool.GetNumIdle())
	suit.NoError(suit.pool.ReturnObject(ctx, obj0))
	suit.assertEquals(0, suit.pool.GetNumActive())
	suit.assertEquals(2, suit.pool.GetNumIdle())
}

func (suit *PoolTestSuite) TestBaseClear() {
	ctx := context.Background()

	suit.pool.Config.MaxTotal = 3

	suit.assertEquals(0, suit.pool.GetNumActive())
	suit.assertEquals(0, suit.pool.GetNumIdle())
	obj0 := suit.NoErrorWithResult(suit.pool.BorrowObject(ctx))
	obj1 := suit.NoErrorWithResult(suit.pool.BorrowObject(ctx))
	suit.assertEquals(2, suit.pool.GetNumActive())
	suit.assertEquals(0, suit.pool.GetNumIdle())
	suit.pool.ReturnObject(ctx, obj1)
	suit.pool.ReturnObject(ctx, obj0)
	suit.assertEquals(0, suit.pool.GetNumActive())
	suit.assertEquals(2, suit.pool.GetNumIdle())
	suit.pool.Clear(ctx)
	suit.assertEquals(0, suit.pool.GetNumActive())
	suit.assertEquals(0, suit.pool.GetNumIdle())
	obj2 := suit.NoErrorWithResult(suit.pool.BorrowObject(ctx))
	suit.assertEquals(getNthObject(2), obj2)
}

func (suit *PoolTestSuite) TestBaseInvalidateObject() {
	ctx := context.Background()

	suit.pool.Config.MaxTotal = 3

	suit.assertEquals(0, suit.pool.GetNumActive())
	suit.assertEquals(0, suit.pool.GetNumIdle())
	obj0 := suit.NoErrorWithResult(suit.pool.BorrowObject(ctx))
	obj1 := suit.NoErrorWithResult(suit.pool.BorrowObject(ctx))
	suit.assertEquals(2, suit.pool.GetNumActive())
	suit.assertEquals(0, suit.pool.GetNumIdle())
	err := suit.pool.InvalidateObject(ctx, obj0)
	suit.NoError(err)
	suit.assertEquals(1, suit.pool.GetNumActive())
	suit.assertEquals(0, suit.pool.GetNumIdle())
	err = suit.pool.InvalidateObject(ctx, obj1)
	suit.NoError(err)
	suit.assertEquals(0, suit.pool.GetNumActive())
	suit.assertEquals(0, suit.pool.GetNumIdle())
}

func (suit *PoolTestSuite) TestBaseClosePool() {
	ctx := context.Background()

	suit.pool.Config.MaxTotal = 3

	obj, err := suit.pool.BorrowObject(ctx)
	suit.NoError(err)
	suit.pool.ReturnObject(ctx, obj)

	suit.pool.Close(ctx)
	obj, err = suit.pool.BorrowObject(ctx)
	suit.NotNil(err)
	suit.Nil(obj)
}

func (suit *PoolTestSuite) TestWhenExhaustedFail() {
	ctx := context.Background()

	suit.pool.Config.MaxTotal = 1

	suit.pool.Config.BlockWhenExhausted = false
	obj1 := suit.NoErrorWithResult(suit.pool.BorrowObject(ctx))

	err2 := suit.ErrorWithResult(suit.pool.BorrowObject(ctx))
	_, ok := err2.(*NoSuchElementErr)
	suit.True(ok, "expect NoSuchElementErr but get", reflect.TypeOf(err2))

	suit.pool.ReturnObject(ctx, obj1)
	suit.assertEquals(1, suit.pool.GetNumIdle())
}

func (suit *PoolTestSuite) TestWhenExhaustedBlock() {
	suit.pool.Config.MaxTotal = 1
	suit.pool.Config.BlockWhenExhausted = true

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	obj1 := suit.NoErrorWithResult(suit.pool.BorrowObject(ctx))

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err2 := suit.ErrorWithResult(suit.pool.BorrowObject(ctx))
	_, ok := err2.(*NoSuchElementErr)
	suit.True(ok, "expect NoSuchElementErr but get", reflect.TypeOf(err2))

	suit.pool.ReturnObject(ctx, obj1)
}

func borrowAndWait(ctx context.Context, pool *ObjectPool, pause time.Duration) chan int {
	ch := make(chan int, 1)
	go func() {
		preborrow := currentTimeMillis()
		obj, _ := pool.BorrowObject(ctx)
		//objectId = obj;
		postborrow := currentTimeMillis()
		ch <- int(postborrow - preborrow)
		time.Sleep(pause)
		if obj != nil {
			pool.ReturnObject(ctx, obj)
		}
		//postreturn = System.currentTimeMillis();
	}()
	return ch
}

func (suit *PoolTestSuite) TestWhenExhaustedBlockInterrupt() {
	ctx := context.Background()

	suit.pool.Config.MaxTotal = 1

	suit.pool.Config.BlockWhenExhausted = true

	obj1, _ := suit.pool.BorrowObject(ctx)

	// Make sure on object was obtained
	suit.assertNotNil(obj1)

	// Create a separate goroutine to try and borrow another object
	//WaitingTestGoroutine wtt = new WaitingTestGoroutine(pool, 200000);
	ch := borrowAndWait(ctx, suit.pool, 200000*time.Millisecond)

	// Give wtt time to start
	time.Sleep(200 * time.Millisecond)

	suit.pool.idleObjects.InterruptTakeWaiters()

	borrowTime := <-ch
	close(ch)
	if debugTest {
		fmt.Println("TestWhenExhaustedBlockInterrupt borrowTime:", borrowTime)
	}
	suit.True(borrowTime >= 200)

	// Check goroutine was interrupted
	//assertTrue(wtt._thrown instanceof InterruptedException);

	// Return object to the pool
	suit.pool.ReturnObject(ctx, obj1)

	// Bug POOL-162 - check there is now an object in the pool
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	obj2 := suit.NoErrorWithResult(suit.pool.BorrowObject(ctx))
	suit.pool.ReturnObject(ctx, obj2)
}

func (suit *PoolTestSuite) TestEvictWhileEmpty() {
	ctx := context.Background()

	suit.pool.evict(ctx)
	suit.pool.evict(ctx)
}

type TestGoroutineArg struct {
	/** pool to borrow from */
	pool *ObjectPool

	/** number of borrow attempts */
	iter int

	/** delay before each borrow attempt */
	startDelay int

	borrowTimeout time.Duration

	/** time to hold each borrowed object before returning it */
	holdTime int

	/** whether or not start and hold time are randomly generated */
	randomDelay bool

	/** object expected to be borrowed (fail otherwise) */
	expectedObject interface{}
}

type TestGoroutineResult struct {
	complete   bool
	failed     bool
	error      error
	preborrow  int64
	postborrow int64
	postreturn int64
	ended      int64
	objectID   interface{}
}

func NewTesGoroutineArgSimple(pool *ObjectPool, iter int, delay int, randomDelay bool) *TestGoroutineArg {
	return NewTestGoroutineArg(pool, iter, delay, time.Duration(0), delay, randomDelay, nil)
}

func NewTestGoroutineArg(pool *ObjectPool, iter int, startDelay int, borrowTimeout time.Duration,
	holdTime int, randomDelay bool, obj interface{}) *TestGoroutineArg {
	return &TestGoroutineArg{
		pool:           pool,
		iter:           iter,
		startDelay:     startDelay,
		borrowTimeout:  borrowTimeout,
		holdTime:       holdTime,
		randomDelay:    randomDelay,
		expectedObject: obj,
	}
}

func goroutineRun(ctx context.Context, arg *TestGoroutineArg) chan TestGoroutineResult {
	resultChan := make(chan TestGoroutineResult, 1)
	result := TestGoroutineResult{}
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
			borrowCtx := ctx
			if arg.borrowTimeout > 0 {
				var borrowCancel func()
				borrowCtx, borrowCancel = context.WithTimeout(ctx, arg.borrowTimeout)
				defer borrowCancel()
			}
			obj, err := arg.pool.BorrowObject(borrowCtx)
			endBorrow := currentTimeMillis()
			if err != nil {
				if debugTest {
					fmt.Println("borrow error, time:", endBorrow-startBorrow)
				}
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
			err = arg.pool.ReturnObject(ctx, obj)
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

func (suit *PoolTestSuite) TestEvictAddObjects() {
	ctx := context.Background()

	suit.factory.makeLatency = 300
	suit.factory.maxTotal = 2
	suit.pool.Config.MaxTotal = 2
	suit.pool.Config.MinIdle = 1
	suit.pool.BorrowObject(ctx) // numActive = 1, numIdle = 0
	// Create a test goroutine that will run once and try a borrow after
	// 150ms fixed delay
	borrower := NewTesGoroutineArgSimple(suit.pool, 1, 150, false)
	//// Set evictor to run in 100 ms - will create idle instance
	suit.pool.Config.TimeBetweenEvictionRunsMillis = int64(100)
	ch := goroutineRun(ctx, borrower)
	result := <-ch
	close(ch)
	if debugTest {
		fmt.Printf("TestEvictAddObjects %v error:%v", borrower, result.error)
	}
	suit.True(!result.failed)
}

func (suit *PoolTestSuite) TestEvictLIFO() {
	suit.checkEvict(context.Background(), true)
}

func (suit *PoolTestSuite) TestEvictFIFO() {
	suit.checkEvict(context.Background(), false)
}

func (suit *PoolTestSuite) checkEvict(ctx context.Context, lifo bool) {
	var idle int
	// yea suit is hairy but it tests all the code paths in GOP.evict()
	suit.pool.Config.SoftMinEvictableIdleTimeMillis = int64(10)
	suit.pool.Config.MinIdle = 2
	suit.pool.Config.TestWhileIdle = true
	suit.pool.Config.Lifo = lifo
	Prefill(ctx, suit.pool, 5)
	suit.pool.evict(ctx)
	idle = suit.pool.GetNumIdle()
	if debugTest {
		fmt.Printf("checkEvict lifo:%v idel:%v \n", lifo, idle)
	}
	suit.factory.evenValid = false
	suit.factory.oddValid = false
	suit.factory.exceptionOnActivate = true
	suit.pool.evict(ctx)
	idle = suit.pool.GetNumIdle()
	if debugTest {
		fmt.Printf("checkEvict lifo:%v idel:%v \n", lifo, idle)
	}
	Prefill(ctx, suit.pool, 5)
	suit.factory.exceptionOnActivate = false
	suit.factory.exceptionOnPassivate = true
	suit.pool.evict(ctx)
	idle = suit.pool.GetNumIdle()
	if debugTest {
		fmt.Printf("checkEvict lifo:%v idel:%v \n", lifo, idle)
	}
	suit.factory.exceptionOnPassivate = false
	suit.factory.evenValid = true
	suit.factory.oddValid = true
	time.Sleep(time.Duration(125) * time.Millisecond)
	suit.pool.evict(ctx)
	idle = suit.pool.GetNumIdle()
	if debugTest {
		fmt.Printf("checkEvict lifo:%v idel:%v \n", lifo, idle)
	}
	suit.assertEquals(2, suit.pool.GetNumIdle())
}

func (suit *PoolTestSuite) TestEvictionOrder() {
	ctx := context.Background()
	suit.checkEvictionOrder(ctx, false)
	suit.TearDownTest()
	suit.SetupTest()
	suit.checkEvictionOrder(ctx, true)
}

func (suit *PoolTestSuite) checkEvictionOrder(ctx context.Context, lifo bool) {
	suit.checkEvictionOrderPart1(ctx, lifo)
	suit.TearDownTest()
	suit.SetupTest()
	suit.checkEvictionOrderPart2(ctx, lifo)
}

func (suit *PoolTestSuite) checkEvictionOrderPart1(ctx context.Context, lifo bool) {
	suit.pool.Config.NumTestsPerEvictionRun = 2
	suit.pool.Config.MinEvictableIdleTimeMillis = 100
	suit.pool.Config.Lifo = lifo
	for i := 0; i < 5; i++ {
		suit.pool.AddObject(ctx)
		time.Sleep(time.Duration(100) * time.Millisecond)
	}
	// Order, oldest to youngest, is "0", "1", ...,"4"
	suit.pool.evict(ctx) // Should evict "0" and "1"
	obj, _ := suit.pool.BorrowObject(ctx)
	suit.True(getNthObject(0) != obj, "oldest not evicted")
	suit.True(getNthObject(1) != obj, "second oldest not evicted")
	// 2 should be next out for FIFO, 4 for LIFO
	var expect *TestObject
	if lifo {
		expect = getNthObject(4)
	} else {
		expect = getNthObject(2)
	}
	suit.Equal(expect, obj, "Wrong instance returned")
}

func (suit *PoolTestSuite) checkEvictionOrderPart2(ctx context.Context, lifo bool) {
	// Two eviction runs in sequence
	suit.pool.Config.NumTestsPerEvictionRun = 2
	suit.pool.Config.MinEvictableIdleTimeMillis = int64(100)
	suit.pool.Config.Lifo = lifo
	for i := 0; i < 5; i++ {
		suit.pool.AddObject(ctx)
		time.Sleep(time.Duration(100) * time.Millisecond)
	}
	suit.pool.evict(ctx) // Should evict "0" and "1"
	suit.pool.evict(ctx) // Should evict "2" and "3"
	obj, _ := suit.pool.BorrowObject(ctx)
	suit.Equal(getNthObject(4), obj, "Wrong instance remaining in pool")
}

func (suit *PoolTestSuite) TestEvictorVisiting() {
	suit.checkEvictorVisiting(true)
	suit.checkEvictorVisiting(false)
}

func (suit *PoolTestSuite) checkEvictorVisiting(lifo bool) {
	//TODO
}

func (suit *PoolTestSuite) TestExceptionOnPassivateDuringReturn() {
	ctx := context.Background()
	obj, _ := suit.pool.BorrowObject(ctx)
	suit.factory.exceptionOnPassivate = true
	suit.pool.ReturnObject(ctx, obj)
	suit.assertEquals(0, suit.pool.GetNumIdle())
}

func (suit *PoolTestSuite) TestExceptionOnPassivateDuringReturnWithBorrowWaiting() {
	ctx := context.Background()

	suit.pool.Config.MaxTotal = 1
	suit.pool.Config.BlockWhenExhausted = true
	suit.factory.exceptionOnPassivate = true

	obj, _ := suit.pool.BorrowObject(ctx)

	goBorrow := func(ch chan<- interface{}, pool *ObjectPool) {
		obj, _ := suit.pool.BorrowObject(ctx)
		ch <- obj
	}

	ch1 := make(chan interface{})
	go goBorrow(ch1, suit.pool)

	ch2 := make(chan interface{})
	go goBorrow(ch2, suit.pool)

	select {
	case <-ch1:
		suit.FailNow("Borrowing additional objects should have blocked")

	case <-ch2:
		suit.FailNow("Borrowing additional objects should have blocked")

	case <-time.After(100 * time.Millisecond):
		// Just wait a moment to make sure neither of those channels, above, read...
	}

	suit.pool.ReturnObject(ctx, obj)

	select {
	case obj1 := <-ch1:
		suit.T().Log("Returning item borrowed (ch1)")
		suit.pool.ReturnObject(ctx, obj1)

	case obj2 := <-ch2:
		suit.T().Log("Returning item borrowed (ch2)")
		suit.pool.ReturnObject(ctx, obj2)

	case <-time.After(100 * time.Millisecond):
		// Just wait a moment to make sure neither of those channels, above, read...

		suit.FailNow("Failed to borrow additional objects")
	}

	// Once again, for the other channel

	select {
	case obj1 := <-ch1:
		suit.T().Log("Returning item borrowed (ch1)")
		suit.pool.ReturnObject(ctx, obj1)

	case obj2 := <-ch2:
		suit.T().Log("Returning item borrowed (ch2)")
		suit.pool.ReturnObject(ctx, obj2)

	case <-time.After(100 * time.Millisecond):
		// Just wait a moment to make sure neither of those channels, above, read...

		suit.FailNow("Failed to borrow additional objects")
	}
}

func (suit *PoolTestSuite) TestExceptionOnDestroyDuringBorrow() {
	ctx := context.Background()
	suit.factory.exceptionOnDestroy = true
	suit.pool.Config.TestOnBorrow = true
	suit.pool.BorrowObject(ctx)
	suit.factory.setValid(false) // Make validation fail on next borrow attempt
	_, err := suit.pool.BorrowObject(ctx)
	suit.NotNil(err)
	suit.assertEquals(1, suit.pool.GetNumActive())
	suit.assertEquals(0, suit.pool.GetNumIdle())
}

func (suit *PoolTestSuite) TestExceptionOnDestroyDuringReturn() {
	ctx := context.Background()
	suit.factory.exceptionOnDestroy = true
	suit.pool.Config.TestOnReturn = true
	obj1, _ := suit.pool.BorrowObject(ctx)
	suit.pool.BorrowObject(ctx)
	suit.factory.setValid(false) // Make validation fail
	suit.pool.ReturnObject(context.Background(), obj1)
	suit.assertEquals(1, suit.pool.GetNumActive())
	suit.assertEquals(0, suit.pool.GetNumIdle())
}

func (suit *PoolTestSuite) TestExceptionOnActivateDuringBorrow() {
	ctx := context.Background()
	obj1, _ := suit.pool.BorrowObject(ctx)
	obj2, _ := suit.pool.BorrowObject(ctx)
	suit.pool.ReturnObject(ctx, obj1)
	suit.pool.ReturnObject(ctx, obj2)
	suit.factory.exceptionOnActivate = true
	suit.factory.evenValid = false
	// Activation will now throw every other time
	// First attempt throws, but loop continues and second succeeds
	obj, _ := suit.pool.BorrowObject(ctx)
	suit.assertEquals(1, suit.pool.GetNumActive())
	suit.assertEquals(0, suit.pool.GetNumIdle())

	suit.pool.ReturnObject(ctx, obj)
	suit.factory.setValid(false)
	// Validation will now fail on activation when borrowObject returns
	// an idle instance, and then when attempting to create a new instance
	_, err := suit.pool.BorrowObject(ctx)
	_, ok := err.(*NoSuchElementErr)
	suit.True(ok, "expect NoSuchElementErr but get", reflect.TypeOf(err))

	suit.assertEquals(0, suit.pool.GetNumActive())
	suit.assertEquals(0, suit.pool.GetNumIdle())
}

func (suit *PoolTestSuite) TestNegativeMaxTotal() {
	ctx := context.Background()
	suit.pool.Config.MaxTotal = -1
	suit.pool.Config.BlockWhenExhausted = false
	obj, _ := suit.pool.BorrowObject(ctx)
	suit.assertEquals(getNthObject(0), obj)
	suit.pool.ReturnObject(context.Background(), obj)
}

func (suit *PoolTestSuite) TestMaxIdle() {
	ctx := context.Background()
	suit.pool.Config.MaxTotal = 100
	suit.pool.Config.MaxIdle = 8
	active := make([]*TestObject, 100)
	for i := 0; i < 100; i++ {
		obj, err := suit.pool.BorrowObject(ctx)
		suit.NoError(err)
		testObj := obj.(*TestObject)
		suit.NotNil(testObj)
		active[i] = testObj
	}
	suit.assertEquals(100, suit.pool.GetNumActive())
	suit.assertEquals(0, suit.pool.GetNumIdle())
	for i := 0; i < 100; i++ {
		obj := active[i]
		if debugTest {
			fmt.Printf("TestMaxIdle ReturnObject %v \n", obj)
		}
		err := suit.pool.ReturnObject(ctx, obj)
		suit.NoError(err)
		suit.assertEquals(99-i, suit.pool.GetNumActive())
		idle := suit.pool.Config.MaxIdle
		if i < idle {
			idle = i + 1
		}
		suit.assertEquals(idle, suit.pool.GetNumIdle())
	}
}

func (suit *PoolTestSuite) TestMaxIdleZero() {
	ctx := context.Background()
	suit.pool.Config.MaxTotal = 100
	suit.pool.Config.MaxIdle = 0
	active := make([]*TestObject, 100)
	for i := 0; i < 100; i++ {
		obj, err := suit.pool.BorrowObject(ctx)
		suit.NoError(err)
		testObj := obj.(*TestObject)
		suit.NotNil(testObj)
		active[i] = testObj
	}
	suit.assertEquals(100, suit.pool.GetNumActive())
	suit.assertEquals(0, suit.pool.GetNumIdle())
	for i := 0; i < 100; i++ {
		suit.pool.ReturnObject(ctx, active[i])
		suit.assertEquals(99-i, suit.pool.GetNumActive())
		suit.assertEquals(0, suit.pool.GetNumIdle())
	}
}

func (suit *PoolTestSuite) TestMaxTotal() {
	ctx := context.Background()
	suit.pool.Config.MaxTotal = 3
	suit.pool.Config.BlockWhenExhausted = false

	suit.NoErrorWithResult(suit.pool.BorrowObject(ctx))
	suit.NoErrorWithResult(suit.pool.BorrowObject(ctx))
	suit.NoErrorWithResult(suit.pool.BorrowObject(ctx))
	_, err := suit.pool.BorrowObject(ctx)
	suit.Error(err)
}

func (suit *PoolTestSuite) TestTimeoutNoLeak() {
	suit.pool.Config.MaxTotal = 2
	suit.pool.Config.BlockWhenExhausted = true

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	obj, err := suit.pool.BorrowObject(ctx)
	suit.NoError(err)

	obj2 := suit.NoErrorWithResult(suit.pool.BorrowObject(ctx))
	err3 := suit.ErrorWithResult(suit.pool.BorrowObject(ctx))
	suit.IsType(&NoSuchElementErr{}, err3, err3)

	ctx, cancel2 := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel2()

	suit.NoError(suit.pool.ReturnObject(ctx, obj2))
	suit.NoError(suit.pool.ReturnObject(ctx, obj))

	suit.NoErrorWithResult(suit.pool.BorrowObject(ctx))
	suit.NoErrorWithResult(suit.pool.BorrowObject(ctx))
}

func (suit *PoolTestSuite) TestMaxTotalZero() {
	ctx := context.Background()
	suit.pool.Config.MaxTotal = 0
	suit.pool.Config.BlockWhenExhausted = false
	err := suit.ErrorWithResult(suit.pool.BorrowObject(ctx))
	suit.Error(err)
	//fail("Expected NoSuchElementException");
}

func (suit *PoolTestSuite) TestMaxTotalUnderLoad() {
	// Config
	numGoroutines := 199 // And main goroutine makes a round 200.
	numIter := 20
	delay := 25
	maxTotal := 10

	suit.factory.maxTotal = maxTotal
	suit.pool.Config.MaxTotal = maxTotal
	suit.pool.Config.BlockWhenExhausted = true
	suit.pool.Config.TimeBetweenEvictionRunsMillis = int64(-1)

	// Start goroutines to borrow objects
	goroutineArgs := make([]*TestGoroutineArg, numGoroutines)
	resultChans := make([]chan TestGoroutineResult, numGoroutines)
	ctx := context.Background()
	for i := 0; i < numGoroutines; i++ {
		// Factor of 2 on iterations so main goroutine does work whilst other
		// goroutines are running. Factor of 2 on delay so average delay for
		// other goroutines == actual delay for main goroutine
		goroutineArgs[i] = NewTesGoroutineArgSimple(suit.pool, numIter*2, delay*2, true)
		resultChans[i] = goroutineRun(ctx, goroutineArgs[i])
	}
	// Give the goroutines a chance to start doing some work
	time.Sleep(time.Duration(5000) * time.Millisecond)

	for i := 0; i < numIter; i++ {
		var obj interface{}
		time.Sleep(time.Duration(delay) * time.Millisecond)

		obj, err := suit.pool.BorrowObject(ctx)
		suit.NoError(err)
		// Under load, observed _numActive > _maxTotal
		if suit.pool.GetNumActive() > suit.pool.Config.MaxTotal {
			suit.Fail("Too many active objects")
		}
		time.Sleep(time.Duration(delay) * time.Millisecond)
		if obj != nil {
			suit.NoError(suit.pool.ReturnObject(ctx, obj))
		}
	}

	for i := 0; i < numGoroutines; i++ {
		result := <-resultChans[i]
		close(resultChans[i])
		if result.failed {
			suit.Fail(fmt.Sprintf("Goroutine %v failed: %v", i, result.error.Error()))
		}
	}
}

func (suit *PoolTestSuite) TestStartAndStopEvictor() {
	defer leaktest.Check(suit.T())()
	// set up pool without evictor
	suit.pool.Config.MaxIdle = 6
	suit.pool.Config.MaxTotal = 6
	suit.pool.Config.NumTestsPerEvictionRun = 6
	suit.pool.Config.MinEvictableIdleTimeMillis = int64(100)

	ctx := context.Background()
	for j := 0; j < 2; j++ {
		// populate the pool
		{
			active := make([]*TestObject, 6)
			for i := 0; i < 6; i++ {
				active[i] = suit.NoErrorWithResult(suit.pool.BorrowObject(ctx)).(*TestObject)
			}
			for i := 0; i < 6; i++ {
				suit.NoError(suit.pool.ReturnObject(ctx, active[i]))
			}
		}

		// note that it stays populated
		suit.Equal(6, suit.pool.GetNumIdle(), "Should have 6 idle")

		// start the evictor
		suit.pool.Config.TimeBetweenEvictionRunsMillis = int64(50)

		//re config evictor
		suit.pool.StartEvictor()

		// wait a second (well, .2 seconds)
		time.Sleep(time.Duration(200) * time.Millisecond)

		// assert that the evictor has cleared out the pool
		suit.Equal(0, suit.pool.GetNumIdle(), "Should have 0 idle")

		// stop the evictor
		suit.pool.startEvictor(int64(0))
	}
}

func (suit *PoolTestSuite) TestStartAndStopEvictorConcurrent() {
	defer leaktest.Check(suit.T())()
	// set up pool without evictor
	suit.pool.Config.MaxIdle = 6
	suit.pool.Config.MaxTotal = 100
	suit.pool.Config.NumTestsPerEvictionRun = 6
	suit.pool.Config.MinEvictableIdleTimeMillis = int64(100)

	ctx := context.Background()
	testWG := sync.WaitGroup{}
	testWG.Add(101)
	for i := 0; i < 100; i++ {
		go func(idx int) {
			time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
			suit.pool.startEvictor(int64(10 + idx))
			testWG.Done()
		}(i)
	}
	go func() {
		for j := 0; j < 10; j++ {
			// populate the pool
			active := make([]*TestObject, 6)
			for i := 0; i < 6; i++ {
				active[i] = suit.NoErrorWithResult(suit.pool.BorrowObject(ctx)).(*TestObject)
			}
			for i := 0; i < 6; i++ {
				suit.NoError(suit.pool.ReturnObject(ctx, active[i]))
			}
			// wait a second (well, .1 seconds)
			time.Sleep(time.Duration(10) * time.Millisecond)
		}
		testWG.Done()
	}()
	testWG.Wait()
	// wait a second (well, .2 seconds)
	time.Sleep(time.Duration(200) * time.Millisecond)

	// assert that the evictor has cleared out the pool
	suit.Equal(0, suit.pool.GetNumIdle(), "Should have 0 idle")
	suit.pool.startEvictor(int64(0))
}

func (suit *PoolTestSuite) TestEvictionWithNegativeNumTests() {
	// when numTestsPerEvictionRun is negative, it represents a fraction of the idle objects to test
	suit.pool.Config.MaxIdle = 6
	suit.pool.Config.MaxTotal = 6
	suit.pool.Config.NumTestsPerEvictionRun = -2
	suit.pool.Config.MinEvictableIdleTimeMillis = int64(50)

	suit.pool.Config.TimeBetweenEvictionRunsMillis = int64(100)
	ctx := context.Background()
	suit.pool.StartEvictor()

	active := make([]*TestObject, 6)
	for i := 0; i < 6; i++ {
		active[i] = suit.NoErrorWithResult(suit.pool.BorrowObject(ctx)).(*TestObject)
	}
	for i := 0; i < 6; i++ {
		suit.NoError(suit.pool.ReturnObject(ctx, active[i]))
	}

	time.Sleep(time.Duration(100) * time.Millisecond)
	suit.True(suit.pool.GetNumIdle() <= 6, "Should at most 6 idle, found %v", suit.pool.GetNumIdle())
	time.Sleep(time.Duration(100) * time.Millisecond)
	suit.True(suit.pool.GetNumIdle() <= 3, "Should at most 3 idle, found %v", suit.pool.GetNumIdle())
	time.Sleep(time.Duration(100) * time.Millisecond)
	suit.True(suit.pool.GetNumIdle() <= 2, "Should be at most 2 idle, found %v", suit.pool.GetNumIdle())
	time.Sleep(time.Duration(100) * time.Millisecond)
	suit.Equal(0, suit.pool.GetNumIdle(), "Should be zero idle, found %v", suit.pool.GetNumIdle())
}

func (suit *PoolTestSuite) TestEviction() {
	suit.pool.Config.MaxIdle = 500
	suit.pool.Config.MaxTotal = 500
	suit.pool.Config.NumTestsPerEvictionRun = 100
	suit.pool.Config.MinEvictableIdleTimeMillis = int64(250)
	suit.pool.Config.TimeBetweenEvictionRunsMillis = int64(500)
	ctx := context.Background()
	suit.pool.StartEvictor()

	suit.pool.Config.TestWhileIdle = true
	active := make([]*TestObject, 500)

	for i := 0; i < 500; i++ {
		active[i] = suit.NoErrorWithResult(suit.pool.BorrowObject(ctx)).(*TestObject)
	}
	for i := 0; i < 500; i++ {
		suit.NoError(suit.pool.ReturnObject(ctx, active[i]))
	}

	time.Sleep(time.Duration(1000) * time.Millisecond)
	suit.True(suit.pool.GetNumIdle() < 500, "Should be less than 500 idle, found %v", suit.pool.GetNumIdle())
	time.Sleep(time.Duration(600) * time.Millisecond)
	suit.True(suit.pool.GetNumIdle() < 400, "Should be less than 400 idle, found %v", suit.pool.GetNumIdle())
	time.Sleep(time.Duration(600) * time.Millisecond)
	suit.True(suit.pool.GetNumIdle() < 300, "Should be less than 300 idle, found %v", suit.pool.GetNumIdle())
	time.Sleep(time.Duration(600) * time.Millisecond)
	suit.True(suit.pool.GetNumIdle() < 200, "Should be less than 200 idle, found %v", suit.pool.GetNumIdle())
	time.Sleep(time.Duration(600) * time.Millisecond)
	suit.True(suit.pool.GetNumIdle() < 100, "Should be less than 100 idle, found %v", suit.pool.GetNumIdle())
	time.Sleep(time.Duration(600) * time.Millisecond)
	suit.Equal(0, suit.pool.GetNumIdle(), "Should be zero idle, found %v", suit.pool.GetNumIdle())

	for i := 0; i < 500; i++ {
		active[i] = suit.NoErrorWithResult(suit.pool.BorrowObject(ctx)).(*TestObject)
	}
	for i := 0; i < 500; i++ {
		suit.NoError(suit.pool.ReturnObject(ctx, active[i]))
	}

	time.Sleep(time.Duration(1000) * time.Millisecond)
	suit.True(suit.pool.GetNumIdle() < 500, "Should be less than 500 idle, found %v", suit.pool.GetNumIdle())
	time.Sleep(time.Duration(600) * time.Millisecond)
	suit.True(suit.pool.GetNumIdle() < 400, "Should be less than 400 idle, found %v", suit.pool.GetNumIdle())
	time.Sleep(time.Duration(600) * time.Millisecond)
	suit.True(suit.pool.GetNumIdle() < 300, "Should be less than 300 idle, found %v", suit.pool.GetNumIdle())
	time.Sleep(time.Duration(600) * time.Millisecond)
	suit.True(suit.pool.GetNumIdle() < 200, "Should be less than 200 idle, found %v", suit.pool.GetNumIdle())
	time.Sleep(time.Duration(600) * time.Millisecond)
	suit.True(suit.pool.GetNumIdle() < 100, "Should be less than 100 idle, found %v", suit.pool.GetNumIdle())
	time.Sleep(time.Duration(600) * time.Millisecond)
	suit.Equal(0, suit.pool.GetNumIdle(), "Should be zero idle, found %v", suit.pool.GetNumIdle())
}

type TestEvictionPolicy struct {
	callCount concurrent.AtomicInteger
}

func (p *TestEvictionPolicy) Evict(config *EvictionConfig, underTest *PooledObject, idleCount int) bool {
	if p.callCount.IncrementAndGet() > 1500 {
		return true
	}
	return false
}

var TestEvictionPolicyName = "github.com/jolestar/go-commons-pool/TestEvictionPolicy"

func (suit *PoolTestSuite) TestEvictionPolicy() {
	suit.pool.Config.MaxIdle = 500
	suit.pool.Config.MaxTotal = 500
	suit.pool.Config.NumTestsPerEvictionRun = 500
	suit.pool.Config.MinEvictableIdleTimeMillis = int64(250)
	suit.pool.Config.TimeBetweenEvictionRunsMillis = int64(500)
	suit.pool.StartEvictor()
	suit.pool.Config.TestWhileIdle = true
	evictionPolicy := new(TestEvictionPolicy)

	RegistryEvictionPolicy(TestEvictionPolicyName, evictionPolicy)

	_, ok := suit.pool.getEvictionPolicy().(*DefaultEvictionPolicy)
	suit.True(ok, "EvictionPolicy is not default policy")

	suit.pool.Config.EvictionPolicyName = TestEvictionPolicyName
	suit.Equal(evictionPolicy, suit.pool.getEvictionPolicy())

	ctx := context.Background()
	active := make([]*TestObject, 500)
	for i := 0; i < 500; i++ {
		active[i] = suit.NoErrorWithResult(suit.pool.BorrowObject(ctx)).(*TestObject)
	}
	for i := 0; i < 500; i++ {
		suit.NoError(suit.pool.ReturnObject(ctx, active[i]))
	}

	// Eviction policy ignores first 1500 attempts to evict and then always
	// evicts. After 1s, there should have been two runs of 500 tests so no
	// evictions
	time.Sleep(time.Duration(1000) * time.Millisecond)
	suit.Equal(500, suit.pool.GetNumIdle(), "Should be 500 idle")
	// A further 1s wasn't enough so allow 2s for the evictor to clear out
	// all of the idle objects.
	time.Sleep(time.Duration(2000) * time.Millisecond)
	suit.Equal(0, suit.pool.GetNumIdle(), "Should be 0 idle")
}

func (suit *PoolTestSuite) TestEvictionSoftMinIdle() {
	suit.pool.Config.MaxIdle = 5
	suit.pool.Config.MaxTotal = 5
	suit.pool.Config.NumTestsPerEvictionRun = 5
	suit.pool.Config.MinEvictableIdleTimeMillis = int64(3000)
	suit.pool.Config.SoftMinEvictableIdleTimeMillis = int64(1000)
	suit.pool.Config.MinIdle = 2

	ctx := context.Background()
	active := make([]*TestObject, 5)
	for i := 0; i < 5; i++ {
		active[i] = suit.NoErrorWithResult(suit.pool.BorrowObject(ctx)).(*TestObject)
	}

	for i := 0; i < 5; i++ {
		suit.pool.ReturnObject(ctx, active[i])
	}

	// Soft evict all but minIdle(2)
	time.Sleep(time.Duration(1500) * time.Millisecond)
	suit.pool.evict(ctx)
	suit.Equal(2, suit.pool.GetNumIdle(), "Idle count different than expected.")

	// Hard evict the rest.
	time.Sleep(time.Duration(1600) * time.Millisecond)
	suit.pool.evict(ctx)
	suit.Equal(0, suit.pool.GetNumIdle(), "Idle count different than expected.")
}

func (suit *PoolTestSuite) TestEvictionInvalid() {
	ctx := context.Background()
	suit.pool = NewObjectPoolWithDefaultConfig(ctx, NewPooledObjectFactory(
		func(context.Context) (interface{}, error) {
			return &TestObject{}, nil
		}, nil, func(ctx context.Context, object *PooledObject) bool {
			if debugTest {
				fmt.Printf("TestEvictionInvalid valid object %v \n", object)
			}
			time.Sleep(time.Duration(1000) * time.Millisecond)
			return false
		}, nil, nil))

	suit.pool.Config.MaxIdle = 1
	suit.pool.Config.MaxTotal = 1
	suit.pool.Config.TestOnBorrow = false
	suit.pool.Config.TestOnReturn = false
	suit.pool.Config.TestWhileIdle = true
	suit.pool.Config.MinEvictableIdleTimeMillis = int64(100000)
	suit.pool.Config.NumTestsPerEvictionRun = 1

	p := suit.NoErrorWithResult(suit.pool.BorrowObject(ctx))
	suit.NoError(suit.pool.ReturnObject(ctx, p))

	// Run eviction in a separate goroutine
	go func() {
		if debugTest {
			fmt.Println("TestEvictionInvalid evict goroutine.")
		}
		suit.pool.evict(suit.pool.Config.EvitionContext)
	}()

	// Sleep to make sure evictor has started
	time.Sleep(300 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	err := suit.ErrorWithResult(suit.pool.borrowObject(ctx))
	suit.IsType(err, &NoSuchElementErr{})

	// Make sure evictor has finished
	time.Sleep(1000 * time.Millisecond)
	// Should have an empty pool
	suit.Equal(0, suit.pool.GetNumIdle(), "Idle count different than expected.")
	suit.Equal(0, suit.pool.GetNumActive(), "Total count different than expected.")
}

func (suit *PoolTestSuite) TestConcurrentInvalidate() {
	ctx := context.Background()
	// Get allObjects and idleObjects loaded with some instances
	nObjects := 1000
	suit.pool.Config.MaxTotal = nObjects
	suit.pool.Config.MaxIdle = nObjects
	active := make([]*TestObject, nObjects)
	for i := 0; i < nObjects; i++ {
		active[i] = suit.NoErrorWithResult(suit.pool.BorrowObject(ctx)).(*TestObject)
	}
	for i := 0; i < nObjects; i++ {
		if i%2 == 0 {
			suit.NoError(suit.pool.ReturnObject(ctx, active[i]))
		}
	}
	nGoroutines := 20
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
		// Launch nGoroutines goroutines all trying to invalidate the target
		results := make(chan bool, nGoroutines)
		for i := 0; i < nGoroutines; i++ {
			go func(pool *ObjectPool, obj *TestObject) {
				err := pool.InvalidateObject(ctx, obj)
				_, ok := err.(*IllegalStateErr)
				if err != nil && !ok {
					results <- false
					if debugTest {
						fmt.Printf("TestConcurrentInvalidate InvalidateObject error:%v, obj: %v \n", err, obj)
					}
				} else {
					results <- true
				}
			}(suit.pool, active[targ])
		}
		for i := 0; i < nGoroutines; i++ {
			done := <-results
			suit.True(done)
		}
	}
	suit.Equal(nIterations, suit.pool.GetDestroyedCount())
}

func sleep(millisecond int) {
	time.Sleep(time.Duration(millisecond) * time.Millisecond)
}

func (suit *PoolTestSuite) TestMinIdle() {
	ctx := context.Background()
	suit.pool.Config.MaxIdle = 500
	suit.pool.Config.MinIdle = 5
	suit.pool.Config.MaxTotal = 10
	suit.pool.Config.NumTestsPerEvictionRun = 0
	suit.pool.Config.MinEvictableIdleTimeMillis = int64(50)
	suit.pool.Config.TimeBetweenEvictionRunsMillis = int64(100)
	suit.pool.Config.TestWhileIdle = true
	suit.pool.StartEvictor()

	sleep(150)
	suit.Equal(5, suit.pool.GetNumIdle(), "Should be 5 idle, found %v", suit.pool.GetNumIdle())

	active := make([]*TestObject, 5)
	active[0] = suit.NoErrorWithResult(suit.pool.BorrowObject(ctx)).(*TestObject)
	sleep(150)
	suit.Equal(5, suit.pool.GetNumIdle(), "Should be 5 idle, found %v", suit.pool.GetNumIdle())

	for i := 1; i < 5; i++ {
		active[i] = suit.NoErrorWithResult(suit.pool.BorrowObject(ctx)).(*TestObject)
	}

	sleep(150)
	suit.Equal(5, suit.pool.GetNumIdle(), "Should be 5 idle, found %v", suit.pool.GetNumIdle())

	for i := 0; i < 5; i++ {
		suit.NoError(suit.pool.ReturnObject(ctx, active[i]))
	}
	sleep(150)
	suit.Equal(10, suit.pool.GetNumIdle(), "Should be 10 idle, found %v", suit.pool.GetNumIdle())
}

func (suit *PoolTestSuite) TestMinIdleMaxTotal() {
	suit.pool.Config.MaxIdle = 500
	suit.pool.Config.MinIdle = 5
	suit.pool.Config.MaxTotal = 10
	suit.pool.Config.NumTestsPerEvictionRun = 0
	suit.pool.Config.MinEvictableIdleTimeMillis = int64(50)
	suit.pool.Config.TimeBetweenEvictionRunsMillis = int64(100)
	suit.pool.Config.TestWhileIdle = true
	ctx := context.Background()
	suit.pool.StartEvictor()

	sleep(150)
	suit.Equal(5, suit.pool.GetNumIdle(), "Should be 5 idle, found %v", suit.pool.GetNumIdle())

	active := make([]*TestObject, 10)
	sleep(150)
	suit.Equal(5, suit.pool.GetNumIdle(), "Should be 5 idle, found %v", suit.pool.GetNumIdle())

	for i := 0; i < 5; i++ {
		active[i] = suit.NoErrorWithResult(suit.pool.BorrowObject(ctx)).(*TestObject)
	}
	sleep(150)
	suit.Equal(5, suit.pool.GetNumIdle(), "Should be 5 idle, found %v", suit.pool.GetNumIdle())

	for i := 0; i < 5; i++ {
		suit.NoError(suit.pool.ReturnObject(ctx, active[i]))
	}
	sleep(150)
	suit.Equal(10, suit.pool.GetNumIdle(), "Should be 10 idle, found %v", suit.pool.GetNumIdle())

	for i := 0; i < 10; i++ {
		active[i] = suit.NoErrorWithResult(suit.pool.BorrowObject(ctx)).(*TestObject)
	}
	sleep(150)
	suit.Equal(0, suit.pool.GetNumIdle(), "Should be 0 idle, found %v", suit.pool.GetNumIdle())

	for i := 0; i < 10; i++ {
		suit.NoError(suit.pool.ReturnObject(ctx, active[i]))
	}
	sleep(150)
	suit.Equal(10, suit.pool.GetNumIdle(), "Should be 10 idle, found %v", suit.pool.GetNumIdle())
}

func runTestGoroutines(ctx context.Context, t *testing.T, numGoroutines int, iterations int, delay int, borrowTimeout time.Duration, testPool *ObjectPool) {
	arg := NewTestGoroutineArg(testPool, iterations, delay, borrowTimeout, delay, true, nil)
	resultChans := make([]chan TestGoroutineResult, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		resultChans[i] = goroutineRun(ctx, arg)
	}
	results := make([]TestGoroutineResult, numGoroutines)
	var failedGoroutines []int
	for i := 0; i < numGoroutines; i++ {
		result := <-resultChans[i]
		results[i] = result
		close(resultChans[i])
		if result.failed {
			failedGoroutines = append(failedGoroutines, i)
		}
	}
	if len(failedGoroutines) > 0 {
		for _, t := range failedGoroutines {
			if debugTest {
				fmt.Printf("Goroutine %v failed %v \n", t, results[t].error)
			}
		}
		assert.Fail(t, fmt.Sprintf("Goroutine %v failed", failedGoroutines))
	}
}

func (suit *PoolTestSuite) TestGoroutineed1() {
	suit.pool.Config.MaxTotal = 15
	suit.pool.Config.MaxIdle = 15
	ctx := context.Background()
	runTestGoroutines(ctx, suit.T(), 20, 100, 50, 1*time.Second, suit.pool)
}

func (suit *PoolTestSuite) TestMaxTotalInvariant() {
	maxTotal := 15
	suit.factory.evenValid = false    // Every other validation fails
	suit.factory.destroyLatency = 100 // Destroy takes 100 ms
	suit.factory.maxTotal = maxTotal  // (makes - destroys) bound
	suit.factory.enableValidation = true
	suit.pool.Config.MaxTotal = maxTotal
	suit.pool.Config.MaxIdle = -1
	suit.pool.Config.TestOnReturn = true
	ctx := context.Background()
	runTestGoroutines(ctx, suit.T(), 5, 10, 50, 1*time.Second, suit.pool)
}

func concurrentBorrowAndEvictGoroutine(ctx context.Context, borrow bool, pool *ObjectPool) chan interface{} {
	ch := make(chan interface{}, 1)
	go func(borrow bool, pool *ObjectPool) {
		if borrow {
			obj, _ := pool.BorrowObject(ctx)
			ch <- obj
		} else {
			pool.evict(ctx)
			ch <- 1
		}
	}(borrow, pool)
	return ch
}

func (suit *PoolTestSuite) TestConcurrentBorrowAndEvict() {
	ctx := context.Background()
	suit.pool.Config.MaxTotal = 1
	suit.NoError(suit.pool.AddObject(ctx))

	for i := 0; i < 5000; i++ {
		ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
		defer cancel()

		one := concurrentBorrowAndEvictGoroutine(ctx, true, suit.pool)
		two := concurrentBorrowAndEvictGoroutine(ctx, false, suit.pool)

		obj := <-one
		close(one)
		<-two
		close(two)
		suit.NotNil(obj)
		suit.NoError(suit.pool.ReturnObject(ctx, obj))

		//Uncomment suit for a progress indication
		//		if i%10 == 0 {
		//			fmt.Println(i)
		//		}
	}
}

//Verifies that concurrent goroutines never "share" instances
func (suit *PoolTestSuite) TestNoInstanceOverlap() {
	maxTotal := 5
	numGoroutines := 100
	delay := 1
	iterations := 1000
	suit.pool.Config.MaxTotal = maxTotal
	suit.pool.Config.MaxIdle = maxTotal
	suit.pool.Config.TestOnBorrow = true
	suit.pool.Config.BlockWhenExhausted = true
	ctx := context.Background()
	runTestGoroutines(ctx, suit.T(), numGoroutines, iterations, delay, time.Duration(0), suit.pool)
	suit.Equal(0, suit.pool.GetDestroyedByBorrowValidationCount())
}

func (suit *PoolTestSuite) TestWhenExhaustedBlockClosePool() {
	suit.pool.Config.MaxTotal = 1
	suit.pool.Config.BlockWhenExhausted = true

	ctx := context.Background()
	obj1 := suit.NoErrorWithResult(suit.pool.BorrowObject(ctx))

	// Make sure an object was obtained
	suit.NotNil(obj1)

	// Create a separate goroutine to try and borrow another object
	ch := waitTestGoroutine(ctx, suit.pool, 200)
	// Give wtt time to start
	sleep(200)

	// close the pool (Bug POOL-189)
	suit.pool.Close(ctx)

	// Give interrupt time to take effect
	sleep(200)

	// Check goroutine was interrupted
	result := <-ch
	close(ch)
	suit.IsType(&collections.InterruptedErr{}, result.error, result.error.Error())
}

func waitTestGoroutine(ctx context.Context, pool *ObjectPool, pause int) chan TestGoroutineResult {
	ch := make(chan TestGoroutineResult, 1)
	go func() {
		result := TestGoroutineResult{}
		result.preborrow = currentTimeMillis()
		obj, err := pool.BorrowObject(ctx)
		result.objectID = obj
		result.postborrow = currentTimeMillis()
		if err == nil {
			sleep(pause)
			pool.ReturnObject(ctx, obj)
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

func (suit *PoolTestSuite) TestFIFO() {
	ctx := context.Background()
	suit.pool.Config.Lifo = false
	suit.NoError(suit.pool.AddObject(ctx)) // "0"
	suit.NoError(suit.pool.AddObject(ctx)) // "1"
	suit.NoError(suit.pool.AddObject(ctx)) // "2"
	suit.Equal(getNthObject(0), suit.NoErrorWithResult(suit.pool.BorrowObject(ctx)), "Oldest")
	suit.Equal(getNthObject(1), suit.NoErrorWithResult(suit.pool.BorrowObject(ctx)), "Middle")
	suit.Equal(getNthObject(2), suit.NoErrorWithResult(suit.pool.BorrowObject(ctx)), "Youngest")
	o := suit.NoErrorWithResult(suit.pool.BorrowObject(ctx))
	suit.Equal(getNthObject(3), o, "new-3")
	suit.NoError(suit.pool.ReturnObject(ctx, o))
	suit.Equal(o, suit.NoErrorWithResult(suit.pool.BorrowObject(ctx)), "returned-3")
	suit.Equal(getNthObject(4), suit.NoErrorWithResult(suit.pool.BorrowObject(ctx)), "new-4")
}

func (suit *PoolTestSuite) TestLIFO() {
	ctx := context.Background()
	suit.pool.Config.Lifo = true
	suit.NoError(suit.pool.AddObject(ctx)) // "0"
	suit.NoError(suit.pool.AddObject(ctx)) // "1"
	suit.NoError(suit.pool.AddObject(ctx)) // "2"
	suit.Equal(getNthObject(2), suit.NoErrorWithResult(suit.pool.BorrowObject(ctx)), "Youngest")
	suit.Equal(getNthObject(1), suit.NoErrorWithResult(suit.pool.BorrowObject(ctx)), "Middle")
	suit.Equal(getNthObject(0), suit.NoErrorWithResult(suit.pool.BorrowObject(ctx)), "Oldest")
	o := suit.NoErrorWithResult(suit.pool.BorrowObject(ctx))
	suit.Equal(getNthObject(3), o, "new-3")
	suit.NoError(suit.pool.ReturnObject(ctx, o))
	suit.Equal(o, suit.NoErrorWithResult(suit.pool.BorrowObject(ctx)), "returned-3")
	suit.Equal(getNthObject(4), suit.NoErrorWithResult(suit.pool.BorrowObject(ctx)), "new-4")
}

func (suit *PoolTestSuite) TestAddObject() {
	ctx := context.Background()
	suit.Equal(0, suit.pool.GetNumIdle(), "should be zero idle")
	suit.NoError(suit.pool.AddObject(ctx))
	suit.Equal(1, suit.pool.GetNumIdle(), "should be one idle")
	suit.Equal(0, suit.pool.GetNumActive(), "should be zero active")
	obj := suit.NoErrorWithResult(suit.pool.BorrowObject(ctx))
	suit.Equal(0, suit.pool.GetNumIdle(), "should be zero idle")
	suit.Equal(1, suit.pool.GetNumActive(), "should be one active")
	suit.NoError(suit.pool.ReturnObject(ctx, obj))
	suit.Equal(1, suit.pool.GetNumIdle(), "should be one idle")
	suit.Equal(0, suit.pool.GetNumActive(), "should be zero active")
}

//TODO
//func (suit *PoolTestSuite)  TestBorrowObjectFairness() {}

/**
 * On first borrow, first object fails validation, second object is OK.
 * Subsequent borrows are OK. This was POOL-152.
 */
func (suit *PoolTestSuite) TestBrokenFactoryShouldNotBlockPool() {
	maxTotal := 1

	suit.factory.maxTotal = maxTotal
	suit.pool.Config.MaxTotal = maxTotal
	suit.pool.Config.BlockWhenExhausted = true
	suit.pool.Config.TestOnBorrow = true

	ctx := context.Background()

	// First borrow object will need to create a new object which will fail
	// validation.
	suit.factory.setValid(false)
	obj, ex := suit.pool.BorrowObject(ctx)
	// Failure expected
	_, ok := ex.(*NoSuchElementErr)
	suit.True(ok, "expect NoSuchElementErr, but get: %v", reflect.TypeOf(ex))
	suit.Nil(obj)

	// Configure factory to create valid objects so subsequent borrows work
	suit.factory.setValid(true)

	// Subsequent borrows should be OK
	obj = suit.NoErrorWithResult(suit.pool.BorrowObject(ctx))
	suit.NoError(suit.pool.ReturnObject(ctx, obj))
}

/*
 * Test multi-goroutineed pool access.
 * Multiple goroutines, but maxTotal only allows half the goroutines to succeed.
 *
 * This test was prompted by Continuum build failures in the Commons DBCP test case:
 * TestPerUserPoolDataSource.testMultipleGoroutines2()
 * Let's see if the suit fails on Continuum too!
 */
func (suit *PoolTestSuite) TestMaxWaitMultiGoroutineed() {
	maxWait := 500          // wait for connection
	holdTime := 2 * maxWait // how long to hold connection
	goroutines := 10        // number of goroutines to grab the object initially
	suit.pool.Config.BlockWhenExhausted = true
	suit.pool.Config.MaxTotal = goroutines
	// Create enough goroutines so half the goroutines will have to wait
	resultChans := make([]chan TestGoroutineResult, goroutines*2)
	origin := currentTimeMillis() - 1000
	for i := 0; i < len(resultChans); i++ {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(maxWait)*time.Millisecond)
		defer cancel()
		resultChans[i] = waitTestGoroutine(ctx, suit.pool, holdTime)
	}
	failed := 0
	results := make([]TestGoroutineResult, len(resultChans))
	for i := 0; i < len(resultChans); i++ {
		ch := resultChans[i]
		result := <-ch
		close(ch)
		results[i] = result
		if result.error != nil {
			failed++
		}
	}
	if debugTest || len(resultChans)/2 != failed {
		fmt.Println(
			"MaxWait: ", maxWait,
			" HoldTime: ", holdTime,
			" MaxTotal: ", goroutines,
			" Goroutines: ", len(resultChans),
			" Failed: ", failed)
		for _, result := range results {
			fmt.Println(
				"Preborrow: ", (result.preborrow - origin),
				" Postborrow: ", (result.postborrow - origin),
				" BorrowTime: ", (result.postborrow - result.preborrow),
				" PostReturn: ", (result.postreturn - origin),
				" Ended: ", (result.ended - origin),
				" ObjId: ", result.objectID)
		}
	}
	suit.Equal(len(resultChans)/2, failed, "Expected half the goroutines to fail")
}

/**
* Test the following scenario:
*   Goroutine 1 borrows an instance
*   Goroutine 2 starts to borrow another instance before goroutine 1 returns its instance
*   Goroutine 1 returns its instance while goroutine 2 is validating its newly created instance
* The test verifies that the instance created by Goroutine 2 is not leaked.
 */
func (suit *PoolTestSuite) TestMakeConcurrentWithReturn() {
	suit.pool.Config.TestOnBorrow = true
	suit.factory.setValid(true)
	ctx := context.Background()
	// Borrow and return an instance, with a short wait
	ch := waitTestGoroutine(ctx, suit.pool, 200)
	sleep(50) // wait for validation to succeed
	// Slow down validation and borrow an instance
	suit.factory.setValidateLatency(400)
	instance := suit.NoErrorWithResult(suit.pool.BorrowObject(ctx))
	// Now make sure that we have not leaked an instance
	suit.Equal(suit.factory.makeCounter, suit.pool.GetNumIdle()+1)
	suit.NoError(suit.pool.ReturnObject(ctx, instance))
	suit.Equal(suit.factory.makeCounter, suit.pool.GetNumIdle())
	<-ch
	close(ch)
}

/**
 * Verify that goroutines waiting on a depleted pool get served when a checked out object is
 * invalidated.
 *
 * JIRA: POOL-240
 */
func (suit *PoolTestSuite) TestInvalidateFreesCapacity() {
	suit.pool.Config.MaxTotal = 2
	suit.pool.Config.BlockWhenExhausted = true

	// Borrow an instance and hold if for 5 seconds
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	ch1 := waitTestGoroutine(ctx, suit.pool, 5000)

	// Borrow another instance
	ctx, cancel = context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	obj := suit.NoErrorWithResult(suit.pool.BorrowObject(ctx))

	// Launch another goroutine - will block, but fail in 500 ms
	ctx, cancel = context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	ch2 := waitTestGoroutine(ctx, suit.pool, 100)

	// Invalidate the object borrowed by suit goroutine - should allow goroutine2 to create
	sleep(20)
	suit.NoError(suit.pool.InvalidateObject(context.Background(), obj))
	sleep(600) // Wait for goroutine2 to timeout
	result2 := <-ch2
	close(ch2)
	if result2.error != nil {
		suit.Fail(result2.error.Error())
	}
	<-ch1
	close(ch1)
}

/**
* Verify that goroutines waiting on a depleted pool get served when a returning object fails
* validation.
*
* JIRA: POOL-240
*
 */
func (suit *PoolTestSuite) TestValidationFailureOnReturnFreesCapacity() {
	ctx := context.Background()
	suit.factory.setValid(false) // Validate will always fail
	suit.factory.enableValidation = true
	suit.pool.Config.MaxTotal = 2
	suit.pool.Config.TestOnReturn = true
	suit.pool.Config.TestOnBorrow = false

	// Borrow an instance and hold if for 5 seconds
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	ch1 := waitTestGoroutine(ctx, suit.pool, 5000)

	// Borrow another instance and return it after 500 ms (validation will fail)
	ctx, cancel = context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	ch2 := waitTestGoroutine(ctx, suit.pool, 500)
	sleep(50)

	// Try to borrow an object
	ctx, cancel = context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	obj := suit.NoErrorWithResult(suit.pool.BorrowObject(ctx))
	suit.NoError(suit.pool.ReturnObject(ctx, obj))

	<-ch1
	close(ch1)
	<-ch2
	close(ch2)
}

//TODO
//func (suit *PoolTestSuite) TestSwallowedExceptionListener() {
//}

// POOL-248
func (suit *PoolTestSuite) TestMultipleReturnOfSameObject() {
	ctx := context.Background()

	suit.Equal(0, suit.pool.GetNumActive())
	suit.Equal(0, suit.pool.GetNumIdle())

	obj := suit.NoErrorWithResult(suit.pool.BorrowObject(ctx))

	suit.Equal(1, suit.pool.GetNumActive())
	suit.Equal(0, suit.pool.GetNumIdle())

	suit.NoError(suit.pool.ReturnObject(ctx, obj))

	suit.Equal(0, suit.pool.GetNumActive())
	suit.Equal(1, suit.pool.GetNumIdle())

	err := suit.pool.ReturnObject(ctx, obj)
	_, ok := err.(*IllegalStateErr)
	suit.True(ok, "expect IllegalStatusErr, but get %v", reflect.TypeOf(err))
	suit.Equal(0, suit.pool.GetNumActive())
	suit.Equal(1, suit.pool.GetNumIdle())
}

// TODO POOL-259
//func (suit *PoolTestSuite) TestClientWaitStats() {
//}

// POOL-276
func (suit *PoolTestSuite) TestValidationOnCreateOnly() {
	suit.pool.Config.MaxTotal = 1
	suit.pool.Config.TestOnCreate = true
	suit.pool.Config.TestOnBorrow = false
	suit.pool.Config.TestOnReturn = false
	suit.pool.Config.TestWhileIdle = false

	ctx := context.Background()

	o1 := suit.NoErrorWithResult(suit.pool.BorrowObject(ctx))
	suit.Equal(getNthObject(0), o1)
	go func() {
		sleep(3000)
		suit.pool.ReturnObject(ctx, o1)
	}()

	o2 := suit.NoErrorWithResult(suit.pool.BorrowObject(ctx))
	suit.Equal(getNthObject(0), o2)

	suit.Equal(1, suit.factory.validateCounter)
}

/**
* Verifies that when a factory's makeObject produces instances that are not
* discernible by == , the pool can handle them.
*
* JIRA: POOL-283
 */
func (suit *PoolTestSuite) TestEqualsIndiscernible() {
	ctx := context.Background()
	pool := NewObjectPoolWithDefaultConfig(ctx, NewPooledObjectFactorySimple(func(context.Context) (interface{}, error) {
		return make(map[string]string), nil
	}))
	m1 := suit.NoErrorWithResult(pool.BorrowObject(ctx))
	m2 := suit.NoErrorWithResult(pool.BorrowObject(ctx))
	suit.NoError(pool.ReturnObject(ctx, m1))
	suit.NoError(pool.ReturnObject(ctx, m2))
	pool.Close(ctx)
}

/**
 * Verifies that when a borrowed object is mutated in a way that does not
 * preserve equality and hashcode, the pool can recognized it on return.
 *
 * JIRA: POOL-284
 */
func (suit *PoolTestSuite) TestMutable() {
	ctx := context.Background()
	pool := NewObjectPoolWithDefaultConfig(ctx, NewPooledObjectFactorySimple(func(context.Context) (interface{}, error) {
		return make(map[string]string), nil
	}))
	m1 := suit.NoErrorWithResult(pool.BorrowObject(ctx)).(map[string]string)
	m2 := suit.NoErrorWithResult(pool.BorrowObject(ctx)).(map[string]string)
	m1["k1"] = "v1"
	m2["k2"] = "v2"
	suit.NoError(pool.ReturnObject(ctx, m1))
	suit.NoError(pool.ReturnObject(ctx, m2))
	suit.Equal(2, pool.GetNumIdle())
	pool.Close(ctx)
}

/**
* Verifies that returning an object twice (without borrow in between) causes ISE
* but does not re-validate or re-passivate the instance.
*
* JIRA: POOL-285
 */
//TODO
//func (suit *PoolTestSuite) TestMultipleReturn() {
//}

func (suit *PoolTestSuite) TestAddError() {
	suit.pool.factory = nil
	err := suit.pool.AddObject(context.Background())
	suit.NotNil(err)
	suit.NotNil(err.Error())
}

func (suit *PoolTestSuite) TestClosePoolError() {
	ctx := context.Background()
	suit.pool.Close(ctx)
	err := suit.pool.AddObject(ctx)
	suit.NotNil(err)
}

func (suit *PoolTestSuite) TestMakeObjectError() {
	suit.factory.exceptionOnMake = true
	err := suit.pool.AddObject(context.Background())
	suit.NotNil(err)
}

func (suit *PoolTestSuite) TestAddObjectPassivateError() {
	suit.factory.exceptionOnPassivate = true
	suit.Equal(0, suit.pool.GetNumActive())
	suit.Equal(0, suit.pool.GetNumIdle())
	err := suit.pool.AddObject(context.Background())
	suit.EqualError(err, "passivate error")
	suit.Equal(0, suit.pool.GetNumActive())
	suit.Equal(0, suit.pool.GetNumIdle())
}

func (suit *PoolTestSuite) TestReturnObjectError() {
	obj := new(TestObject)
	err := suit.pool.ReturnObject(context.Background(), obj)
	suit.NotNil(err)
}

func (suit *PoolTestSuite) TestPreparePool() {
	ctx := context.Background()
	suit.pool.Config.MinIdle = 1
	suit.pool.Config.MaxTotal = 1
	suit.pool.PreparePool(ctx)
	suit.Equal(1, suit.pool.GetNumIdle())
	obj := suit.NoErrorWithResult(suit.pool.BorrowObject(ctx))
	suit.pool.PreparePool(ctx)
	suit.Equal(0, suit.pool.GetNumIdle())
	suit.pool.Config.MinIdle = 0
	suit.NoError(suit.pool.ReturnObject(ctx, obj))
	suit.pool.PreparePool(ctx)
	suit.Equal(1, suit.pool.GetNumIdle())
}

var perf bool

func init() {
	flag.BoolVar(&perf, "perf", false, "perf")
	flag.BoolVar(&debugTest, "debug_test", false, "debug_test")
}

func TestMain(m *testing.M) {
	flag.Parse()
	exit := 0
	if perf {
		perfMain(context.Background())
	} else {
		exit = m.Run()
	}
	os.Exit(exit)
}
