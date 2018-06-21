package collections

import (
	"context"
	"fmt"
	"math/rand"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/jolestar/go-commons-pool/concurrent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
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
	t.Parallel()

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
	ctx := context.Background()
	suit.deque.PutFirst(ctx, nil)
	suit.deque.PutFirst(ctx, ONE)
	suit.deque.PutFirst(ctx, TWO)
	suit.Equal(2, suit.deque.Size())
	suit.Equal(TWO, suit.deque.PollFirst())
}

func (suit *LinkedBlockDequeTestSuite) TestPutLast() {
	ctx := context.Background()
	suit.deque.PutLast(ctx, nil)
	suit.deque.PutLast(ctx, ONE)
	suit.deque.PutLast(ctx, TWO)
	suit.Equal(2, suit.deque.Size())
	suit.Equal(ONE, suit.deque.PollFirst())
}

func (suit *LinkedBlockDequeTestSuite) TestPutLastWait() {
	ctx := context.Background()
	suit.deque.PutLast(ctx, ONE)
	suit.deque.PutLast(ctx, TWO)
	wait := sync.WaitGroup{}
	wait.Add(1)
	go func() {
		suit.deque.PutLast(ctx, THREE)
		wait.Done()
	}()
	time.Sleep(100 * time.Millisecond)
	suit.Equal(TWO, suit.deque.PollLast())
	wait.Wait()
	suit.Equal(2, suit.deque.Size())
	suit.Equal(THREE, suit.deque.PollLast())
}

func (suit *LinkedBlockDequeTestSuite) TestPutFirstWait() {
	ctx := context.Background()
	suit.deque.PutFirst(ctx, ONE)
	suit.deque.PutFirst(ctx, TWO)
	wait := sync.WaitGroup{}
	wait.Add(1)
	go func() {
		suit.deque.PutFirst(ctx, THREE)
		wait.Done()
	}()
	time.Sleep(100 * time.Millisecond)
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
	ctx := context.Background()
	assert.True(suit.T(), suit.deque.OfferFirst(ONE))
	assert.True(suit.T(), suit.deque.OfferFirst(TWO))
	suit.Equal(TWO, suit.NoErrorWithResult(suit.deque.TakeFirst(ctx)))
}

func (suit *LinkedBlockDequeTestSuite) TestTakeFirstWait() {
	ctx := context.Background()
	ch := make(chan interface{}, 1)
	go func() {
		o, _ := suit.deque.TakeFirst(ctx)
		ch <- o
	}()
	time.Sleep(100 * time.Millisecond)
	suit.True(suit.deque.OfferFirst(ONE))
	o := <-ch
	close(ch)
	suit.Equal(ONE, o)
}

func (suit *LinkedBlockDequeTestSuite) TestTakeLastWait() {
	ctx := context.Background()
	ch := make(chan interface{}, 1)
	go func() {
		o, _ := suit.deque.TakeLast(ctx)
		ch <- o
	}()
	time.Sleep(100 * time.Millisecond)
	suit.True(suit.deque.OfferFirst(ONE))
	o := <-ch
	close(ch)
	suit.Equal(ONE, o)
}

func (suit *LinkedBlockDequeTestSuite) TestTakeLast() {
	ctx := context.Background()
	assert.True(suit.T(), suit.deque.OfferFirst(ONE))
	assert.True(suit.T(), suit.deque.OfferFirst(TWO))
	suit.Equal(ONE, suit.NoErrorWithResult(suit.deque.TakeLast(ctx)))
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

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	assert.Nil(suit.T(), suit.NoErrorWithResult(suit.deque.PollFirstWithContext(ctx)))
}

func (suit *LinkedBlockDequeTestSuite) TestPollLastWithTimeout() {
	assert.Nil(suit.T(), suit.deque.PollLast())

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	assert.Nil(suit.T(), suit.NoErrorWithResult(suit.deque.PollLastWithContext(ctx)))
}

