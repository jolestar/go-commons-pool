package collections

import (
	"fmt"
	"github.com/jolestar/go-commons-pool/concurrent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"math/rand"
	"reflect"
	"sync"
	"testing"
	"time"
)

var ONE = 1
var TWO = 2
var THREE = 3

type LinkedBlockDequeTestSuite struct {
	suite.Suite
	deque *LinkedBlockingDeque
}

func (suit *LinkedBlockDequeTestSuite) NoErrorWithResult(object interface{}, err error) interface{} {
	//suit.NotNil(object)
	suit.Nil(err)
	return object
}

func (suit *LinkedBlockDequeTestSuite) ErrorWithResult(object interface{}, err error) error {
	suit.Nil(object)
	suit.NotNil(err)
	return err
}

func TestLinkedBlockQueueTestSuite(t *testing.T) {
	suite.Run(t, new(LinkedBlockDequeTestSuite))
}

func (suit *LinkedBlockDequeTestSuite) SetupTest() {
	suit.deque = NewDeque(2)
}

func (suit *LinkedBlockDequeTestSuite) TestAddFirst() {
	suit.deque.AddFirst(ONE)
	suit.deque.AddFirst(TWO)
	//fmt.Println(deque.Size())
	suit.Equal(2, suit.deque.Size(), "deque size != 2")
	e := suit.deque.AddFirst(THREE)
	assert.NotNil(suit.T(), e, "deque can not add three element")
	suit.Equal(TWO, suit.deque.PollFirst())
}

func (suit *LinkedBlockDequeTestSuite) TestAddLast() {
	suit.deque.AddLast(ONE)
	suit.deque.AddLast(TWO)
	suit.Equal(2, suit.deque.Size())
	e := suit.deque.AddLast(THREE)
	assert.NotNil(suit.T(), e, "deque can not add three element")
	suit.Equal(ONE, suit.deque.PollFirst())
}

func (suit *LinkedBlockDequeTestSuite) TestOfferFirst() {
	suit.deque.OfferFirst(ONE)
	suit.deque.OfferFirst(TWO)
	suit.Equal(2, suit.deque.Size())
	suit.deque.OfferFirst(nil)
	suit.Equal(TWO, suit.deque.PollFirst())
}

func (suit *LinkedBlockDequeTestSuite) TestOfferLast() {
	suit.deque.OfferLast(ONE)
	suit.deque.OfferLast(TWO)
	suit.Equal(2, suit.deque.Size())
	suit.deque.OfferLast(nil)
	suit.Equal(ONE, suit.deque.PollFirst())
}

func (suit *LinkedBlockDequeTestSuite) TestPutFirst() {
	suit.deque.PutFirst(nil)
	suit.deque.PutFirst(ONE)
	suit.deque.PutFirst(TWO)
	suit.Equal(2, suit.deque.Size())
	suit.Equal(TWO, suit.deque.PollFirst())
}

func (suit *LinkedBlockDequeTestSuite) TestPutLast() {
	suit.deque.PutLast(nil)
	suit.deque.PutLast(ONE)
	suit.deque.PutLast(TWO)
	suit.Equal(2, suit.deque.Size())
	suit.Equal(ONE, suit.deque.PollFirst())
}

func (suit *LinkedBlockDequeTestSuite) TestPutLastWait() {
	suit.deque.PutLast(ONE)
	suit.deque.PutLast(TWO)
	wait := sync.WaitGroup{}
	wait.Add(1)
	go func() {
		suit.deque.PutLast(THREE)
		wait.Done()
	}()
	sleep(100)
	suit.Equal(TWO, suit.deque.PollLast())
	wait.Wait()
	suit.Equal(2, suit.deque.Size())
	suit.Equal(THREE, suit.deque.PollLast())
}

func (suit *LinkedBlockDequeTestSuite) TestPutFirstWait() {
	suit.deque.PutFirst(ONE)
	suit.deque.PutFirst(TWO)
	wait := sync.WaitGroup{}
	wait.Add(1)
	go func() {
		suit.deque.PutFirst(THREE)
		wait.Done()
	}()
	sleep(100)
	suit.Equal(TWO, suit.deque.PollFirst())
	wait.Wait()
	suit.Equal(2, suit.deque.Size())
	suit.Equal(THREE, suit.deque.PollFirst())
}