func (suit *LinkedBlockDequeTestSuite) TestInterrupt() {
	ctx := context.Background()
	wait := sync.WaitGroup{}
	wait.Add(2)
	go func() {
		for i := 0; i < 2; i++ {
			time.Sleep(1 * time.Second)
			suit.deque.InterruptTakeWaiters()
			fmt.Println("TestInterrupt suit.deque.InterruptTakeWaiters")
			wait.Done()
		}
	}()
	for i := 0; i < 2; i++ {
		_, e := suit.deque.TakeFirst(ctx)
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
	ctx := context.Background()
	suit.deque = NewDeque(1)
	ch := make(chan int)
	go func() {
		ch <- suit.NoErrorWithResult(suit.deque.TakeFirst(ctx)).(int)
		fmt.Printf("TestQueueLock take finish.\n")
	}()
	//time.Sleep(1*time.Second)
	go func() {
		suit.deque.PutFirst(ctx, 1)
		fmt.Printf("TestQueueLock put finish.\n")
	}()
	val := <-ch
	close(ch)
	suit.Equal(1, val)
}

func (suit *LinkedBlockDequeTestSuite) TestQueueConcurrent() {
	ctx := context.Background()
	count := 100
	suit.deque = NewDeque(count)
	ch := make(chan int, count)
	for i := 0; i < count; i++ {
		go func() {
			ch <- suit.NoErrorWithResult(suit.deque.TakeFirst(ctx)).(int)
		}()
	}
	time.Sleep(100 * time.Millisecond)
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

func (suit *LinkedBlockDequeTestSuite) TestQueueConcurrentTimeout() {
	suit.deque = NewDeque(10)
	count := 20
	ch := make(chan int, count/2)
	timeoutChan := make(chan time.Duration, count/2)
	timeout := 1 * time.Second
	for i := 0; i < count; i++ {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			startWait := time.Now()
			r := suit.NoErrorWithResult(suit.deque.PollFirstWithContext(ctx))
			waitElapsed := time.Since(startWait)
			//timeout
			if r == nil {
				timeoutChan <- waitElapsed
			} else {
				ch <- r.(int)
			}
		}()
	}
	for i := 0; i < count/2; i++ {
		go func(val int) {
			time.Sleep(time.Duration(rand.Int63n(int64(timeout))))
			suit.deque.AddFirst(val)
		}(i)
	}
	for i := 0; i < count/2; i++ {
		t := <-timeoutChan
		//fmt.Println(t)
		if t < timeout {
			suit.Fail(fmt.Sprintf("%v timeout %v < %v", t, i, timeout))
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
	ctx := context.Background()
	suit.deque = NewDeque(1)
	ch := make(chan int)
	timeout := 500 * time.Millisecond
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		ch <- suit.NoErrorWithResult(suit.deque.PollFirstWithContext(ctx)).(int)
		fmt.Printf("TestQueueLock take finish.\n")
	}()
	time.Sleep(100 * time.Millisecond)
	go func() {
		suit.deque.PutFirst(ctx, 1)
		fmt.Printf("TestQueueLock put finish.\n")
	}()
	val := <-ch
	close(ch)
	suit.Equal(1, val)
}

func (suit *LinkedBlockDequeTestSuite) TestHasTakeWaitersWithTimeout() {
	ctx := context.Background()
	suit.deque = NewDeque(1)
	suit.False(suit.deque.HasTakeWaiters())
	timeout := 500 * time.Millisecond
	ch := make(chan int)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		ch <- suit.NoErrorWithResult(suit.deque.PollFirstWithContext(ctx)).(int)
		fmt.Printf("TestQueueLock take finish.\n")
	}()
	time.Sleep(50 * time.Millisecond)
	suit.True(suit.deque.HasTakeWaiters())
	go func() {
		suit.deque.PutFirst(ctx, 1)
		fmt.Printf("TestQueueLock put finish.\n")
	}()
	val := <-ch
	close(ch)
	suit.Equal(1, val)
	suit.False(suit.deque.HasTakeWaiters())
}

func (suit *LinkedBlockDequeTestSuite) TestHasTakeWaiters() {
	ctx := context.Background()
	suit.deque = NewDeque(1)
	suit.False(suit.deque.HasTakeWaiters())
	ch := make(chan int)
	go func() {
		ch <- suit.NoErrorWithResult(suit.deque.TakeFirst(ctx)).(int)
		fmt.Printf("TestQueueLock take finish.\n")
	}()
	time.Sleep(50 * time.Millisecond)
	suit.True(suit.deque.HasTakeWaiters())
	go func() {
		suit.deque.PutFirst(ctx, 1)
		fmt.Printf("TestQueueLock put finish.\n")
	}()
	val := <-ch
	close(ch)
	suit.Equal(1, val)
	suit.False(suit.deque.HasTakeWaiters())
}