func (suit *LinkedBlockDequeTestSuite) TestPollFirst() {
	assert.Nil(suit.T(), suit.deque.PollFirst())
	assert.True(suit.T(), suit.deque.OfferFirst(ONE))
	assert.True(suit.T(), suit.deque.OfferFirst(TWO))
	suit.Equal(TWO, suit.deque.PollFirst())
}

func (suit *LinkedBlockDequeTestSuite) TestPollLast() {
	assert.Nil(suit.T(), suit.deque.PollLast())
	assert.True(suit.T(), suit.deque.OfferFirst(ONE))
	assert.True(suit.T(), suit.deque.OfferFirst(TWO))
	suit.Equal(ONE, suit.deque.PollLast())
}

func (suit *LinkedBlockDequeTestSuite) TestTakeFirst() {
	assert.True(suit.T(), suit.deque.OfferFirst(ONE))
	assert.True(suit.T(), suit.deque.OfferFirst(TWO))
	suit.Equal(TWO, suit.NoErrorWithResult(suit.deque.TakeFirst()))
}

func (suit *LinkedBlockDequeTestSuite) TestTakeFirstWait() {
	ch := make(chan interface{}, 1)
	go func() {
		o, _ := suit.deque.TakeFirst()
		ch <- o
	}()
	sleep(100)
	suit.True(suit.deque.OfferFirst(ONE))
	o := <-ch
	close(ch)
	suit.Equal(ONE, o)
}

func (suit *LinkedBlockDequeTestSuite) TestTakeLastWait() {
	ch := make(chan interface{}, 1)
	go func() {
		o, _ := suit.deque.TakeLast()
		ch <- o
	}()
	sleep(100)
	suit.True(suit.deque.OfferFirst(ONE))
	o := <-ch
	close(ch)
	suit.Equal(ONE, o)
}

func (suit *LinkedBlockDequeTestSuite) TestTakeLast() {
	assert.True(suit.T(), suit.deque.OfferFirst(ONE))
	assert.True(suit.T(), suit.deque.OfferFirst(TWO))
	suit.Equal(ONE, suit.NoErrorWithResult(suit.deque.TakeLast()))
}

func (suit *LinkedBlockDequeTestSuite) TestRemoveFirstOccurence() {
	suit.deque = NewDeque(3)
	assert.False(suit.T(), suit.deque.RemoveFirstOccurrence(nil))
	assert.False(suit.T(), suit.deque.RemoveFirstOccurrence(ONE))
	suit.deque.AddLast(ONE)
	suit.deque.AddLast(TWO)
	suit.deque.AddLast(ONE)
	assert.True(suit.T(), suit.deque.RemoveFirstOccurrence(ONE))
	assert.True(suit.T(), suit.deque.Size() == 2)
	assert.True(suit.T(), reflect.DeepEqual(suit.deque.ToSlice(), []interface{}{TWO, ONE}))
}

func (suit *LinkedBlockDequeTestSuite) TestRemoveLastOccurence() {
	suit.deque = NewDeque(3)
	assert.False(suit.T(), suit.deque.RemoveLastOccurrence(nil))
	assert.False(suit.T(), suit.deque.RemoveLastOccurrence(ONE))
	suit.deque.AddLast(ONE)
	suit.deque.AddLast(TWO)
	suit.deque.AddLast(ONE)
	assert.True(suit.T(), suit.deque.RemoveLastOccurrence(ONE))
	assert.True(suit.T(), suit.deque.Size() == 2)
	assert.True(suit.T(), reflect.DeepEqual(suit.deque.ToSlice(), []interface{}{ONE, TWO}))
}

func (suit *LinkedBlockDequeTestSuite) TestPeek() {
	suit.deque.AddLast(ONE)
	suit.deque.AddLast(TWO)
	suit.Equal(2, suit.deque.Size())
	suit.Equal(ONE, suit.deque.PeekFirst())
	suit.Equal(TWO, suit.deque.PeekLast())
	suit.Equal(2, suit.deque.Size())
}

func (suit *LinkedBlockDequeTestSuite) TestPollFirstWithTimeout() {
	assert.Nil(suit.T(), suit.deque.PollFirst())
	assert.Nil(suit.T(), suit.NoErrorWithResult(suit.deque.PollFirstWithTimeout(50*time.Millisecond)))
}

func (suit *LinkedBlockDequeTestSuite) TestPollLastWithTimeout() {
	assert.Nil(suit.T(), suit.deque.PollLast())
	assert.Nil(suit.T(), suit.NoErrorWithResult(suit.deque.PollLastWithTimeout(50*time.Millisecond)))
}

func (suit *LinkedBlockDequeTestSuite) TestInterrupt() {
	wait := sync.WaitGroup{}
	wait.Add(2)
	go func() {
		for i := 0; i < 2; i++ {
			time.Sleep(time.Duration(1000) * time.Millisecond)
			suit.deque.InterruptTakeWaiters()
			fmt.Println("TestInterrupt suit.deque.InterruptTakeWaiters")
			wait.Done()
		}
	}()
	for i := 0; i < 2; i++ {
		_, e := suit.deque.TakeFirst()
		_, ok := e.(*InterruptedErr)
		suit.True(ok, "expect InterruptedErr bug get %v", reflect.TypeOf(e))
		suit.NotNil(e.Error())
	}
	wait.Wait()
}

func (suit *LinkedBlockDequeTestSuite) TestIterator() {
	suit.deque.AddLast(ONE)
	suit.deque.AddLast(TWO)
	iterator := suit.deque.Iterator()
	var list []int
	for iterator.HasNext() {
		item := iterator.Next().(int)
		list = append(list, item)
	}
	//fmt.Println("list:",list)
	assert.True(suit.T(), reflect.DeepEqual(list, []int{ONE, TWO}))
}

func (suit *LinkedBlockDequeTestSuite) TestDescendingIterator() {
	suit.deque.AddLast(ONE)
	suit.deque.AddLast(TWO)
	iterator := suit.deque.DescendingIterator()
	var list []int
	for iterator.HasNext() {
		item := iterator.Next().(int)
		list = append(list, item)
	}
	//fmt.Println("list:",list)
	assert.True(suit.T(), reflect.DeepEqual(list, []int{TWO, ONE}))
}

func (suit *LinkedBlockDequeTestSuite) TestIteratorRemove() {
	count := 100
	suit.deque = NewDeque(count)

	for i := 0; i < count; i++ {
		suit.deque.AddFirst(i)
	}
	suit.Equal(count, suit.deque.Size())
	startWait := sync.WaitGroup{}
	startWait.Add(1)

	endWait := sync.WaitGroup{}
	endWait.Add(count + 1)

	counts := make(map[int]concurrent.AtomicInteger, count)
	hasErr := concurrent.AtomicInteger(0)
	for i := 0; i < count; i++ {
		go func(idx int) {
			startWait.Wait()
			iterator := suit.deque.Iterator()
			for iterator.HasNext() {
				item := iterator.Next()
				if item == nil {
					hasErr.IncrementAndGet()
				} else {
					c := counts[idx]
					c.IncrementAndGet()
				}
			}
			endWait.Done()
		}(i)
	}
	go func() {
		startWait.Wait()
		iterator := suit.deque.Iterator()
		c := 0
		for iterator.HasNext() {
			iterator.Next()
			if c%2 == 1 {
				iterator.Remove()
			}
			c = c + 1
		}
		endWait.Done()
	}()
	startWait.Done()
	endWait.Wait()
	iterator := suit.deque.Iterator()
	var list []int
	for iterator.HasNext() {
		item := iterator.Next().(int)
		list = append(list, item)
	}
	//fmt.Println("list:",list)
	//fmt.Println("counts:", counts)
	suit.Equal(count/2, suit.deque.Size())
	suit.Equal(count/2, len(list))
	suit.Equal(int32(0), hasErr.Get())
}

func (suit *LinkedBlockDequeTestSuite) TestQueueLock() {
	suit.deque = NewDeque(1)
	ch := make(chan int)
	go func() {
		ch <- suit.NoErrorWithResult(suit.deque.TakeFirst()).(int)
		fmt.Printf("TestQueueLock take finish.\n")
	}()
	//time.Sleep(time.Duration(1)*time.Second)
	go func() {
		suit.deque.PutFirst(1)
		fmt.Printf("TestQueueLock put finish.\n")
	}()
	val := <-ch
	close(ch)
	suit.Equal(1, val)
}

func (suit *LinkedBlockDequeTestSuite) TestQueueConcurrent() {
	count := 100
	suit.deque = NewDeque(count)
	ch := make(chan int, count)
	for i := 0; i < count; i++ {
		go func() {
			ch <- suit.NoErrorWithResult(suit.deque.TakeFirst()).(int)
		}()
	}
	sleep(100)
	for i := 0; i < count; i++ {
		go func(val int) {
			suit.deque.AddFirst(val)
		}(i)
	}
	values := make([]int, count)
	valueset := make(map[int]int)
	for i := 0; i < count; i++ {
		val := <-ch
		values[i] = val
		valueset[val] = val
	}
	fmt.Println("TestQueueConcurrent", values)
	suit.Equal(count, len(valueset))
	close(ch)
	//suit.Equal(1, val)
}

func currentTimeMillis() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func sleep(millisecond int) {
	time.Sleep(time.Duration(millisecond) * time.Millisecond)
}

func (suit *LinkedBlockDequeTestSuite) TestQueueConcurrentTimeout() {
	suit.deque = NewDeque(10)
	count := 20
	ch := make(chan int, count/2)
	timeoutChan := make(chan int, count/2)
	timeout := 1000
	for i := 0; i < count; i++ {
		go func() {
			startWait := currentTimeMillis()
			r := suit.NoErrorWithResult(suit.deque.PollFirstWithTimeout(time.Duration(timeout) * time.Millisecond))
			endWait := currentTimeMillis()
			//timeout
			if r == nil {
				timeoutChan <- int((endWait - startWait))
			} else {
				ch <- r.(int)
			}
		}()
	}
	for i := 0; i < count/2; i++ {
		go func(val int) {
			sleep(int(rand.Int31n(int32(timeout))))
			suit.deque.AddFirst(val)
		}(i)
	}
	for i := 0; i < count/2; i++ {
		t := <-timeoutChan
		//fmt.Println(t)
		if t < timeout {
			suit.Fail(fmt.Sprintf("%v timeout %v < 1000", t, i))
		}
	}
	close(timeoutChan)
	valueset := make(map[int]int)
	for i := 0; i < count/2; i++ {
		val := <-ch
		valueset[val] = val
	}
	close(ch)
	suit.Equal(count/2, len(valueset))
}

func (suit *LinkedBlockDequeTestSuite) TestQueuePutAndPullTimeout() {
	suit.deque = NewDeque(1)
	ch := make(chan int)
	timeout := time.Duration(500) * time.Millisecond
	go func() {
		ch <- suit.NoErrorWithResult(suit.deque.PollFirstWithTimeout(timeout)).(int)
		fmt.Printf("TestQueueLock take finish.\n")
	}()
	time.Sleep(time.Duration(100) * time.Millisecond)
	go func() {
		suit.deque.PutFirst(1)
		fmt.Printf("TestQueueLock put finish.\n")
	}()
	val := <-ch
	close(ch)
	suit.Equal(1, val)
}

func (suit *LinkedBlockDequeTestSuite) TestHasTakeWaitersWithTimeout() {
	suit.deque = NewDeque(1)
	suit.False(suit.deque.HasTakeWaiters())
	timeout := time.Duration(500) * time.Millisecond
	ch := make(chan int)
	go func() {
		ch <- suit.NoErrorWithResult(suit.deque.PollFirstWithTimeout(timeout)).(int)
		fmt.Printf("TestQueueLock take finish.\n")
	}()
	time.Sleep(time.Duration(50) * time.Millisecond)
	suit.True(suit.deque.HasTakeWaiters())
	go func() {
		suit.deque.PutFirst(1)
		fmt.Printf("TestQueueLock put finish.\n")
	}()
	val := <-ch
	close(ch)
	suit.Equal(1, val)
	suit.False(suit.deque.HasTakeWaiters())
}

func (suit *LinkedBlockDequeTestSuite) TestHasTakeWaiters() {
	suit.deque = NewDeque(1)
	suit.False(suit.deque.HasTakeWaiters())
	ch := make(chan int)
	go func() {
		ch <- suit.NoErrorWithResult(suit.deque.TakeFirst()).(int)
		fmt.Printf("TestQueueLock take finish.\n")
	}()
	time.Sleep(time.Duration(50) * time.Millisecond)
	suit.True(suit.deque.HasTakeWaiters())
	go func() {
		suit.deque.PutFirst(1)
		fmt.Printf("TestQueueLock put finish.\n")
	}()
	val := <-ch
	close(ch)
	suit.Equal(1, val)
	suit.False(suit.deque.HasTakeWaiters())
}
